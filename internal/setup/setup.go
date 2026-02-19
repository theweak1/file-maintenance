package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ConfigExists checks if the configuration file exists in the given config directory.
//
// Returns:
//   - true if config.ini exists
//   - false if config.ini does not exist
func ConfigExists(configDir string) bool {
	configFile := filepath.Join(configDir, "config.ini")
	_, err := os.Stat(configFile)
	return err == nil
}

// LaunchSetupWizard launches the PowerShell GUI setup wizard.
//
// The setup wizard will:
// - Guide the user through configuring backup location, paths to clean, and other settings
// - Create the config.ini file in the specified config directory
//
// Parameters:
//   - configDir: The directory where config.ini will be created
//   - exeDir: The directory containing the running executable (for locating setup.ps1)
//
// Returns:
//   - error if the setup wizard fails to launch
func LaunchSetupWizard(configDir, exeDir string) error {
	// First, try to use the embedded script
	// Note: If user cancels, the script exits with code 1 - we treat that as "user cancelled" and don't fall through
	if err := LaunchEmbeddedSetup(configDir, exeDir); err != nil {
		// Check if it's a user cancellation (exit code 1) vs a real error
		// If the error contains "exit status 1", user likely cancelled - don't try fallback
		if err.Error() != "failed to launch setup wizard: exit status 1" {
			// Try fallback to external script
			if fallbackErr := launchExternalSetup(configDir, exeDir); fallbackErr == nil {
				return nil
			}
		}
		return err
	}
	return nil
}

// launchExternalSetup tries to find and run an external setup.ps1 file
func launchExternalSetup(configDir, exeDir string) error {
	setupPaths := []string{
		filepath.Join(exeDir, "config", "setup.ps1"),
		filepath.Join(exeDir, "setup.ps1"),
		filepath.Join(configDir, "setup.ps1"),
		"config/setup.ps1",
		"../config/setup.ps1",
		filepath.Join(".", "config", "setup.ps1"),
	}

	var setupScript string
	for _, path := range setupPaths {
		absolutePath := path
		if !filepath.IsAbs(path) {
			if cwd, err := os.Getwd(); err == nil {
				absolutePath = filepath.Join(cwd, path)
			}
		}
		if _, err := os.Stat(absolutePath); err == nil {
			setupScript = absolutePath
			break
		}
	}

	if setupScript == "" {
		return fmt.Errorf("setup.ps1 not found")
	}

	cmd := exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-File", setupScript, "-ConfigDir", configDir)
	scriptDir := filepath.Dir(setupScript)
	cmd.Dir = scriptDir
	cmd.Env = os.Environ()

	return cmd.Run()
}

// EnsureConfig checks if configuration exists and launches the setup wizard if not.
//
// Parameters:
//   - configDir: The directory where config.ini should be located
//   - exeDir: The directory containing the running executable
//
// Returns:
//   - true if configuration now exists (either it did, or setup was completed successfully)
//   - false if setup was cancelled or failed
//   - error if there was an error checking or creating configuration
func EnsureConfig(configDir, exeDir string) (bool, error) {
	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return false, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Check if config already exists
	if ConfigExists(configDir) {
		return true, nil
	}

	// Config doesn't exist, launch setup wizard
	fmt.Println("No configuration found. Launching setup wizard...")
	fmt.Println("Please configure your settings in the GUI window.")

	if err := LaunchSetupWizard(configDir, exeDir); err != nil {
		return false, err
	}

	// Check if config was created
	if ConfigExists(configDir) {
		return true, nil
	}

	return false, nil
}

// GetConfigPath returns the full path to the config.ini file.
func GetConfigPath(configDir string) string {
	return filepath.Join(configDir, "config.ini")
}

// GetDefaultConfigDir returns the default config directory based on the executable location.
func GetDefaultConfigDir(exeDir string) string {
	return filepath.Join(exeDir, "config")
}
