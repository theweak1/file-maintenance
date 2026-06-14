// internal/platform/windows/path.go
package windows

import (
	"path/filepath"
	"strings"
)

func (Platform) SamePath(a, b string) bool {
	pa, err1 := filepath.Abs(a)
	pb, err2 := filepath.Abs(b)

	if err1 != nil || err2 != nil {
		return false
	}

	return strings.EqualFold(filepath.Clean(pa), filepath.Clean(pb))
}
