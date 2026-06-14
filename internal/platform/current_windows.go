// internal/platform/current_windows.go
//go:build windows

package platform

import "file-maintenance/internal/platform/windows"

var _ Platform = windows.Platform{}

func Current() Platform {
	return windows.Platform{}
}
