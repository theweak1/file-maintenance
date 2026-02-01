package maintenance

import (
	"os"
	"path/filepath"
	"time"
)

// IsFileOlder determines whether a file should be considered "expired"
// based on its modification time.
//
// Behavior:
// - Uses the file's ModTime().
// - Computes cutoff := time.Now().AddDate(0, 0, -days).
// - Returns true only if ModTime is strictly before the cutoff (strict comparison).
//
// Notes:
// - We intentionally use ModTime() instead of access time because:
//   - Access time may be disabled on some filesystems.
//   - ModTime is more predictable across platforms.
//
// - The strict comparison matters:
//   - If a file is exactly at the cutoff timestamp, this returns false.
//   - Tests should avoid asserting the equality boundary because time.Now() is dynamic.
//
//   - days == 0 means "older than now" which will match almost all existing files,
//     except files with future timestamps.
func IsFileOlder(info os.FileInfo, days int) bool {
	cutoff := time.Now().AddDate(0, 0, -days)
	return info.ModTime().Before(cutoff)
}

// DoesFileExist checks whether a file or directory exists at the given path.
//
// Behavior:
// - Returns true if os.Stat succeeds.
// - Returns false only if the error is os.IsNotExist.
// - Returns true for other errors (e.g., permission issues).
//
// Why:
//   - For backup logic, we treat "exists but inaccessible" as "exists" to avoid
//     overwriting or clobbering unexpected paths.
func DoesFileExist(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return !os.IsNotExist(err)
}

// CheckBackupPath validates that the backup destination is safe to use.
//
// This function is intentionally conservative because failures here can
// lead to data loss if files are deleted without a successful backup.
//
// Validation steps:
// 1. Clean the path (removes trailing separators, collapses "..", etc.).
// 2. Ensure the path exists.
// 3. Ensure it is a directory.
// 4. Attempt to create and delete a temporary file inside the directory.
//
// If ANY step fails, the backup path is considered invalid.
//
// Notes (important):
// - The temp-file test is a best-effort writability check, especially useful for SMB shares:
//   - A directory may exist but be read-only.
//   - Credentials may be expired.
//   - A share may be reachable but not writable at runtime.
//
// - Even if this returns true, later writes can still fail (network volatility).
// - This does not validate free space, quota, or long-term availability.
//
// - The temp file is removed immediately.
func CheckBackupPath(backupRoot string) bool {
	backupRoot = filepath.Clean(backupRoot)

	// Ensure path exists and is a directory.
	info, err := os.Stat(backupRoot)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		return false
	}

	// Attempt a real write test inside the directory.
	f, err := os.CreateTemp(backupRoot, ".backup_test_*")
	if err != nil {
		return false
	}

	// Clean up the temporary file.
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)

	return true
}

// isDirEmpty reports whether a directory contains zero entries.
//
// Behavior:
// - Reads only the immediate directory entries (non-recursive).
// - Returns true if the directory has no files or subdirectories.
//
// Why:
//   - Used after deleting a file to determine whether its parent directory can be removed.
//   - Using os.ReadDir is fast and avoids expensive recursive scans,
//     which is especially important on network filesystems.
//
// Errors:
//   - If the directory cannot be read (permissions, missing, etc.),
//     the error is returned to the caller so the caller can behave conservatively.
func isDirEmpty(dirPath string) (bool, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}
