// internal/platform/windows/platform.go
package windows

import "file-maintenance/internal/platform/windows/setup"

// Platform implements platform.Platform for Windows.
//
// Windows owns the interactive setup wizard because the wizard depends on
// PowerShell and System.Windows.Forms.
type Platform struct{}

// EnsureConfig verifies that config.ini exists, launching the embedded Windows
// setup wizard when the file is missing.
func (Platform) EnsureConfig(configDir string, exeDir string) (bool, error) {
	return setup.EnsureConfig(configDir, exeDir)
}
