// internal/platform/current_darwin.go
//go:build darwin

package platform

import "file-maintenance/internal/platform/macos"

var _ Platform = macos.Platform{}

func Current() Platform {
	return macos.Platform{}
}
