package app

import (
	"fmt"

	"file-maintenance/internal/config"
	"file-maintenance/internal/logging"
	"file-maintenance/internal/maintenance"
	"file-maintenance/internal/types"
)

type runtimePlatform interface {
	ShowCritical(title, message string)
	AvailableBytes(path string) (uint64, error)
}

func Run(cfg types.AppConfig, log *logging.Logger, platform runtimePlatform, cliRuntime types.RuntimeConfigOverrides) error {
	// -----------------------------------------------------------------------------
	// Read all configuration from config.ini.
	//
	// config.ini now owns the maintenance plan:
	// - [backup] section with 'path' key for backup destination
	// - [paths] section containing all paths to process
	//
	// Runtime behavior is resolved separately using this precedence:
	// defaults -> config.ini -> explicitly passed CLI flags
	// -----------------------------------------------------------------------------
	plan, fileRuntime, err := config.ReadAllConfig(cfg.ConfigDir, log)
	if err != nil {
		return err
	}

	runtimeCfg := types.DefaultRuntimeConfig()
	runtimeCfg = types.ApplyRuntimeOverrides(runtimeCfg, fileRuntime)
	runtimeCfg = types.ApplyRuntimeOverrides(runtimeCfg, cliRuntime)
	cfg = types.ApplyRuntimeConfig(cfg, runtimeCfg)
	cfg.BackupDir = plan.BackupDir

	pathconfig := plan.Paths
	if cfg.NoBackup {
		log.Warn("No-backup mode enabled - all configured paths will run as delete-only for this run")
		for i := range pathconfig {
			pathconfig[i].Backup = false
		}
	}

	// Log paths and their backup settings.
	for _, pc := range pathconfig {
		backupStr := "yes"
		if !pc.Backup {
			backupStr = "no"
		}
		log.Infof("Path: %s (backup: %s)", pc.Path, backupStr)
	}

	// -----------------------------------------------------------------------------
	// Validate backup location (only if any paths have backup enabled).
	//
	// When any path has backup enabled:
	// - Validate the backup path is accessible (important for network shares).
	// - If the backup path is not accessible, abort BEFORE any deletions occur.
	//
	// When all paths have backup disabled:
	// - Skip backup validation.
	// - Worker will run in "delete only" mode.
	// -----------------------------------------------------------------------------
	anyBackupEnabled := false
	for _, pc := range pathconfig {
		if pc.Backup {
			anyBackupEnabled = true
			break
		}
	}

	if anyBackupEnabled {
		log.Infof("Backup location: %s", plan.BackupDir)

		// Safety check:
		// Ensure the destination is reachable (especially on SMB shares) to avoid
		// the dangerous case of deleting source files without successfully copying
		// them somewhere safe first.
		if !maintenance.CheckBackupPath(plan.BackupDir) {
			errMsg := fmt.Sprintf("Backup path is not accessible: %s\n\nPlease check path and permissions.", plan.BackupDir)
			// Show a platform-specific critical notification for the user.
			platform.ShowCritical("Backup Location Error", errMsg)

			return fmt.Errorf("backup path not accessible: %s", plan.BackupDir)
		}
	} else {
		log.Warn("All paths have backup disabled - running in delete-only mode")
	}

	// -----------------------------------------------------------------------------
	// Run the worker.
	//
	// Worker responsibilities (high-level contract):
	// - Scan configured paths (optionally concurrent / bounded by cfg.Walkers).
	// - Process only files older than cfg.Days.
	// - If backup is enabled for a path: copy to backupLocation before deleting.
	// - If backup is disabled: delete without copying.
	//
	// Important:
	// - We must return Worker errors; otherwise failures are invisible to callers
	//   and Task Scheduler exit codes.
	// -----------------------------------------------------------------------------
	if err := maintenance.Worker(pathconfig, plan.BackupDir, cfg, log, platform); err != nil {
		return err
	}

	// -----------------------------------------------------------------------------
	// Housekeeping: prune old logs.
	//
	// Only do this when file logging is enabled. When -no-logs is set, LogDir may
	// be unused and we should not attempt filesystem cleanup.
	// -----------------------------------------------------------------------------
	if !cfg.LogSettings.NoLogs {
		if err := maintenance.RemoveOldLogs(cfg.LogSettings.LogDir, cfg.LogRetention); err != nil {
			return err
		}
	}

	return nil
}
