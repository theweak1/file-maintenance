package app

import (
	"fmt"

	"file-maintenance/internal/config"
	"file-maintenance/internal/logging"
	"file-maintenance/internal/maintenance"
	"file-maintenance/internal/types"
)

func Run(cfg types.AppConfig, log *logging.Logger) error {
	// -----------------------------------------------------------------------------
	// Read list of folders to process.
	//
	// This is the "input set" for the worker. For unattended/scheduled runs we
	// prefer to fail early if configs are missing or malformed rather than doing
	// partial work with unclear outcomes.
	// -----------------------------------------------------------------------------
	folders, err := config.ReadFolderList(cfg.ConfigDir)
	if err != nil {
		return err
	}
	log.Infof("Folders to process: %v", folders)

	// -----------------------------------------------------------------------------
	// Determine backup location (only when backups are enabled).
	//
	// When backups are ON:
	// - Read backup root from config (e.g., configs/backup.txt).
	// - Validate the path is accessible (important for network shares).
	// - If the backup path is not accessible, abort BEFORE any deletions occur.
	//
	// When backups are OFF:
	// - Skip reading/validating the backup location.
	// - Worker will run in "delete only" mode.
	// -----------------------------------------------------------------------------
	backupLocation := ""
	if cfg.NoBackup {
		log.Warn("Backup is disabled")
	} else {
		backupLocation, err = config.ReadBackupLocation(cfg.ConfigDir)
		if err != nil {
			return err
		}

		// Safety check:
		// Ensure the destination is reachable (especially on SMB shares) to avoid
		// the dangerous case of deleting source files without successfully copying
		// them somewhere safe first.
		if !maintenance.CheckBackupPath(backupLocation) {
			// Fatalf is expected to log at a fatal level. We also return an error so
			// the caller gets a non-zero exit code even if Fatalf does not exit.
			log.Fatalf("Backup path is not accessible: %s", backupLocation)
			return fmt.Errorf("backup path not accessible: %s", backupLocation)
		}

		log.Infof("Backup location: %s", backupLocation)
	}

	// -----------------------------------------------------------------------------
	// Run the worker.
	//
	// Worker responsibilities (high-level contract):
	// - Scan configured folders (optionally concurrent / bounded by cfg.Walkers).
	// - Process only files older than cfg.Days.
	// - If backups are enabled: copy to backupLocation before deleting.
	// - If backups are disabled: delete without copying.
	//
	// Important:
	// - We must return Worker errors; otherwise failures are invisible to callers
	//   and Task Scheduler exit codes.
	// -----------------------------------------------------------------------------
	if err := maintenance.Worker(folders, backupLocation, cfg, log); err != nil {
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
