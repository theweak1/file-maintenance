package maintenance

import (
	"fmt"
	"os"
	"path/filepath"
)

// RemoveOldLogs deletes log files older than `days` inside logPath.
//
// Behavior:
// - Operates only on files in the top-level of logPath (non-recursive).
// - Skips subdirectories.
// - Best-effort per file: continues on per-file errors (locked files, permission issues, etc.).
//
// Error behavior:
//   - Returns an error only for "environment/config" failures (e.g., logPath is not a directory,
//     cannot read logPath entries, or cannot create logPath when missing).
//   - Does not return an error just because a particular log file couldn't be deleted.
//
// Safety / design notes:
// - Intended to be called only when file logging is enabled (i.e., -no-logs is NOT set).
// - Conservative: never deletes anything outside logPath and does not recurse.
//
// Note on days:
//   - days is interpreted using IsFileOlder (strictly "before cutoff").
//     If days <= 0, cutoff is "now" or later, so most existing log files qualify as old.
func RemoveOldLogs(logPath string, days int) error {
	// -----------------------------------------------------------------------------
	// Validate logPath
	//
	// We expect logPath to:
	// - exist (normal case), OR
	// - be creatable (first run / fresh install).
	//
	// IMPORTANT:
	// - If os.Stat fails, we must not use `info`.
	// - If the folder doesn't exist yet, creating it is sufficient; there are no
	//   logs to clean up in that case.
	// -----------------------------------------------------------------------------
	info, err := os.Stat(logPath)
	if err != nil {
		// If logPath can't be stat'ed (missing or otherwise), attempt to create it.
		// If creation succeeds, there is nothing to prune yet.
		if err := os.MkdirAll(logPath, 0o755); err != nil {
			return fmt.Errorf("create log path: %w", err)
		}
		return nil
	}

	// If the path exists but is not a directory, this is a configuration error.
	if !info.IsDir() {
		return fmt.Errorf("log path is not a directory: %s", logPath)
	}

	// -----------------------------------------------------------------------------
	// Read directory entries (non-recursive).
	//
	// We intentionally do NOT recurse:
	// - Log files are expected to be flat under logPath.
	// - Recursive deletion increases risk unnecessarily.
	// -----------------------------------------------------------------------------
	entries, err := os.ReadDir(logPath)
	if err != nil {
		return fmt.Errorf("read log folder contents: %w", err)
	}

	for _, entry := range entries {
		// Skip subdirectories entirely.
		if entry.IsDir() {
			continue
		}

		full := filepath.Join(logPath, entry.Name())

		// entry.Info() avoids an extra os.Stat() in many cases.
		fi, err := entry.Info()
		if err != nil {
			// If we cannot read file info (locked, permissions, race),
			// skip this file and continue.
			continue
		}

		// Use the same age logic as file cleanup (shared helper).
		if IsFileOlder(fi, days) {
			// Attempt deletion.
			// If the file is locked (common on Windows), skip it.
			if err := os.Remove(full); err != nil {
				continue
			}
		}
	}

	return nil
}
