package setup

import (
	"fmt"
	"os"
	"path/filepath"

	"file-maintenance/internal/types"
)

const (
	setupExitSaved       = 0
	setupExitCancelled   = 1
	setupExitSavedAndRun = 2
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

// SetupActionFromExitCode maps setup.ps1 exit codes to app-level setup actions.
func SetupActionFromExitCode(exitCode int) (types.SetupAction, bool) {
	switch exitCode {
	case setupExitSaved:
		return types.SetupActionSaved, true
	case setupExitCancelled:
		return types.SetupActionCancelled, true
	case setupExitSavedAndRun:
		return types.SetupActionSavedAndRun, true
	default:
		return types.SetupActionCancelled, false
	}
}

// LaunchSetupWizard launches the PowerShell GUI setup wizard.
//
// The setup wizard will:
// - Guide the user through configuring backup location, paths to clean, and other settings
// - Create the config.ini file in the specified config directory
// - Return whether the user selected Cancel, Save & Close, or Save & Run
func LaunchSetupWizard(configDir, exeDir string) (types.SetupAction, error) {
	return LaunchEmbeddedSetup(configDir, exeDir)
}

// RunSetup launches the setup wizard regardless of whether config.ini already exists.
func RunSetup(configDir, exeDir string) (types.SetupAction, error) {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return types.SetupActionCancelled, fmt.Errorf("failed to create config directory: %w", err)
	}

	fmt.Println("Opening setup wizard...")
	return LaunchSetupWizard(configDir, exeDir)
}

// EnsureConfig checks if configuration exists and launches the setup wizard if not.
//
// This function is kept for compatibility. The current main flow opens setup by
// default and requires -run for background maintenance.
func EnsureConfig(configDir, exeDir string) (bool, error) {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return false, fmt.Errorf("failed to create config directory: %w", err)
	}

	if ConfigExists(configDir) {
		return true, nil
	}

	fmt.Println("No configuration found. Launching setup wizard...")
	action, err := LaunchSetupWizard(configDir, exeDir)
	if err != nil {
		return false, err
	}
	if action == types.SetupActionCancelled {
		return false, nil
	}

	return ConfigExists(configDir), nil
}

// GetConfigPath returns the full path to the config.ini file.
func GetConfigPath(configDir string) string {
	return filepath.Join(configDir, "config.ini")
}

// GetDefaultConfigDir returns the default config directory based on the executable location.
func GetDefaultConfigDir(exeDir string) string {
	return filepath.Join(exeDir, "config")
}
