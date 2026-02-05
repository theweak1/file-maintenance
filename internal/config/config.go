package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadFolderList reads the list of paths to process from `folders.txt`.
//
// Contract:
// - One path per line (can be a file OR a folder)
// - Empty lines are ignored
// - Lines starting with '#' are treated as comments
//
// This allows operators to temporarily disable folders or add notes
// without changing code or redeploying the binary.
// Path handling:
// - Folders: All files inside the folder (recursively) that meet age criteria are processed.
// - Files: The individual file is processed directly if it meets age criteria.
//
// This allows operators to:
// - Delete all old files from a folder (just specify the folder path)
// - Delete specific files (just specify the full file path)
// - Temporarily disable paths or add notes without changing code
//
// Example folders.txt:
//
//	# Delete all old files from these folders
//	C:\temp\old
//
//	# Network share
//	\\server\share\incoming
//
//	# Delete specific files
//	C:\Data\Images\old-photo.jpg
//	C:\Logs\debug.log
//
// Errors:
//   - Returns an error if folders.txt cannot be read.
//   - No validation of path existence is performed here; that is deferred
//     to later stages so configuration errors fail fast and explicitly.
func ReadFolderList(configDir string) ([]string, error) {
	// Read the entire file at once; this file is expected to be small.
	b, err := os.ReadFile(filepath.Join(configDir, "folders.txt"))
	if err != nil {
		return nil, fmt.Errorf("read folders.txt: %w", err)
	}

	// Split by newline and normalize each entry.
	lines := strings.Split(string(b), "\n")
	folders := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		folders = append(folders, line)
	}

	return folders, nil
}

// ReadBackupLocation reads the backup destination path from `backup.txt`.
//
// Contract:
// - If backup.txt contains a non-empty path, that path is returned as-is.
// - If backup.txt exists but is empty or whitespace-only, a default path is used.
// - The default backup location is "../backups" relative to the config directory.
//
// This design allows:
// - Easy redirection to a network share (UNC path).
// - A sane local fallback for development, testing, or first-time installs.
//
// Safety:
//   - This function does NOT validate existence or permissions.
//   - Accessibility and directory checks are enforced later by
//     maintenance.CheckBackupPath() before any files are deleted.
func ReadBackupLocation(configDir string) (string, error) {
	b, err := os.ReadFile(filepath.Join(configDir, "backup.txt"))
	if err != nil {
		return "", fmt.Errorf("read backup.txt: %w", err)
	}

	path := strings.TrimSpace(string(b))
	if path == "" {
		// Default backup location:
		//   <configDir>/../backups
		//
		// Keeps backups adjacent to the app/config layout
		// without hard-coding an absolute path.
		return filepath.Join(configDir, "..", "backups"), nil
	}

	return path, nil
}
