package main

import (
	"errors"
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
	"file-maintenance/internal/version"
)

const appName = "file-maintenance"

func main() {
	// -----------------------------------------------------------------------------
	// Resolve the current platform implementation.
	// -----------------------------------------------------------------------------
	pf := platform.Current()

	// -----------------------------------------------------------------------------
	// Resolve application root (directory of the running executable).
	// -----------------------------------------------------------------------------
	root, err := utils.ExeDir()
	if err != nil {
		root, _ = os.Getwd()
	}

	// -----------------------------------------------------------------------------
	// Default locations relative to the app root.
	// -----------------------------------------------------------------------------
	defaultCfgDir := filepath.Join(root, "config")
	defaultLogDir := filepath.Join(root, "logs")
	defaultRuntime := types.DefaultRuntimeConfig()

	// -----------------------------------------------------------------------------
	// CLI flags
	//
	// Mode flags:
	// - No -run flag: open setup/configuration UI and do not perform maintenance
	//   unless the user chooses Save & Run in the UI.
	// - -run: execute the background backup/delete process.
	// - -setup: explicit alias for setup/configuration mode.
	//
	// Runtime flags only override config.ini when they are explicitly passed.
	// -----------------------------------------------------------------------------
	var (
		runMode   = flag.Bool("run", false, "Run the background backup/delete maintenance process")
		setupMode = flag.Bool("setup", false, "Open the setup/configuration UI and exit unless Save & Run is selected")

		// Retention policy for candidate files (only files older than this are processed).
		days = flag.Int("days", defaultRuntime.Days, "Number of days to retain files")

		// Retention policy for *log files* (housekeeping).
		logRetention = flag.Int("log-retention", defaultRuntime.LogRetention, "Number of days to retain log files")

		// Config directory path (defaults next to the binary).
		configDir = flag.String("config-dir", defaultCfgDir, "Config directory (defaults next to the binary)")

		// Logging controls.
		logDir = flag.String("log-dir", defaultLogDir, "Log directory (defaults next to the binary)")
		noLogs = flag.Bool("no-logs", false, "If set, logging is disabled and output is sent to stdout")

		// Resource controls used by maintenance.Worker.
		walkers    = flag.Int("walkers", defaultRuntime.Walkers, "Number of concurrent folder walkers")
		queueSize  = flag.Int("queue-size", defaultRuntime.QueueSize, "Maximum jobs collected into one worker batch")
		maxFiles   = flag.Int("max-files", defaultRuntime.MaxFiles, "Maximum number of files to process (0 = unlimited)")
		maxRuntime = flag.Duration("max-runtime", defaultRuntime.MaxRuntime, "Maximum runtime duration (0 = unlimited)")
		cooldown   = flag.Duration("cooldown", defaultRuntime.Cooldown, "Cooldown duration after each file operation")
		retries    = flag.Int("retries", defaultRuntime.Retries, "Number of copy retries on failure")
		noBackup   = flag.Bool("no-backup", defaultRuntime.NoBackup, "Disable all backups for this run and delete eligible files directly")

		shortVersion = flag.Bool("version", false, "Print version and exit")
		longVersion  = flag.Bool("long-version", false, "Print long version and exit")
	)

	flag.Parse()
	seenFlags := explicitFlags()

	if *shortVersion {
		fmt.Printf("file-maintenance %s\n", version.ShortVersion())
		return
	}

	if *longVersion {
		fmt.Printf("file-maintenance %s\n", version.LongVersion())
		return
	}

	cliRuntime := runtimeOverridesFromFlags(seenFlags, *days, *logRetention, *walkers, *queueSize, *maxFiles, *maxRuntime, *cooldown, *retries, *noBackup)

	// -----------------------------------------------------------------------------
	// Build the base AppConfig passed into internal/app.
	//
	// Runtime values are finalized inside app.Run() using:
	// defaults -> config.ini -> explicit CLI overrides.
	// -----------------------------------------------------------------------------
	cfg := types.ApplyRuntimeConfig(types.AppConfig{
		ConfigDir: *configDir,
		BackupDir: "",
		LogSettings: logging.LogSettings{
			NoLogs: *noLogs,
			LogDir: *logDir,
		},
	}, defaultRuntime)

	// -----------------------------------------------------------------------------
	// Initialize the logger once.
	// -----------------------------------------------------------------------------
	log, err := logging.New(cfg.ConfigDir, cfg.LogSettings)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}

	log.Infof("file-maintenance version: %s", version.ShortVersion())

	// -----------------------------------------------------------------------------
	// Setup-first mode.
	//
	// A plain double-click or plain CLI launch opens setup/configuration. This keeps
	// destructive backup/delete work behind an explicit -run flag, unless the user
	// chooses Save & Run from the Windows setup UI.
	// -----------------------------------------------------------------------------
	if !*runMode || *setupMode {
		action, err := pf.RunSetup(cfg.ConfigDir, root)
		if err != nil {
			log.Errorf("setup failed: %v", err)
			fmt.Fprintf(os.Stderr, "setup failed: %v\n", err)
			os.Exit(1)
		}

		switch action {
		case types.SetupActionSavedAndRun:
			log.Info("Setup saved. Continuing with maintenance because Save & Run was selected.")
		case types.SetupActionSaved:
			log.Info("Setup saved. Exiting without running maintenance.")
			return
		case types.SetupActionCancelled:
			log.Info("Setup was cancelled. Exiting without running maintenance.")
			os.Exit(1)
		default:
			log.Errorf("unknown setup action: %d", action)
			os.Exit(1)
		}
	}

	// -----------------------------------------------------------------------------
	// Explicit run mode safety check.
	//
	// -run should never unexpectedly open a GUI. If config.ini is missing, fail
	// clearly so scheduled/background runs do not hang waiting for user input.
	// -----------------------------------------------------------------------------
	configExists, err := configFileExists(cfg.ConfigDir)
	if err != nil {
		log.Errorf("failed to check config.ini: %v", err)
		os.Exit(1)
	}
	if !configExists {
		log.Errorf("config.ini not found in %s. Run without -run to open setup.", cfg.ConfigDir)
		fmt.Fprintf(os.Stderr, "config.ini not found in %s. Run without -run to open setup.\n", cfg.ConfigDir)
		os.Exit(1)
	}

	// -----------------------------------------------------------------------------
	// Run the application.
	// -----------------------------------------------------------------------------
	if err := app.Run(cfg, log, pf, cliRuntime); err != nil {
		log.Errorf("internal exited with error: %v", err)
		fmt.Fprintf(os.Stderr, "internal exited with error: %v\n", err)
		os.Exit(1)
	}
}

