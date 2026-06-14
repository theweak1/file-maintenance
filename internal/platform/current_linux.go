// internal/platform/current_linux.go
//go:build linux

package platform

import "file-maintenance/internal/platform/linux"

var _ Platform = linux.Platform{}

func Current() Platform {
	return linux.Platform{}
}
