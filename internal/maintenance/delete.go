package maintenance

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"file-maintenance/internal/logging"
)

// DeleteFile removes a single file from disk.
//
// Contract:
// - Performs a hard delete (no recycle bin).
// - Callers must ensure any required backup has already completed successfully.
// - Errors are wrapped so higher layers can decide whether to abort the run.
//
// Why this is a separate helper:
//   - Keeps Worker() logic readable.
//   - Centralizes delete behavior if future changes are needed
//     (e.g., retries, logging, or a dry-run mode).
func DeleteFile(srcPath string) error {
	if err := os.Remove(srcPath); err != nil {
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}

// cleanupEmptyDirs removes empty directories starting from startDir and moving
// upward toward stopDir (but never beyond it).
//
// Behavior:
// - Works bottom-up: removes startDir if empty, then checks its parent, etc.
// - Stops immediately if:
//   - a directory is not empty
//   - an error occurs (permissions, transient SMB issue, etc.)
//   - stopDir is reached
//
// Safety invariants (VERY IMPORTANT):
// - stopDir acts as a hard boundary; directories above it are never removed.
// - samePath() is used to safely compare paths on Windows (case-insensitive).
//
// Conservative-by-design:
// - If anything unexpected happens, this function stops quietly.
// - No errors are returned to avoid risking removal of unintended directories.
func cleanupEmptyDirs(startDir, stopDir string, log *logging.Logger) {
	cur := startDir

	for {
		// Stop once we reach the configured folder root.
		// This prevents deleting directories outside the scope of this run.
		if samePath(cur, stopDir) {
			return
		}

		// Check whether the directory is empty (non-recursive).
		empty, err := isDirEmpty(cur)
		if err != nil {
			// If we can't read the directory, do not attempt deletion.
			return
		}
		if !empty {
			// As soon as we find a non-empty directory, stop.
			return
		}

		// Attempt to remove the empty directory.
		// On Windows, this will fail if:
		// - the directory is not empty
		// - another process has it open
		// - permissions are insufficient
		if err := os.Remove(cur); err != nil {
			return
		}

		log.Infof("Removed empty directory: %s", cur)

		// Move one level up and repeat.
		cur = filepath.Dir(cur)
	}
}

// samePath compares two filesystem paths for equality in a Windows-safe way.
//
// Behavior:
// - Converts both paths to absolute paths.
// - Compares them case-insensitively.
//
// Why this matters:
// - Windows paths are case-insensitive.
// - Paths like "C:\Data\Folder" and "c:\data\folder" should be treated as equal.
//
// Failure mode:
// - If absolute path resolution fails for either input, this returns false.
// - Callers should treat false as "not equal" and behave conservatively.
func samePath(a, b string) bool {
	pa, err1 := filepath.Abs(a)
	pb, err2 := filepath.Abs(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return strings.EqualFold(pa, pb)
}
