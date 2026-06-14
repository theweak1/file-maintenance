package maintenance

import (
	"fmt"
	"os"
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
