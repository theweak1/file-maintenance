package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"file-maintenance/internal/app"
	"file-maintenance/internal/logging"
	"file-maintenance/internal/platform"
	"file-maintenance/internal/types"
	"file-maintenance/internal/utils"
)

const appName = "file-maintenance"

func main() {
	// -----------------------------------------------------------------------------
	// Resolve the current platform implementation.
	//
	// Platform owns OS-specific behavior such as:
	// - native critical notifications
	// - first-run configuration setup behavior
	// - optional OS-conventional config/log path resolution
	//
	// The application currently uses portable defaults for config and logs
	// (<exe>/config and <exe>/logs), while still routing setup and notifications
	// through the active platform implementation.
	// -----------------------------------------------------------------------------
	pf := platform.Current()

	// -----------------------------------------------------------------------------
	// Resolve application root (directory of the running executable).
	//
	// Why:
	// - The default config and log directories are intentionally located beside the
	//   executable to support portable deployments and predictable Task Scheduler use.
	// - If ExeDir() fails in an unusual launch context, fall back to the current
	//   working directory rather than exiting before logging can be initialized.
	// -----------------------------------------------------------------------------
	root, err := utils.ExeDir()
	if err != nil {
		root, _ = os.Getwd()
	}

	// -----------------------------------------------------------------------------
	// Default locations relative to the app root.
	//
	// - config/ holds config.ini, logging.json, etc.
	// - logs/ is where the logger writes log files (unless -no-logs is set)
	// -----------------------------------------------------------------------------

	// NOTICE: platform.DefaultConfigDir() and DefaultLogDir() use OS-specific logic to determine where config and logs should go. For example:
	// - On Windows, config might go to %APPDATA%\file-maintenance\config.ini and logs to %LOCALAPPDATA%\file-maintenance\logs\
	// - On Linux, config might go to ~/.config/file-maintenance/config.ini and logs to ~/.cache/file-maintenance/logs/
	// - On macOS, config might go to ~/Library/Application Support/file-maintenance/config.ini and logs to ~/Library/Caches/file-maintenance/logs/
	// If you want that behavior uncomment the lines below and remove the fallbacks that point to the app root. The platform defaults are more in line with user expectations and OS conventions, but the app root fallback can be useful for portable use cases (e.g., running from a USB drive).

	// -----------------------------------------------------------------------------
	//  Resolve platform-specific default config and log directories, with fallbacks to app root.
	// defaultCfgDir, err := pf.DefaultConfigDir(appName)
	// if err != nil {
	// 	defaultCfgDir = filepath.Join(root, "config")
	// }

	// defaultLogDir, err := pf.DefaultLogDir(appName)
	// if err != nil {
	// 	defaultLogDir = filepath.Join(root, "logs")
	// }

	defaultCfgDir := filepath.Join(root, "config")
	defaultLogDir := filepath.Join(root, "logs")

	// -----------------------------------------------------------------------------
	// CLI flags
	//
	// Keep flags grouped and explicit. This tool is commonly run unattended
	// (Task Scheduler), so predictable flags + safe defaults matter.
	//
	// Resource-control flags:
	// - walkers / queue-size: bound concurrency and memory usage while scanning.
	// - max-files / max-runtime: hard caps to prevent runaway work.
	// - cooldown: optional pacing between file operations (SMB-friendly).
	// - retries: tolerate transient copy failures (e.g., network share hiccups).
	// -----------------------------------------------------------------------------
	var (
		// Retention policy for candidate files (only files older than this are processed).
		days = flag.Int("days", 7, "Number of days to retain files")

		// Retention policy for *log files* (housekeeping).
		logRetention = flag.Int("log-retention", 30, "Number of days to retain log files")

		// Config directory path (defaults next to the binary).
		configDir = flag.String("config-dir", defaultCfgDir, "Config directory (defaults next to the binary)")

		// Logging controls
		logDir = flag.String("log-dir", defaultLogDir, "Log directory (defaults next to the binary)")
		noLogs = flag.Bool("no-logs", false, "If set, logging is disabled and output is sent to stdout")

		// Resource controls used by maintenance.Worker
		walkers    = flag.Int("walkers", 1, "Number of concurrent folder walkers")
		queueSize  = flag.Int("queue-size", 300, "Size of the buffered jobs channel")
		maxFiles   = flag.Int("max-files", 0, "Maximum number of files to process (0 = unlimited)")
		maxRuntime = flag.Duration("max-runtime", 30*time.Minute, "Maximum runtime duration (0 = unlimited)")
		cooldown   = flag.Duration("cooldown", 0, "Cooldown duration after each file operation")
		retries    = flag.Int("retries", 2, "Number of copy retries on failure")
	)

	// Parse CLI flags once at process startup.
	flag.Parse()

	// -----------------------------------------------------------------------------
	// Build AppConfig (the single configuration object passed into internal/app).
	//
	// Notes:
	// - BackupDir is intentionally left empty here; app.Run() resolves it from
	//   config files (e.g., config/config.ini) so scheduled runs don't require
	//   passing a long path via CLI flags.
	// - LogSettings control whether logs are written to disk or printed to stdout.
	// -----------------------------------------------------------------------------
	cfg := types.AppConfig{
		Days:         *days,
		ConfigDir:    *configDir,
		LogRetention: *logRetention,

		// Read from config/config.ini inside app.Run().
		BackupDir: "",

		LogSettings: logging.LogSettings{
			NoLogs: *noLogs,
			LogDir: *logDir,
		},

		// Resource controls enforced by maintenance.Worker()
		Walkers:    *walkers,
		QueueSize:  *queueSize,
		MaxFiles:   *maxFiles,
		MaxRuntime: *maxRuntime,
		Cooldown:   *cooldown,
		Retries:    *retries,
	}

	// -----------------------------------------------------------------------------
	// Initialize the logger once.
	//
	// Why:
	// - Worker may use multiple goroutines; they all share one logger instance.
	// - If logger init fails, we can't reliably log, so we fall back to stderr.
	// -----------------------------------------------------------------------------
	log, err := logging.New(cfg.ConfigDir, cfg.LogSettings)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}

	// If you later add Close() (flush buffers / close handles), you can defer it here:
	// defer log.Close()

	// -----------------------------------------------------------------------------
	// Ensure configuration exists before running maintenance.
	//
	// Windows behavior:
	// - If <config-dir>/config.ini is missing, launch the embedded PowerShell setup
	//   wizard and continue only if the wizard creates the file successfully.
	//
	// Linux/macOS behavior:
	// - No GUI wizard is launched. The platform returns false if config.ini is
	//   missing so the application exits safely without processing files.
	// -----------------------------------------------------------------------------
	configExists, err := pf.EnsureConfig(cfg.ConfigDir, root)
	if err != nil {
		log.Errorf("failed to ensure config: %v", err)
		os.Exit(1)
	}

	if !configExists {
		log.Info("Setup was cancelled or failed. Please run the tool again after configuring.")
		os.Exit(1)
	}

	// -----------------------------------------------------------------------------
	// Run the application.
	//
	// app.Run() is responsible for:
	// - reading config (paths list, backup root, logging settings)
	// - pruning old logs (if logging enabled)
	// - calling maintenance.Worker() to process eligible files
	// -----------------------------------------------------------------------------
	if err := app.Run(cfg, log, pf); err != nil {
		// We already have a logger, so report the error there as well.
		log.Errorf("internal exited with error: %v", err)
		fmt.Fprintf(os.Stderr, "internal exited with error: %v\n", err)
		os.Exit(1)
	}
}
