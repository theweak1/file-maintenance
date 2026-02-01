package maintenance

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"file-maintenance/internal/logging"
)

// copyFileWithRetry copies a file from srcPath to dstPath, retrying on failure
// using a small, capped backoff strategy.
//
// Why retries exist (especially for Windows / SMB / network shares):
// - Transient network hiccups can cause short-lived failures.
// - Antivirus scanners may temporarily lock files.
// - The destination share may be reachable but momentarily busy.
//
// Behavior:
// - Attempts the copy up to (retries + 1) total times.
// - Uses a small backoff between attempts to avoid hammering the destination.
// - Honors context cancellation so maintenance runs can stop cleanly.
//
// Assumptions / contract:
//   - The caller has already decided it is safe to copy this file.
//   - The caller must ensure dstPath does not already exist (no overwrite semantics).
//   - This function will create/overwrite a temporary file (dstPath + ".tmp") during the copy,
//     but the final destination should not exist.
func copyFileWithRetry(ctx context.Context, srcPath, dstPath string, retries int, log *logging.Logger) error {
	var lastErr error

	for attempt := 0; attempt <= retries; attempt++ {
		// Allow hard cancellation (max runtime reached, shutdown, etc.)
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Attempt a streaming copy (low memory usage, safe for large files).
		if err := copyfileStream(srcPath, dstPath); err != nil {
			lastErr = err

			// Backoff pattern: 250ms → 1s → 3s
			// Keeps retries responsive without stalling the whole run.
			backoff := backoffForAttempt(attempt)

			if attempt < retries {
				log.Warnf(
					"Copy failed (attempt %d/%d) for %s: %v. Retrying in %s...",
					attempt+1, retries+1, srcPath, err, backoff,
				)

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(backoff):
				}
				continue
			}

			// No retries remaining.
			break
		}

		// Copy succeeded.
		return nil
	}

	// All attempts failed.
	return fmt.Errorf("copy failed after %d attempts: %w", retries+1, lastErr)
}

// backoffForAttempt returns the wait duration before retrying a failed copy.
//
// Design notes:
// - Backoff is intentionally small and capped.
// - Maintenance runs should recover from brief glitches quickly.
// - We avoid long exponential backoffs that could stall a run for minutes.
func backoffForAttempt(attempt int) time.Duration {
	switch attempt {
	case 0:
		return 250 * time.Millisecond
	case 1:
		return 1 * time.Second
	default:
		return 3 * time.Second
	}
}

// copyfileStream performs a safe, low-memory, streaming copy from srcPath to dstPath.
//
// Key design goals:
// - Low memory usage (streaming buffer).
// - Avoid leaving partially copied files at the destination.
// - Behave reliably on Windows and SMB/network shares.
//
// Implementation details:
// - Ensures destination directory structure exists (MkdirAll).
// - Writes into a temporary file (dstPath + ".tmp").
// - Closes the file handle before renaming (required on Windows).
// - Renames temp → final path for safer "atomic-ish" behavior.
//
// Safety notes:
//   - os.Rename is not guaranteed fully atomic on all filesystems, especially network shares,
//     but it is still much safer than writing directly to the final filename.
//   - Rename behavior when dstPath already exists is platform/filesystem dependent.
//     Callers should treat dstPath as "must not exist" and check before copying.
//   - If anything fails, the temporary file is cleaned up.
func copyfileStream(srcPath, dstPath string) error {
	// Ensure destination directory exists (recreates relative folder structure).
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}

	// Open source file for reading.
	in, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer in.Close()

	// Write to a temporary file first to avoid partial backups.
	tmp := dstPath + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	// Ensure output is closed and temp file removed on failure.
	closeOK := false
	defer func() {
		_ = out.Close()
		if !closeOK {
			_ = os.Remove(tmp)
		}
	}()

	// Streaming buffer:
	// - 256KB balances memory usage and throughput well.
	buf := make([]byte, 256*1024)
	if _, err := io.CopyBuffer(out, in, buf); err != nil {
		return err
	}

	// Close before rename (Windows requires the handle to be closed).
	if err := out.Close(); err != nil {
		return err
	}
	closeOK = true

	// Finalize copy by renaming temp → destination.
	// Caller is responsible for ensuring dstPath does not exist.
	return os.Rename(tmp, dstPath)
}

// buildBackupPath constructs the final destination path for a backup file.
//
// Resulting structure:
//
//	backupRoot/
//	  └── DDMmmYY/
//	      └── <path relative to the configured folder root>/
//	          └── file.ext
//
// Responsibilities:
// - Create a date-based top-level folder per run/day (DDMmmYY).
// - Preserve the relative directory structure from the configured folder root.
// - Produce a deterministic destination path for logging, retries, and restores.
//
// Notes:
//   - This function does NOT touch the filesystem.
//   - Directory creation is handled later by copyfileStream via MkdirAll.
//   - This helper does not enforce that srcPath is under folder; it relies on the caller
//     to provide a srcPath discovered under that folder (e.g., via walking the folder).
//     If callers need explicit "must be under folder" safety, enforce it at a higher level.
func buildBackupPath(backupRoot, folder, srcPath string) (string, error) {
	// Format date folder (e.g. 30Jan26).
	dateFolder := time.Now().Format("02Jan06")

	// Compute path relative to the configured folder root.
	relPath, err := filepath.Rel(folder, srcPath)
	if err != nil {
		return "", err
	}

	return filepath.Join(
		backupRoot,
		dateFolder,
		relPath,
	), nil
}
