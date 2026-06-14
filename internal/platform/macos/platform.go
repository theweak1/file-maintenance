// internal/platform/macos/platform.go
package macos

import (
	"os"
	"path/filepath"
)

// Platform implements platform.Platform for macOS.
//
// This implementation intentionally does not launch a GUI setup wizard. If
// config.ini is missing, EnsureConfig returns false so the application exits
// without processing files.
type Platform struct{}

// EnsureConfig verifies that config.ini exists in configDir.
//
// Unlike Windows, macOS does not currently provide an interactive setup wizard.
// Returning false for a missing config keeps startup fail-safe and avoids any
// deletion work before the user has explicitly configured the tool.
func (Platform) EnsureConfig(configDir string, exeDir string) (bool, error) {
	configFile := filepath.Join(configDir, "config.ini")
	_, err := os.Stat(configFile)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
