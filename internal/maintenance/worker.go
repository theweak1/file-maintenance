package maintenance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"file-maintenance/internal/logging"
	"file-maintenance/internal/types"
)

// FileJob represents a single candidate file selected for processing.
//
// Concurrency model:
//
// - Folder walking (discovering candidate files) can be concurrent.
// - File operations (backup + delete) are intentionally serialized by ONE processor goroutine.
//
// Why serialize file operations?
// - Prevent hammering SMB/network shares with concurrent copy/delete.
// - Reduce contention on busy workstations during scheduled runs.
// - Keep bandwidth, CPU, and disk I/O predictable and stable.
type FileJob struct {
	// srcPath is the full path to the candidate file on disk.
	srcPath string

	// folderRoot is the configured top-level folder currently being walked.
	//
	// We keep it for two reasons:
	//  1) buildBackupPath() needs it to compute a relative path and preserve folder structure in backups
	//  2) per-folder deletion counting needs a stable key (folderRoot) for accurate reporting
	folderRoot string

	// backup indicates whether this file should be backed up before deletion.
	// This is set based on the path's configuration in paths.txt.
	backup bool
}

// Worker scans configured folders, selects "old" files (based on cfg.Days),
// optionally backs them up, and then deletes them.
//
// High-level flow:
//  1. Walk folders to discover candidate files (concurrent walkers)
//  2. Queue matching files as jobs (bounded channel provides backpressure)
//  3. Process jobs one-at-a-time (backup then delete) for safety and stability
//
// Concurrency model:
// - Walkers: cfg.Walkers goroutines scan folders concurrently (bounded).
// - Processor: ONE goroutine performs backup+delete sequentially (one file at a time).
//
// Safety guarantee:
//   - A file is deleted ONLY after it is successfully backed up,
//     unless cfg.NoBackup is true (in which case delete happens immediately).
//
// Stop conditions:
// - MaxRuntime: caps total runtime of a run (best-effort).
// - MaxFiles: caps how many *jobs are handled* by the processor (not “files deleted”).
//
// Counting notes:
//   - processed is GLOBAL and increments after each job is handled by the processor
//     (regardless of whether delete succeeded). It exists for stop conditions/reporting.
//   - deletedByFolder is PER-FOLDER and increments ONLY when a delete succeeds.
//   - Per-folder counts are logged AFTER all processing finishes so totals are accurate.
func Worker(pathConfigs []types.PathConfig, backupRoot string, cfg types.AppConfig, log *logging.Logger) error {
	log.Info("Starting maintenance worker")

	// -------------------------------------------------------------------------
	// Per-folder deleted counters
	//
	// Why this exists:
	// - Walking can finish long before processing finishes.
	// - If we log counts in walkers, we'd log too early (partial results).
	//
	// So:
	// - We increment counts in the processor (where deletion actually succeeds).
	// - We log final per-folder totals after the processor exits.
	// -------------------------------------------------------------------------
	var (
		perFolderMu     sync.Mutex
		deletedByFolder = make(map[string]uint64) // key: folderRoot, value: successful deletions
	)

	// -------------------------------------------------------------------------
	// Defaults / defensive config normalization
	//
	// These defaults are intentionally conservative to protect scheduled runs.
	// -------------------------------------------------------------------------
	if cfg.Walkers <= 0 {
		cfg.Walkers = 1
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 300
	}
	if cfg.Retries < 0 {
		cfg.Retries = 0
	}

	// Track start time for MaxRuntime enforcement.
	start := time.Now()

	// ctx cancels both walkers and the processor.
	//
	// Cancel triggers:
	// - a hard walker failure
	// - stop conditions met (max runtime / max files)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// jobs is a bounded queue of candidate files for the processor.
	//
	// Bounded queue == backpressure:
	// - prevents unbounded memory growth if walking outruns processing
	// - forces walkers to slow down when the processor is busy (especially important for SMB)
	jobs := make(chan FileJob, cfg.QueueSize)

	// processed counts how many jobs the processor handled (global).
	// Used only for stop conditions and end-of-run reporting.
	var processed uint64

	// firstErr stores the first "hard" error encountered during folder walking.
	//
	// Notes:
	// - Walk errors for individual paths are logged and ignored (non-fatal).
	// - A "hard" error is something like WalkDir aborting unexpectedly for a folder.
	// - File operation failures (backup/delete) are logged and generally do not fail the run.
	var firstErr atomic.Value // stores error (firstErr.Store(err))

	// -------------------------------------------------------------------------
	// Stop conditions helper
	//
	// shouldStop() is consulted by BOTH walkers and the processor so the system
	// can stop quickly without extra coordination or complex signaling.
	//
	// Important behavior:
	// - When shouldStop() becomes true, the processor exits early.
	// - Any jobs still buffered in `jobs` will not be processed (intentional).
	// -------------------------------------------------------------------------
	shouldStop := func() bool {
		if cfg.MaxRuntime > 0 && time.Since(start) >= cfg.MaxRuntime {
			return true
		}
		if cfg.MaxFiles > 0 && int(atomic.LoadUint64(&processed)) >= cfg.MaxFiles {
			return true
		}
		return false
	}

	// -------------------------------------------------------------------------
	// Processor goroutine (single-threaded file operations)
	//
	// Reads jobs from the channel and, for each job:
	//  1) builds backup destination path (dated folder + preserved structure)
	//  2) optionally backs up (with retry/backoff)
	//  3) deletes the source file (only after successful backup, unless NoBackup)
	//  4) increments counters + optionally cleans empty directories
	//
	// This is intentionally one-at-a-time for predictable resource usage.
	// -------------------------------------------------------------------------
	var procWG sync.WaitGroup
	procWG.Add(1)
	go func() {
		defer procWG.Done()

		for job := range jobs {
			// Stop quickly if we hit run limits.
			if shouldStop() {
				log.Info("Stop condition met, halting processing")
				// We do not drain jobs here:
				// - walkers will stop producing as they observe shouldStop/ctx
				// - main will close(jobs) after walkers exit
				return
			}

			// Build destination path:
			// backupRoot/<DDMmmYY>/<relative folder structure>/<filename>
			dstPath, err := buildBackupPath(backupRoot, job.folderRoot, job.srcPath)
			if err != nil {
				log.Errorf("Building backup path failed for %s: %v", job.srcPath, err)
				continue
			}

			// Backup phase (unless disabled for this path):
			// - If destination already exists, skip backup to avoid overwriting.
			// - Otherwise copy with retries/backoff to tolerate transient issues (SMB hiccups, locks, etc.).
			if job.backup {
				if DoesFileExist(dstPath) {
					log.Warnf("File already exists in backup, skipping: %s", dstPath)
				} else {
					if err := copyFileWithRetry(ctx, job.srcPath, dstPath, cfg.Retries, log); err != nil {
						log.Errorf("Backup failed for %s -> %s: %v", job.srcPath, dstPath, err)
						continue // do NOT delete if backup failed
					}
					log.Successf("Backed up: %s -> %s", job.srcPath, dstPath)
				}
			}

			// Delete phase:
			// - Only delete after successful backup (or immediately if NoBackup).
			// - This ordering is the main safety guarantee of the worker.
			if err := DeleteFile(job.srcPath); err != nil {
				log.Errorf("Delete failed for %s: %v", job.srcPath, err)
			} else {
				log.Successf("Deleted: %s", job.srcPath)

				// Per-folder counting:
				// Increment only on successful delete so the count reflects reality.
				perFolderMu.Lock()
				deletedByFolder[job.folderRoot]++
				perFolderMu.Unlock()

				// Optional cleanup: remove now-empty directories bottom-up.
				//
				// Invariants:
				// - must NOT delete above the configured folder root
				// - Windows requires directories to be empty before removing them
				cleanupEmptyDirs(filepath.Dir(job.srcPath), job.folderRoot, log)
			}

			// Global processed count for stop conditions and run reporting.
			atomic.AddUint64(&processed, 1)

			// Optional throttle:
			// - Reduces burst load on SMB/network shares
			// - Helps keep the machine responsive during scheduled runs
			if cfg.Cooldown > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(cfg.Cooldown):
				}
			}
		}
	}()

	// -------------------------------------------------------------------------
	// Folder walkers (bounded concurrency)
	//
	// Walkers discover candidate files and enqueue jobs.
	// They do NOT perform file operations (copy/delete).
	//
	// Why bound walkers?
	// - filepath.WalkDir can generate lots of filesystem metadata I/O
	// - too many concurrent walkers can hammer SMB/network shares
	// - bounding keeps scheduled runs stable and predictable
	// -------------------------------------------------------------------------
	sem := make(chan struct{}, cfg.Walkers)
	var walkWG sync.WaitGroup

	for _, pathConfig := range pathConfigs {
		// Avoid launching new walkers once stop conditions are met.
		if shouldStop() {
			break
		}

		folder := pathConfig.Path
		backupEnabled := pathConfig.Backup

		// Acquire a slot (blocks if cfg.Walkers walkers already running).
		sem <- struct{}{}
		walkWG.Add(1)

		go func() {
			defer walkWG.Done()
			defer func() { <-sem }() // release slot

			// Exit early if canceled.
			if ctx.Err() != nil {
				return
			}

			// Validate path exists.
			fi, err := os.Stat(folder)
			if err != nil {
				log.Errorf("Error accessing path %s: %v", folder, err)
				return
			}

			// Handle file paths directly (not directories).
			// This allows users to specify individual files in folders.txt.
			if !fi.IsDir() {
				// Check if the file is old enough to be deleted.
				if !IsFileOlder(fi, cfg.Days) {
					log.Debugf("File is not old enough, skipping: %s", folder)
					return
				}

				// For file paths, use the parent directory as folderRoot for counting.
				folderRoot := filepath.Dir(folder)

				// Enqueue the file directly for processing.
				select {
				case <-ctx.Done():
					return
				case jobs <- FileJob{srcPath: folder, folderRoot: folderRoot, backup: backupEnabled}:
				}
				log.Infof("Queued file for deletion: %s", folder)
				return
			}

			log.Infof("Processing folder: %s", folder)

			// WalkDir recursively scans the folder.
			//
			// For each file:
			// - evaluate age
			// - enqueue a job to be processed serially
			//
			// NOTE: Enqueue may block if the jobs channel is full (backpressure).
			err = filepath.WalkDir(folder, func(path string, d os.DirEntry, err error) error {
				// Walk errors are logged and ignored (non-fatal).
				if err != nil {
					log.Errorf("Walk error (%s): %v", path, err)
					return nil
				}

				// Directories: keep walking, but allow early cancel/stop.
				if d.IsDir() {
					if ctx.Err() != nil || shouldStop() {
						return context.Canceled
					}
					return nil
				}

				// Files: check stop conditions before doing extra work.
				if ctx.Err() != nil || shouldStop() {
					return context.Canceled
				}

				// Gather metadata to evaluate age.
				info, err := d.Info()
				if err != nil {
					log.Errorf("Info error %s: %v", path, err)
					return nil
				}

				// Skip files that are not older than cfg.Days.
				if !IsFileOlder(info, cfg.Days) {
					return nil
				}

				// Enqueue work for the processor (blocks if queue is full).
				select {
				case <-ctx.Done():
					return context.Canceled
				case jobs <- FileJob{srcPath: path, folderRoot: folder, backup: backupEnabled}:
				}

				return nil
			})

			// WalkDir returns context.Canceled when we cancel early for stop conditions.
			// Any other error here is treated as a "hard" walk failure for this folder.
			if err != nil && err != context.Canceled {
				if firstErr.Load() == nil {
					firstErr.Store(fmt.Errorf("Walk failed for %s: %w", folder, err))
				}
				cancel()
				return
			}

			log.Infof("Finished walking folder: %s", folder)
		}()
	}

	// -------------------------------------------------------------------------
	// Shutdown sequence
	//
	// 1) wait for walkers to finish producing jobs
	// 2) close jobs channel (signals processor to stop once drained)
	// 3) wait for processor to finish
	// 4) log final per-folder deletion counts (now accurate)
	// -------------------------------------------------------------------------
	walkWG.Wait()
	close(jobs)
	procWG.Wait()

	// Accurate per-path reporting happens AFTER processing finishes.
	perFolderMu.Lock()
	for _, pathConfig := range pathConfigs {
		count := deletedByFolder[pathConfig.Path]
		if pathConfig.IsDir {
			log.Countf("Amount of files deleted from folder %s: %d", pathConfig.Path, count)
		} else {
			if count > 0 {
				log.Successf("File deleted: %s", pathConfig.Path)
			}
			// Note: Files that don't meet criteria are logged earlier (during scanning)
		}
	}
	perFolderMu.Unlock()

	// Return the first hard error (if any).
	if v := firstErr.Load(); v != nil {
		return v.(error)
	}

	// End-of-run reporting: helps diagnose why scheduled runs ended early.
	if cfg.MaxRuntime > 0 && time.Since(start) >= cfg.MaxRuntime {
		log.Warnf("Stopped due to max runtime (%s). Jobs handled: %d", cfg.MaxRuntime, atomic.LoadUint64(&processed))
	}
	if cfg.MaxFiles > 0 && int(atomic.LoadUint64(&processed)) >= cfg.MaxFiles {
		log.Warnf("Stopped due to max files (%d). Jobs handled: %d", cfg.MaxFiles, atomic.LoadUint64(&processed))
	}

	return nil
}
