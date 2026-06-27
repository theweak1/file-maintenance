// internal/platform/windows/platform.go
package windows

import (
	"file-maintenance/internal/platform/windows/setup"
	"file-maintenance/internal/types"
)

// Platform implements platform.Platform for Windows.
//
// Windows owns the interactive setup wizard because the wizard depends on
// PowerShell and System.Windows.Forms.
type Platform struct{}

// RunSetup launches the embedded Windows setup wizard and returns the action
// selected by the user: cancel, save, or save and run.
func (Platform) RunSetup(configDir string, exeDir string) (types.SetupAction, error) {
	return setup.RunSetup(configDir, exeDir)
}

// EnsureConfig verifies that config.ini exists, launching the embedded Windows
// setup wizard when the file is missing. This method is kept for compatibility;
// main uses RunSetup for setup-first mode and performs a non-interactive config
// existence check before -run maintenance.
func (Platform) EnsureConfig(configDir string, exeDir string) (bool, error) {
	return setup.EnsureConfig(configDir, exeDir)
}
