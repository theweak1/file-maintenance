package types

import (
	"time"

	"file-maintenance/internal/logging"
)

// SetupAction describes what the setup/configuration UI asked the application
// to do after the user closed it.
type SetupAction int

const (
	SetupActionCancelled SetupAction = iota
	SetupActionSaved
	SetupActionSavedAndRun
)

// PathConfig represents a path entry from config.ini with its associated backup setting.
//
// Format: path, yes|no (comma-separated)
// - path: the file or folder to process
// - backup: "yes" to enable backup, "no" to disable backup for this path
type PathConfig struct {
	Path   string
	Backup bool
	IsDir  bool
}

// FilePlanConfig is the maintenance plan loaded from config.ini.
//
// This answers:
// - Where are backups written?
// - Which files/folders are eligible for cleanup?
// - Which paths require backup before deletion?
type FilePlanConfig struct {
	BackupDir string
	Paths     []PathConfig
}

// RuntimeConfig contains execution behavior that can come from defaults,
// config.ini, or explicit CLI overrides.
//
// This answers:
// - How long should this run?
// - How old must files be?
// - How many walkers/batch jobs/retries should be used?
type RuntimeConfig struct {
	Days         int
	NoBackup     bool
	LogRetention int
	Walkers      int
	QueueSize    int
	MaxFiles     int
	MaxRuntime   time.Duration
	Cooldown     time.Duration
	Retries      int
}

// RuntimeConfigOverrides represents values explicitly provided by config.ini or
// by CLI flags. Pointers let zero be a real configured value instead of meaning
// "not provided".
type RuntimeConfigOverrides struct {
	Days         *int
	NoBackup     *bool
	LogRetention *int
	Walkers      *int
	QueueSize    *int
	MaxFiles     *int
	MaxRuntime   *time.Duration
	Cooldown     *time.Duration
	Retries      *int
}

// DefaultRuntimeConfig returns the safe default runtime behavior used before
// config.ini and explicit CLI overrides are applied.
func DefaultRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		Days:         7,
		NoBackup:     false,
		LogRetention: 30,
		Walkers:      1,
		QueueSize:    300,
		MaxFiles:     0,
		MaxRuntime:   30 * time.Minute,
		Cooldown:     0,
		Retries:      2,
	}
}

// ApplyRuntimeOverrides applies only provided runtime values to a base runtime
// configuration.
func ApplyRuntimeOverrides(base RuntimeConfig, overrides RuntimeConfigOverrides) RuntimeConfig {
	if overrides.Days != nil {
		base.Days = *overrides.Days
	}
	if overrides.NoBackup != nil {
		base.NoBackup = *overrides.NoBackup
	}
	if overrides.LogRetention != nil {
		base.LogRetention = *overrides.LogRetention
	}
	if overrides.Walkers != nil {
		base.Walkers = *overrides.Walkers
	}
	if overrides.QueueSize != nil {
		base.QueueSize = *overrides.QueueSize
	}
	if overrides.MaxFiles != nil {
		base.MaxFiles = *overrides.MaxFiles
	}
	if overrides.MaxRuntime != nil {
		base.MaxRuntime = *overrides.MaxRuntime
	}
	if overrides.Cooldown != nil {
		base.Cooldown = *overrides.Cooldown
	}
	if overrides.Retries != nil {
		base.Retries = *overrides.Retries
	}
	return base
}

// ApplyRuntimeConfig copies RuntimeConfig values into AppConfig so existing
// worker code can continue receiving one AppConfig value.
func ApplyRuntimeConfig(cfg AppConfig, runtime RuntimeConfig) AppConfig {
	cfg.Days = runtime.Days
	cfg.NoBackup = runtime.NoBackup
	cfg.LogRetention = runtime.LogRetention
	cfg.Walkers = runtime.Walkers
	cfg.QueueSize = runtime.QueueSize
	cfg.MaxFiles = runtime.MaxFiles
	cfg.MaxRuntime = runtime.MaxRuntime
	cfg.Cooldown = runtime.Cooldown
	cfg.Retries = runtime.Retries
	return cfg
}

// AppConfig is the central configuration object for the application.
//
// It is constructed once in main(), finalized in app.Run(), and then shared
// with the maintenance worker. Treat it as read-only after finalization.
//
// Design goals:
// - Keep the maintenance plan in config.ini
// - Let explicit CLI flags override matching runtime settings
// - Keep scheduled runs predictable and safe
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
	// When true, the app treats all configured paths as delete-only for the run.
	// This should only be used intentionally (e.g., controlled cleanup runs).
	NoBackup bool

	// LogRetention controls how long log files are kept (in days).
	// Used by maintenance.RemoveOldLogs().
	LogRetention int

	// ConfigDir is the directory containing configuration files such as:
	// - config.ini
	// - logging.json
	//
	// Typically defaults to "<exeDir>/config".
	ConfigDir string

	// BackupDir is the resolved backup destination root (loaded from config.ini).
	//
	// Note:
	// - app.Run() reads this from the file plan and passes it to Worker().
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
	// - 2-3 only if scans are slow and safe to parallelize
	Walkers int

	// QueueSize is the maximum number of jobs collected into one worker batch.
	//
	// This provides backpressure, prevents unbounded memory usage, and controls
	// how often backup destination space is checked.
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