func explicitFlags() map[string]bool {
	seen := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		seen[f.Name] = true
	})
	return seen
}

func runtimeOverridesFromFlags(
	seen map[string]bool,
	days int,
	logRetention int,
	walkers int,
	queueSize int,
	maxFiles int,
	maxRuntime time.Duration,
	cooldown time.Duration,
	retries int,
	noBackup bool,
) types.RuntimeConfigOverrides {
	var overrides types.RuntimeConfigOverrides

	if seen["days"] {
		overrides.Days = intPtr(days)
	}
	if seen["log-retention"] {
		overrides.LogRetention = intPtr(logRetention)
	}
	if seen["walkers"] {
		overrides.Walkers = intPtr(walkers)
	}
	if seen["queue-size"] {
		overrides.QueueSize = intPtr(queueSize)
	}
	if seen["max-files"] {
		overrides.MaxFiles = intPtr(maxFiles)
	}
	if seen["max-runtime"] {
		overrides.MaxRuntime = durationPtr(maxRuntime)
	}
	if seen["cooldown"] {
		overrides.Cooldown = durationPtr(cooldown)
	}
	if seen["retries"] {
		overrides.Retries = intPtr(retries)
	}
	if seen["no-backup"] {
		overrides.NoBackup = boolPtr(noBackup)
	}

	return overrides
}

func configFileExists(configDir string) (bool, error) {
	_, err := os.Stat(filepath.Join(configDir, "config.ini"))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func intPtr(v int) *int { return &v }

func boolPtr(v bool) *bool { return &v }

func durationPtr(v time.Duration) *time.Duration { return &v }
