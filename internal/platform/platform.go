// Package platform defines the boundary between application logic and
// operating-system-specific behavior.
//
// The core application should depend on this package's Platform interface rather
// than importing Windows, Linux, or macOS implementations directly. That keeps
// setup, notifications, and optional OS-specific path conventions isolated from
// the maintenance worker and application orchestration code.
package platform

import "file-maintenance/internal/types"

// Platform describes the OS-specific behavior required by the application.
//
// Current responsibilities:
// - ShowCritical displays a user-visible critical notification.
// - DefaultConfigDir returns an OS-conventional config directory.
// - DefaultLogDir returns an OS-conventional log/cache directory.
// - RunSetup opens the setup/configuration experience when available.
// - EnsureConfig verifies config.ini exists before maintenance begins.
// - AvailableBytes returns writable bytes available at a destination path.
//
// Note: main currently chooses portable defaults (<exe>/config and <exe>/logs)
// instead of DefaultConfigDir and DefaultLogDir, but these methods remain part of
// the abstraction for deployments that prefer OS-conventional paths.
type Platform interface {
	ShowCritical(title, message string)
	DefaultConfigDir(appName string) (string, error)
	DefaultLogDir(appName string) (string, error)
	RunSetup(configDir string, exeDir string) (types.SetupAction, error)
	EnsureConfig(configDir string, exeDir string) (bool, error)

	AvailableBytes(path string) (uint64, error)
}
