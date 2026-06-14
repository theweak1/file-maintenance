//go:build windows

package windows

import "golang.org/x/sys/windows"

type DiskUsage struct {
	Total     uint64
	Free      uint64
	Available uint64
}

func DiskUsageForPath(path string) (DiskUsage, error) {
	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64

	err := windows.GetDiskFreeSpaceEx(
		windows.StringToUTF16Ptr(path),
		&freeBytesAvailable,
		&totalNumberOfBytes,
		&totalNumberOfFreeBytes,
	)
	if err != nil {
		return DiskUsage{}, err
	}

	return DiskUsage{
		Total:     totalNumberOfBytes,
		Free:      totalNumberOfFreeBytes,
		Available: freeBytesAvailable,
	}, nil
}
