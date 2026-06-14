package macos

import "path/filepath"

func (Platform) SamePath(a, b string) bool {
	pa, err1 := filepath.Abs(a)
	pb, err2 := filepath.Abs(b)
	if err1 != nil || err2 != nil {
		return false
	}

	return filepath.Clean(pa) == filepath.Clean(pb)
}
