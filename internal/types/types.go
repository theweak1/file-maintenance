package types

import (
	"time"

	"file-maintenance/internal/logging"
)

// AppConfig is the central configuration object for the application.
//
// It is constructed once in main(), passed through app.Run(), and then shared
// with the maintenance worker. Treat it as read-only after creation.
//
// Design goals:
// - Keep runtime behavior configurable via CLI flags + config files
// - Make scheduled runs predictable and safe
// - Avoid globals by threading config explicitly
type AppConfig struct {
	// Days controls file retention:
	// - Only files strictly older than (now - Days) are eligible.
	// - A value of 0 means "older than now" (effectively selects all files).
	//
	// NOTE: "Strictly older" means files exactly at the cutoff timestamp are NOT considered old.
	Days int

	// NoBackup disables backup entirely.
	//
	// When true, the worker deletes eligible files without copying them first.
	// This should only be used intentionally (e.g., controlled cleanup runs).
	NoBackup bool

	// LogRetention controls how long log files are kept (in days).
	// Used by maintenance.RemoveOldLogs().
	LogRetention int

	// ConfigDir is the directory containing configuration files such as:
	// - folders.txt
	// - backup.txt
	// - logging.json
	//
	// Typically defaults to "<exeDir>/configs".
	ConfigDir string

	// BackupDir is the resolved backup destination root (typically loaded from backup.txt).
	//
	// Note:
	// - app.Run() usually reads this from config and passes it to Worker().
	// - The worker takes the backup root path as an argument, so this field is optional
	//   depending on how you choose to structure app wiring.
	BackupDir string

	// LogSettings controls logging behavior (file vs stdout, log directory).
	LogSettings logging.LogSettings

	// ---------------------------------------------------------------------
	// Resource controls (important for Windows + SMB/network + scheduled runs)
	// ---------------------------------------------------------------------

	// Walkers controls how many configured folders can be scanned concurrently.
	//
	// Recommended:
	// - 1 for network-heavy or busy systems
	// - 2–3 only if scans are slow and safe to parallelize
	Walkers int

	// QueueSize is the size of the buffered channel between folder walkers
	// and the single file processor.
	//
	// This provides backpressure and prevents unbounded memory usage if folder
	// scans run ahead of copy/delete operations.
	QueueSize int

	// MaxFiles caps how many candidate file jobs the processor will *handle*
	// in a single run.
	//
	// - 0 means unlimited
	// - This is a safety/operational cap for predictable run size
	//
	// NOTE: "handled" means the processor attempted to process the job; it does
	// not necessarily mean "deleted successfully".
	MaxFiles int

	// MaxRuntime caps how long a run is allowed to execute.
	//
	// - 0 means unlimited
	// - Parsed as a time.Duration (e.g., 30m, 1h)
	//
	// This is best-effort: once the limit is hit, the worker stops scheduling
	// new work and exits as soon as practical.
	MaxRuntime time.Duration

	// Cooldown inserts a small sleep after each handled job.
	//
	// This smooths out I/O bursts and reduces load on:
	// - SMB/network shares
	// - busy workstations
	Cooldown time.Duration

	// Retries controls how many times a file copy is retried on failure.
	//
	// This is especially important for:
	// - transient network issues
	// - temporary file locks (e.g., antivirus scanners)
	Retries int
}
