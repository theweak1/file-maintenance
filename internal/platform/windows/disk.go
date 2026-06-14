//go:build windows

package windows

import "golang.org/x/sys/windows"

type DiskUsage struct {
	Total     uint64
	Free      uint64
	Available uint64
}

func (Platform) AvailableBytes(path string) (uint64, error) {
	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64

	err := windows.GetDiskFreeSpaceEx(
		windows.StringToUTF16Ptr(path),
		&freeBytesAvailable,
		&totalNumberOfBytes,
		&totalNumberOfFreeBytes,
	)
	if err != nil {
		return 0, err
	}

	return freeBytesAvailable, nil
}
