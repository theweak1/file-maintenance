package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"file-maintenance/internal/app"
	"file-maintenance/internal/logging"
	"file-maintenance/internal/setup"
	"file-maintenance/internal/types"
	"file-maintenance/internal/utils"
)

func main() {
	// -----------------------------------------------------------------------------
	// Resolve the "app root" (directory of the running executable).
	//
	// Why:
	// - This tool is meant to run unattended and from arbitrary working directories
	//   (Windows Task Scheduler, terminal, etc.).
	// - Default paths (config/, logs/) live next to the .exe to reduce surprises.
	//
	// Note:
	// - ExeDir() can fail in some environments (permissions, odd launch contexts),
	//   so we fall back to the current working directory.
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
	defaultLogDir := filepath.Join(root, "logs")
	defaultCfgDir := filepath.Join(root, "config")

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
	// Check if configuration exists, launch setup wizard if not.
	//
	// This ensures first-time users are guided through the setup process.
	// The setup wizard uses PowerShell GUI to create config.ini.
	// -----------------------------------------------------------------------------
	configExists, err := setup.EnsureConfig(cfg.ConfigDir, root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to ensure configuration: %v\n", err)
		os.Exit(1)
	}

	if !configExists {
		fmt.Println("Setup was cancelled or failed. Please run the tool again after configuring.")
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
	if err := app.Run(cfg, log); err != nil {
		// We already have a logger, so report the error there as well.
		log.Errorf("internal exited with error: %v", err)
		os.Exit(1)
	}
}
