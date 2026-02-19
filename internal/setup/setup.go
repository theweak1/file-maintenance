package setup

import (
	"fmt"
	"os"
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
	return LaunchEmbeddedSetup(configDir, exeDir)
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
