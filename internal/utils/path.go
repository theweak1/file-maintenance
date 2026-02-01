package utils

import (
	"os"
	"path/filepath"
)

// ExeDir returns the directory containing the currently running executable.
//
// Why this exists:
//   - Scheduled tasks (e.g., Windows Task Scheduler) often run with an unexpected
//     working directory such as C:\Windows\System32.
//   - Relying on os.Getwd() alone can cause config/log paths to resolve incorrectly.
//   - Using the executable’s location makes the app self-contained and predictable.
//
// Behavior:
// - Uses os.Executable() to obtain the full path to the running binary
// - Resolves symlinks (important when launched via shortcuts, symlinks, or wrappers)
// - Returns the parent directory of the executable
//
// Errors:
// - Returns an error if the executable path cannot be resolved
// - Callers may safely fall back to os.Getwd() if this fails
//
// Typical usage:
//
//	root, err := utils.ExeDir()
//	if err != nil {
//	    root, _ = os.Getwd()
//	}
//	defaultConfigDir := filepath.Join(root, "configs")
//	defaultLogDir    := filepath.Join(root, "logs")
func ExeDir() (string, error) {
	// Get the absolute path to the currently running executable.
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}

	// Resolve any symlinks to get the real on-disk location.
	// This avoids surprises when the binary is invoked via a shortcut.
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}

	// Return the directory containing the executable.
	return filepath.Dir(exe), nil
}
