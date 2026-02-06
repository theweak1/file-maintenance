package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"file-maintenance/internal/logging"
	"file-maintenance/internal/types"
)

// ReadFolderList reads the list of paths to process from `paths.txt`.
//
// Contract:
// - One entry per line (can be a file OR a folder)
// - Empty lines are ignored
// - Lines starting with '#' are treated as comments
// - Each line can have an optional backup setting: "path, yes" or "path, no"
//
// Format:
//   - path                    - uses default backup behavior (backup enabled)
//   - path, yes               - backup enabled for this path
//   - path, no                - backup disabled for this path
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
// - Control backup behavior per-path using "yes" or "no" after a comma
//
// Example folders.txt:
//
//	# Delete all old files from these folders (with backup)
//	C:\temp\old, yes
//
//	# Network share (without backup)
//	\\server\share\incoming, no
//
//	# Delete specific files (with backup)
//	C:\Data\Images\old-photo.jpg, yes

//	# Delete specific files (without backup)
//	C:\Logs\debug.log, no
//
// Errors:
//   - Returns an error if paths.txt cannot be read.
//   - No validation of path existence is performed here; that is deferred
//     to later stages so configuration errors fail fast and explicitly.
func ReadFolderList(configDir string, log *logging.Logger) ([]types.PathConfig, error) {

	pathsFile := filepath.Join(configDir, "paths.txt")

	b, err := os.ReadFile(pathsFile)
	if err != nil {
		return nil, fmt.Errorf("read paths.txt: %w", err)
	}

	// Split by newline and parse each entry.
	lines := strings.Split(string(b), "\n")
	configs := make([]types.PathConfig, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse path and optional backup setting.
		path, backup, err := parsePathLine(line)
		if err != nil {
			// Skip malformed lines but log a warning (could extend to return error)
			log.Warnf("Skipping malformed line in paths.txt: %s (error: %v)", line, err)
			continue
		}

		// Check if path is a directory or file.
		isDir := true
		fi, err := os.Stat(path)
		if err == nil {
			isDir = fi.IsDir()
		}

		configs = append(configs, types.PathConfig{
			Path:   path,
			Backup: backup,
			IsDir:  isDir,
		})
	}

	return configs, nil
}

// parsePathLine parses a single line from paths.txt.
//
// Returns:
//   - path: the file or folder path
//   - backup: true if backup is enabled, false otherwise
//   - error: if the line is malformed
func parsePathLine(line string) (string, bool, error) {
	// Check for comma-separated format.
	if strings.Contains(line, ",") {
		parts := strings.SplitN(line, ",", 2)
		path := strings.TrimSpace(parts[0])
		backupStr := strings.ToLower(strings.TrimSpace(parts[1]))

		if path == "" {
			return "", false, fmt.Errorf("empty path in line: %s", line)
		}

		switch backupStr {
		case "yes", "Y", "y", "true", "1":
			return path, true, nil
		case "no", "N", "n", "false", "0":
			return path, false, nil
		default:
			// Unrecognized backup setting, default to true (backup enabled)
			return path, true, nil
		}
	}

	// No comma - use default behavior (backup enabled)
	return strings.TrimSpace(line), true, nil
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
