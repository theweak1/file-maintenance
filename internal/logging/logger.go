package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LogSettings controls where logs go.
//
// Modes:
// - NoLogs=true  => console-only (stdout). No log files are created.
// - NoLogs=false => write logs to files under LogDir.
//
// Why this exists:
//   - Scheduled runs usually need file logs (inspect runs after the fact).
//   - Quick/manual runs sometimes prefer console-only output (no file I/O,
//     fewer permissions issues).
type LogSettings struct {
	NoLogs bool
	LogDir string
}

// Logger is a lightweight, goroutine-safe logger intended for:
// - a single shared instance across the entire app
// - safe concurrent writes from multiple goroutines (folder walkers + processor)
//
// Thread safety model:
//   - All file writes are guarded by mu to prevent log line interleaving.
//   - In NoLogs mode we write to stdout; stdout writes may still interleave across
//     goroutines (acceptable for console-only mode).
type Logger struct {
	// ConfigDir is where we look for logging.json (enabled/disabled log levels).
	ConfigDir string

	// settings controls whether we log to stdout only or also to files.
	settings LogSettings

	// levels stores enabled log levels loaded once at startup from logging.json.
	levels map[string]bool

	// mu serializes file writes so multiple goroutines can call Log() safely.
	mu sync.Mutex
}

// New initializes a Logger.
//
// Behavior:
// - Reads configDir/logging.json (if present) to determine enabled log levels.
// - If logging.json is missing, sensible defaults are used (see loadLevels).
// - If settings.NoLogs is false:
//   - settings.LogDir must be set
//   - the directory is created if needed (fail early if invalid/unwritable)
//
// Notes:
//   - Creating LogDir early is helpful for Task Scheduler runs: if permissions are
//     wrong, we fail fast at startup instead of silently losing logs.
//   - For network paths, mkdir failure is a strong signal of access/permission problems.
func New(configDir string, settings LogSettings) (*Logger, error) {
	levels, err := loadLevels(configDir)
	if err != nil {
		return nil, err
	}

	// If file logging is enabled, ensure log directory exists.
	// If NoLogs is true, we intentionally skip all file/directory requirements.
	if !settings.NoLogs {
		if settings.LogDir == "" {
			return nil, fmt.Errorf("log dir is empty (settings.LogDir)")
		}
		if err := os.MkdirAll(settings.LogDir, os.ModePerm); err != nil {
			return nil, fmt.Errorf("create log directory: %w", err)
		}
	}

	return &Logger{
		ConfigDir: configDir,
		settings:  settings,
		levels:    levels,
	}, nil
}

// loadLevels loads log-level enable/disable configuration from logging.json.
//
// If logging.json does not exist, default levels are returned:
// - INFO/WARN/ERROR/SUCCESS/FATAL enabled
// - COUNT enabled (used for end-of-run totals and summary counters)
// - DEBUG disabled (to avoid noisy scheduled runs)
//
// Policy for unknown levels (fail-open):
//   - If code introduces a new level and logging.json hasn't been updated yet,
//     it's safer to log than to silently drop messages.
func loadLevels(configDir string) (map[string]bool, error) {
	path := filepath.Join(configDir, "logging.json")

	// If config file is missing, return default levels.
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return map[string]bool{
				"DEBUG":   false,
				"COUNT":   true,
				"INFO":    true,
				"WARN":    true,
				"ERROR":   true,
				"SUCCESS": true,
				"FATAL":   true,
			}, nil
		}
		return nil, fmt.Errorf("stat logging config: %w", err)
	}

	// Config exists: read and parse JSON.
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read logging config: %w", err)
	}

	var levels map[string]bool
	if err := json.Unmarshal(b, &levels); err != nil {
		return nil, fmt.Errorf("parse logging config: %w", err)
	}
	return levels, nil
}

// Enabled returns whether a log level is enabled.
//
// Policy:
// - If the level exists in config and is false => disabled.
// - If the level does not exist in config => enabled (fail-open).
//
// This prevents new levels from being unintentionally dropped until logging.json is updated.
func (l *Logger) Enabled(level string) bool {
	level = strings.ToUpper(strings.TrimSpace(level))

	enabled, ok := l.levels[level]
	if ok && !enabled {
		return false
	}
	return true
}

