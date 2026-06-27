// internal/platform/linux/platform.go
package linux

import (
	"fmt"
	"os"
	"path/filepath"

	"file-maintenance/internal/types"
)

// Platform implements platform.Platform for Linux.
//
// This implementation intentionally does not launch a GUI setup wizard. If
// config.ini is missing, EnsureConfig returns false so the application exits
// without processing files.
type Platform struct{}

// EnsureConfig verifies that config.ini exists in configDir.
//
// Unlike Windows, Linux does not currently provide an interactive setup wizard.
// Returning false for a missing config keeps startup fail-safe and avoids any
// deletion work before the user has explicitly configured the tool.

// RunSetup is not implemented on Linux. Returning SetupActionCancelled keeps
// default startup safe on platforms without a GUI configuration flow.
func (Platform) RunSetup(configDir string, exeDir string) (types.SetupAction, error) {
	return types.SetupActionCancelled, fmt.Errorf("setup wizard is not implemented for this platform")
}

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

func (Platform) AvailableBytes(path string) (uint64, error) {
	return 0, fmt.Errorf("disk space check not implemented for this platform")
}
