package app

import (
	"fmt"

	"file-maintenance/internal/config"
	"file-maintenance/internal/logging"
	"file-maintenance/internal/maintenance"
	"file-maintenance/internal/types"
	"file-maintenance/internal/utils"
)

func Run(cfg types.AppConfig, log *logging.Logger) error {
	// -----------------------------------------------------------------------------
	// Read all configuration from config.ini (single file).
	//
	// config.ini contains:
	// - [backup] section with 'path' key for backup destination
	// - [paths] section with 'paths' key containing all paths to process
	//
	// For unattended/scheduled runs we prefer to fail early if configs are
	// missing or malformed rather than doing partial work with unclear outcomes.
	// -----------------------------------------------------------------------------
	pathConfigs, backupLocation, err := config.ReadAllConfig(cfg.ConfigDir, log)
	if err != nil {
		return err
	}

	// Log paths and their backup settings.
	for _, pc := range pathConfigs {
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
	for _, pc := range pathConfigs {
		if pc.Backup {
			anyBackupEnabled = true
			break
		}
	}

	if anyBackupEnabled {
		log.Infof("Backup location: %s", backupLocation)

		// Safety check:
		// Ensure the destination is reachable (especially on SMB shares) to avoid
		// the dangerous case of deleting source files without successfully copying
		// them somewhere safe first.
		if !maintenance.CheckBackupPath(backupLocation) {
			errMsg := fmt.Sprintf("Backup path is not accessible: %s\n\nPlease check path and permissions.", backupLocation)
			// Show popup notification for the user
			utils.ShowPopup("Backup Location Error", errMsg)

			return fmt.Errorf("backup path not accessible: %s", backupLocation)
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
	if err := maintenance.Worker(pathConfigs, backupLocation, cfg, log); err != nil {
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
