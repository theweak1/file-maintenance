package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"file-maintenance/internal/logging"
	"file-maintenance/internal/types"
)

// ReadAllConfig reads the file maintenance plan and runtime settings from config.ini.
//
// Required sections:
//
//	[backup]
//	path=D:\backups
//
//	[paths]
//	C:\temp\old, yes
//	\\server\share\incoming, no
//
// Optional sections:
//
//	[settings]
//	days=7
//	log-retention=30
//
//	[advanced]
//	walkers=1
//	queue-size=300
//	retries=2
//	cooldown=50ms
//	max-files=0
//	max-runtime=55m
//	no-backup=false
//
// Path entries support both folders and individual files. Each path can opt in
// or out of backup with yes/no. If omitted, backup defaults to enabled.
//
// Errors:
//   - Returns an error if config.ini cannot be read.
//   - Returns an error if [backup] section is missing or has no path.
//   - Returns an error if [paths] section is missing or contains no valid paths.
//   - No validation of path existence is performed here; that is deferred
//     to later stages so configuration errors fail fast and explicitly.
func ReadAllConfig(configDir string, log *logging.Logger) (types.FilePlanConfig, types.RuntimeConfigOverrides, error) {
	configFile := filepath.Join(configDir, "config.ini")

	b, err := os.ReadFile(configFile)
	if err != nil {
		return types.FilePlanConfig{}, types.RuntimeConfigOverrides{}, fmt.Errorf("read config.ini: %w", err)
	}

	// Remove UTF-8 BOM if present.
	content := string(b)
	if len(content) > 0 && content[0] == 0xEF && len(content) > 2 && content[1] == 0xBB && content[2] == 0xBF {
		content = content[3:]
	}

	sections, standaloneLines, err := parseIniSections(content)
	if err != nil {
		return types.FilePlanConfig{}, types.RuntimeConfigOverrides{}, fmt.Errorf("parse config.ini: %w", err)
	}

	backupSection, ok := sections["backup"]
	if !ok {
		return types.FilePlanConfig{}, types.RuntimeConfigOverrides{}, fmt.Errorf("missing [backup] section in config.ini")
	}

	backupPath, ok := backupSection["path"]
	if !ok || backupPath == "" {
		return types.FilePlanConfig{}, types.RuntimeConfigOverrides{}, fmt.Errorf("missing 'path' key in [backup] section")
	}

	pathsSection, ok := sections["paths"]
	if !ok {
		return types.FilePlanConfig{}, types.RuntimeConfigOverrides{}, fmt.Errorf("missing [paths] section in config.ini")
	}

	pathconfig, err := parsePathsSection(log, pathsSection, standaloneLines["paths"])
	if err != nil {
		return types.FilePlanConfig{}, types.RuntimeConfigOverrides{}, err
	}
	if len(pathconfig) == 0 {
		return types.FilePlanConfig{}, types.RuntimeConfigOverrides{}, fmt.Errorf("missing valid path entries in [paths] section")
	}

	plan := types.FilePlanConfig{
		BackupDir: backupPath,
		Paths:     pathconfig,
	}

	runtimeOverrides := parseRuntimeSettings(sections)

	return plan, runtimeOverrides, nil
}

// parseIniSections parses a simple INI-style config file.
// Returns a map of section name to key-value pairs and a list of standalone lines.
func parseIniSections(content string) (map[string]map[string]string, map[string][]string, error) {
	sections := make(map[string]map[string]string)
	standaloneLines := make(map[string][]string)
	var currentSection string

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check for section header
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			sectionName := strings.Trim(line, "[]")
			if sectionName == "" {
				return nil, nil, fmt.Errorf("empty section name")
			}
			currentSection = sectionName
			sections[currentSection] = make(map[string]string)
			continue
		}

		// Skip comments
		if strings.HasPrefix(line, ";") {
			continue
		}

		// Parse key-value pair or standalone line
		if currentSection == "" {
			return nil, nil, fmt.Errorf("line outside of section: %s", line)
		}

		// Check if line contains '='
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				sections[currentSection][key] = value
			}
		} else {
			// Standalone line (e.g., a path without key)
			standaloneLines[currentSection] = append(standaloneLines[currentSection], line)
		}
	}

	return sections, standaloneLines, nil
}

// parsePathsSection parses the [paths] section entries.
// Supports both inline format and key-value format:
//   - Inline: paths listed directly under [paths] section
//   - Key-value: paths under 'paths' key
func parsePathsSection(log *logging.Logger, section map[string]string, standalone []string) ([]types.PathConfig, error) {
	var pathsContent string

	// Check for 'paths' key first
	if content, ok := section["paths"]; ok && content != "" {
		pathsContent = content
	} else {
		// Use standalone lines
		pathsContent = strings.Join(standalone, "\n")
	}

	lines := strings.Split(pathsContent, "\n")
	config := make([]types.PathConfig, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse path and optional backup setting
		path, backup, err := parsePathLine(line)
		if err != nil {
			log.Warnf("Skipping malformed line in config.ini [paths]: %s (error: %v)", line, err)
			continue
		}

		// Check if path is a directory or file
		isDir := true
		fi, err := os.Stat(path)
		if err == nil {
			isDir = fi.IsDir()
		}

		config = append(config, types.PathConfig{
			Path:   path,
			Backup: backup,
			IsDir:  isDir,
		})
	}

	return config, nil
}

