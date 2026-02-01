package maintenance

import (
	"errors"
	"path/filepath"
	"strings"
)

// ErrPathEscapesRoot is returned when a source path is not safely contained
// within the configured source root.
//
// This is a hard safety error: callers must not attempt to back up or delete
// files when this error is returned.
var ErrPathEscapesRoot = errors.New("source path escapes srcRoot")

// backupDestPath builds the destination path for a backup file,
// preserving the directory structure relative to the source root.
//
// Example:
//
//	backupRoot = "Z:\\Backups"
//	srcRoot    = "C:\\Data"
//	srcFull    = "C:\\Data\\Images\\2026\\img.jpg"
//
// Result:
//
//	Z:\Backups\Images\2026\img.jpg
//
// Design goals:
// - Preserve the relative directory structure of each configured source folder.
// - Reject any path that would escape srcRoot (path traversal or wrong root).
// - Be safe on Windows paths and network shares.
//
// Safety contract (IMPORTANT):
//   - If srcFull is not under srcRoot, this function returns ErrPathEscapesRoot.
//   - Cleaned paths like "A\\..\\B\\file.txt" are allowed as long as the final
//     resolved path remains under srcRoot.
//
// This function does NOT:
// - Create directories.
// - Check whether the destination exists.
// - Perform any file I/O.
//
// Those responsibilities belong to the copy layer.
func backupDestPath(backupRoot, srcRoot, srcFull string) (string, error) {
	// Compute the relative path from the source root to the file.
	// Example:
	//   srcRoot = C:\Data
	//   srcFull = C:\Data\Images\file.jpg
	//   rel     = Images\file.jpg
	rel, err := filepath.Rel(srcRoot, srcFull)
	if err != nil {
		return "", err
	}

	// Normalize the relative path.
	// This collapses things like "A\..\B" to "B".
	rel = filepath.Clean(rel)

	// Safety check:
	// If rel is ".." or starts with "..\" (or "../" on non-Windows),
	// then srcFull is outside srcRoot or attempted to escape it.
	//
	// Rejecting this prevents writing files outside the backup root.
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrPathEscapesRoot
	}

	// Join backup root with the validated relative path.
	return filepath.Join(backupRoot, rel), nil
}