// Log writes a single log line to either stdout (NoLogs mode) or daily log files.
//
// Output format:
//
//	[MM/DD/YY HH:MM:SS] [LEVEL] -> message
//
// File mode behavior:
// - Writes every line to: maintenance_YYYY-MM-DD.log
// - Writes COUNT lines also to: count_YYYY-MM-DD.log   (summary counters, per-folder totals, etc.)
// - Writes ERROR lines also to: errors_YYYY-MM-DD.log  (quick place to scan failures)
//
// Thread safety:
// - File writes are guarded by l.mu so multiple goroutines can't interleave lines.
// - We lock once per Log() call and write to all relevant files within that lock.
func (l *Logger) Log(level, msg string) {
	level = strings.ToUpper(strings.TrimSpace(level))

	// Respect configured levels.
	if !l.Enabled(level) {
		return
	}

	now := time.Now()
	date := now.Format("2006-01-02")
	timeStamp := now.Format("01/02/06 15:04:05")

	stamp := fmt.Sprintf("[%s] [%s]", timeStamp, level)
	line := fmt.Sprintf("%s -> %s\n", stamp, msg)

	// Console-only mode: do not touch filesystem.
	if l.settings.NoLogs {
		fmt.Print(line)
		return
	}

	// Daily rolling main log filename.
	// Stable per-day filenames make it easy to inspect scheduled runs.
	maintenanceFile := filepath.Join(l.settings.LogDir, fmt.Sprintf("maintenance_%s.log", date))

	// Serialize file writes across goroutines.
	l.mu.Lock()
	defer l.mu.Unlock()

	// Write main log line first.
	if err := appendLine(maintenanceFile, line); err != nil {
		// If file logging fails, stdout is our fallback visibility.
		fmt.Printf("Error writing to log file: %v\n", err)
		return
	}

	// COUNT is used for summary numbers (like "deleted files per folder" at end of run).
	if level == "COUNT" {
		countFile := filepath.Join(l.settings.LogDir, fmt.Sprintf("count_%s.log", date))
		if err := appendLine(countFile, line); err != nil {
			fmt.Printf("Error writing to count log file: %v\n", err)
			return
		}
	}

	// ERROR is duplicated into a dedicated file so failures are easy to scan.
	if level == "ERROR" {
		errorFile := filepath.Join(l.settings.LogDir, fmt.Sprintf("errors_%s.log", date))
		if err := appendLine(errorFile, line); err != nil {
			fmt.Printf("Error writing to error log file: %v\n", err)
			return
		}
	}
}

// appendLine appends a single line to a file, creating it if needed.
//
// Notes:
//   - Each call opens and closes the file (simple + robust).
//   - If log throughput becomes a bottleneck, you can optimize by keeping open file
//     handles per day and rotating at midnight (still guarded by the same mutex).
func appendLine(path string, line string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(line)
	return err
}

// Convenience methods avoid passing level strings everywhere.
// They also make it easier to refactor/rename levels later without touching call sites.
func (l *Logger) Debug(msg string)   { l.Log("DEBUG", msg) }
func (l *Logger) Info(msg string)    { l.Log("INFO", msg) }
func (l *Logger) Warn(msg string)    { l.Log("WARN", msg) }
func (l *Logger) Error(msg string)   { l.Log("ERROR", msg) }
func (l *Logger) Success(msg string) { l.Log("SUCCESS", msg) }
func (l *Logger) Count(msg string)   { l.Log("COUNT", msg) }

// Fatal logs the message and exits the process with code 1.
//
// IMPORTANT:
//   - os.Exit(1) terminates immediately (defers do NOT run).
//   - Use Fatal only for unrecoverable states where continuing could cause harm.
//     Example: backup location invalid while backups are required (Run aborts before
//     Worker runs to avoid deleting without a successful backup destination).
func (l *Logger) Fatal(msg string) { l.Log("FATAL", msg); os.Exit(1) }

// Formatted helpers reduce repeated fmt.Sprintf usage at call sites.
func (l *Logger) Debugf(format string, args ...any)   { l.Debug(fmt.Sprintf(format, args...)) }
func (l *Logger) Infof(format string, args ...any)    { l.Info(fmt.Sprintf(format, args...)) }
func (l *Logger) Warnf(format string, args ...any)    { l.Warn(fmt.Sprintf(format, args...)) }
func (l *Logger) Errorf(format string, args ...any)   { l.Error(fmt.Sprintf(format, args...)) }
func (l *Logger) Successf(format string, args ...any) { l.Success(fmt.Sprintf(format, args...)) }
func (l *Logger) Countf(format string, args ...any)   { l.Count(fmt.Sprintf(format, args...)) }
func (l *Logger) Fatalf(format string, args ...any)   { l.Fatal(fmt.Sprintf(format, args...)) }