// parsePathLine parses a single path entry from paths section.
//
// Returns:
//   - path: the file or folder path
//   - backup: true if backup is enabled, false otherwise
//   - error: if the line is malformed
func parsePathLine(line string) (string, bool, error) {
	// Check for comma-separated format
	if strings.Contains(line, ",") {
		parts := strings.SplitN(line, ",", 2)
		path := strings.TrimSpace(parts[0])
		backupStr := strings.ToLower(strings.TrimSpace(parts[1]))

		if path == "" {
			return "", false, fmt.Errorf("empty path in line: %s", line)
		}

		switch backupStr {
		case "yes", "y", "true", "1":
			return path, true, nil
		case "no", "n", "false", "0":
			return path, false, nil
		default:
			// Unrecognized backup setting, default to true (backup enabled)
			return path, true, nil
		}
	}

	// No comma - use default behavior (backup enabled)
	return strings.TrimSpace(line), true, nil
}

// ReadFolderList reads the list of paths to process from `paths.txt`.
//
// Deprecated: Use ReadAllConfig instead which reads from config.ini
func ReadFolderList(configDir string, log *logging.Logger) ([]types.PathConfig, error) {
	pathsFile := filepath.Join(configDir, "paths.txt")

	b, err := os.ReadFile(pathsFile)
	if err != nil {
		return nil, fmt.Errorf("read paths.txt: %w", err)
	}

	lines := strings.Split(string(b), "\n")
	config := make([]types.PathConfig, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		path, backup, err := parsePathLine(line)
		if err != nil {
			log.Warnf("Skipping malformed line in paths.txt: %s (error: %v)", line, err)
			continue
		}

		isDir := true
		fi, err := os.Stat(path)
		if err == nil {
			isDir = fi.IsDir()
		}

		config = append(config, types.PathConfig{
			Path:   path,
			Backup: backup,
			IsDir:  isDir,
		})
	}

	return config, nil
}

// ReadBackupLocation reads the backup destination path from `backup.txt`.
//
// Deprecated: Use ReadAllConfig instead which reads from config.ini
func ReadBackupLocation(configDir string) (string, error) {
	b, err := os.ReadFile(filepath.Join(configDir, "backup.txt"))
	if err != nil {
		return "", fmt.Errorf("read backup.txt: %w", err)
	}

	path := strings.TrimSpace(string(b))
	if path == "" {
		return filepath.Join(configDir, "..", "backups"), nil
	}

	return path, nil
}

// parseRuntimeSettings parses [settings] and [advanced] as runtime overrides.
func parseRuntimeSettings(sections map[string]map[string]string) types.RuntimeConfigOverrides {
	var cfg types.RuntimeConfigOverrides

	if settings, ok := sections["settings"]; ok {
		if v, ok := settings["days"]; ok && v != "" {
			if days, err := strconv.Atoi(v); err == nil {
				cfg.Days = intPtr(days)
			}
		}
		if v, ok := settings["log-retention"]; ok && v != "" {
			if logRetention, err := strconv.Atoi(v); err == nil {
				cfg.LogRetention = intPtr(logRetention)
			}
		}
	}

	if advanced, ok := sections["advanced"]; ok {
		if v, ok := advanced["walkers"]; ok && v != "" {
			if walkers, err := strconv.Atoi(v); err == nil {
				cfg.Walkers = intPtr(walkers)
			}
		}
		if v, ok := advanced["queue-size"]; ok && v != "" {
			if queueSize, err := strconv.Atoi(v); err == nil {
				cfg.QueueSize = intPtr(queueSize)
			}
		}
		if v, ok := advanced["retries"]; ok && v != "" {
			if retries, err := strconv.Atoi(v); err == nil {
				cfg.Retries = intPtr(retries)
			}
		}
		if v, ok := advanced["cooldown"]; ok && v != "" {
			if cooldown, err := parseDurationValue(v); err == nil {
				cfg.Cooldown = durationPtr(cooldown)
			}
		}
		if v, ok := advanced["max-files"]; ok && v != "" {
			if maxFiles, err := strconv.Atoi(v); err == nil {
				cfg.MaxFiles = intPtr(maxFiles)
			}
		}
		if v, ok := advanced["max-runtime"]; ok && v != "" {
			if maxRuntime, err := parseDurationValue(v); err == nil {
				cfg.MaxRuntime = durationPtr(maxRuntime)
			}
		}
		if v, ok := advanced["no-backup"]; ok && v != "" {
			if noBackup, err := strconv.ParseBool(v); err == nil {
				cfg.NoBackup = boolPtr(noBackup)
			}
		}
	}

	return cfg
}

// parseDurationValue accepts Go duration strings such as 55m or 50ms. For
// backward compatibility with older config files, plain numbers are interpreted
// as milliseconds.
func parseDurationValue(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("empty duration")
	}
	if value == "0" {
		return 0, nil
	}
	if d, err := time.ParseDuration(value); err == nil {
		return d, nil
	}
	ms, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return time.Duration(ms) * time.Millisecond, nil
}

func intPtr(v int) *int { return &v }

func boolPtr(v bool) *bool { return &v }

func durationPtr(v time.Duration) *time.Duration { return &v }
