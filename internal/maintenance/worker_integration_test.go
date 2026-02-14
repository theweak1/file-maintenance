package maintenance

import (
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"file-maintenance/internal/types"
)

// Worker integration tests
//
// These tests use real filesystem I/O under t.TempDir(). They validate end-to-end
// behavior of Worker (backup + delete + structure preservation + resource limits).
//
// Note: Worker writes backups under a date folder, so backup assertions scan the
// backupRoot tree rather than checking an exact path.

func TestWorker_Integration_Table(t *testing.T) {
	tests := []struct {
		name           string
		fileAgeDays    int
		backupEnabled  bool
		expectDeleted  bool
		expectBackedUp bool
	}{
		{"old file + backup", 10, true, true, true},
		{"old file + no backup", 10, false, true, false},
		{"recent file untouched", 1, true, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, src, backup := newSandbox(t)
			cfg, log := newTestCfgAndLogger(t, root)

			cfg.Days = 5
			cfg.Walkers = 1
			cfg.QueueSize = 10
			cfg.Retries = 0

			target := filepath.Join(src, "test.txt")
			mustWriteFile(t, target, "hello")
			mustSetAgeDays(t, target, tt.fileAgeDays)

			pathconfig := []types.PathConfig{
				{Path: src, Backup: tt.backupEnabled, IsDir: true},
			}

			if err := Worker(pathconfig, backup, cfg, log); err != nil {
				t.Fatalf("worker error: %v", err)
			}

			if tt.expectDeleted {
				assertNotExists(t, target)
			} else {
				assertExists(t, target)
			}

			// Backup assertion:
			// Worker stores backups under <backupRoot>/<dateFolder>/..., so scan the tree.
			found := countBackupsWithBase(t, backup, "test.txt") > 0
			if tt.expectBackedUp && !found {
				t.Fatalf("expected backup, but not found")
			}
			if !tt.expectBackedUp && found {
				t.Fatalf("expected no backup, but found one")
			}
		})
	}
}

func TestWorker_Integration_MultiFiles_Table(t *testing.T) {
	tests := []struct {
		name              string
		backupEnabled     bool
		expectOldBackedUp bool
	}{
		{"backup enabled", true, true},
		{"no backup", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, src, backup := newSandbox(t)
			cfg, log := newTestCfgAndLogger(t, root)

			cfg.Days = 5

			oldPath := filepath.Join(src, "old.txt")
			newPath := filepath.Join(src, "new.txt")

			mustWriteFile(t, oldPath, "old")
			mustWriteFile(t, newPath, "new")

			mustSetAgeDays(t, oldPath, 10) // old enough => candidate
			mustSetAgeDays(t, newPath, 1)  // too new => must remain

			pathconfig := []types.PathConfig{
				{Path: src, Backup: tt.backupEnabled, IsDir: true},
			}

			if err := Worker(pathconfig, backup, cfg, log); err != nil {
				t.Fatalf("worker error: %v", err)
			}

			assertNotExists(t, oldPath)
			assertExists(t, newPath)

			oldBackups := countBackupsWithBase(t, backup, "old.txt")
			newBackups := countBackupsWithBase(t, backup, "new.txt")

			if tt.expectOldBackedUp && oldBackups == 0 {
				t.Fatalf("expected old.txt backup, but none found")
			}
			if !tt.expectOldBackedUp && oldBackups > 0 {
				t.Fatalf("expected old.txt NOT backed up, but found %d", oldBackups)
			}
			if newBackups > 0 {
				t.Fatalf("expected new.txt NOT backed up, but found %d", newBackups)
			}
		})
	}
}

func TestWorker_Integration_NestedFolders_Table(t *testing.T) {
	tests := []struct {
		name           string
		backupEnabled  bool
		expectBackedUp bool
	}{
		{"backup enabled", true, true},
		{"no backup", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, src, backup := newSandbox(t)
			cfg, log := newTestCfgAndLogger(t, root)

			cfg.Days = 5

			nestedDir := filepath.Join(src, "sub", "deep")
			mustMkdirAll(t, nestedDir)

			target := filepath.Join(nestedDir, "old.txt")
			mustWriteFile(t, target, "Hello")
			mustSetAgeDays(t, target, 10)

			pathconfig := []types.PathConfig{
				{Path: src, Backup: tt.backupEnabled, IsDir: true},
			}

			if err := Worker(pathconfig, backup, cfg, log); err != nil {
				t.Fatalf("worker error: %v", err)
			}

			assertNotExists(t, target)

			// We don't care about the date folder; we only care that the relative suffix exists.
			suffix := filepath.Join("sub", "deep", "old.txt")
			found := anyBackupHasSuffix(t, backup, suffix)

			if tt.expectBackedUp && !found {
				t.Fatalf("expected backup to contain suffix %q", suffix)
			}
			if !tt.expectBackedUp && found {
				t.Fatalf("expected no backup, but found suffix %q", suffix)
			}
		})
	}
}

func TestWorker_Integration_MultipleFolders(t *testing.T) {
	tests := []struct {
		name          string
		backupEnabled bool
	}{
		{"backup enabled", true},
		{"no backup", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()

			src1 := filepath.Join(root, "source1")
			src2 := filepath.Join(root, "source2")
			backup := filepath.Join(root, "backup")

			mustMkdirAll(t, src1)
			mustMkdirAll(t, src2)
			mustMkdirAll(t, backup)

			cfg, log := newTestCfgAndLogger(t, root)
			cfg.Days = 5

			f1 := filepath.Join(src1, "a.txt")
			f2 := filepath.Join(src2, "b.txt")
			mustWriteFile(t, f1, "a")
			mustWriteFile(t, f2, "b")
			mustSetAgeDays(t, f1, 10)
			mustSetAgeDays(t, f2, 10)

			pathconfig := []types.PathConfig{
				{Path: src1, Backup: tt.backupEnabled, IsDir: true},
				{Path: src2, Backup: tt.backupEnabled, IsDir: true},
			}

			if err := Worker(pathconfig, backup, cfg, log); err != nil {
				t.Fatalf("worker error: %v", err)
			}

			assertNotExists(t, f1)
			assertNotExists(t, f2)
		})
	}
}

func TestWorker_Integration_MaxFiles_Table(t *testing.T) {
	// MaxFiles is a stop condition and may not be perfectly exact in the presence of
	// concurrency / buffering. These tests assert an upper bound (<=) rather than an exact count.
	tests := []struct {
		name              string
		backupEnabled     bool
		maxFiles          int
		totalOldFiles     int
		expectDeletedMax  int // at most this many should be deleted
		expectBackedUpMax int // at most this many backups should exist (when backup enabled)
	}{
		{"backup enabled stops at 1", true, 1, 3, 1, 1},
		{"no-backup stops at 1", false, 1, 3, 1, 0},
		{"backup enabled stops at 2", true, 2, 5, 2, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, src, backup := newSandbox(t)
			cfg, log := newTestCfgAndLogger(t, root)

			cfg.Days = 5
			cfg.MaxFiles = tt.maxFiles
			cfg.Walkers = 1
			cfg.QueueSize = 10
			cfg.Retries = 0

			// Create N old files.
			for i := 0; i < tt.totalOldFiles; i++ {
				p := filepath.Join(src, "file"+strconv.Itoa(i)+".txt")
				mustWriteFile(t, p, "x")
				mustSetAgeDays(t, p, 10)
			}

			pathconfig := []types.PathConfig{
				{Path: src, Backup: tt.backupEnabled, IsDir: true},
			}

			if err := Worker(pathconfig, backup, cfg, log); err != nil {
				t.Fatalf("worker error: %v", err)
			}

			// Deletions: total - remaining.
			remaining := countNonDirFiles(t, src)
			deleted := tt.totalOldFiles - remaining
			if deleted > tt.expectDeletedMax {
				t.Fatalf("expected deleted <= %d, got %d (remaining=%d total=%d)",
					tt.expectDeletedMax, deleted, remaining, tt.totalOldFiles)
			}

			// Backups: count file*.txt occurrences under backup.
			backedUp := countFilesWithPrefixSuffix(t, backup, "file", ".txt")
			if backedUp > tt.expectBackedUpMax {
				t.Fatalf("expected backedUp <= %d, got %d", tt.expectBackedUpMax, backedUp)
			}
		})
	}
}

func TestWorker_Integration_MaxRuntime_StopsEarly(t *testing.T) {
	// This test validates that MaxRuntime behaves as a stop condition (best-effort).
	// It is inherently timing-sensitive, so the assertions are intentionally broad.
	root, src, backup := newSandbox(t)
	cfg, log := newTestCfgAndLogger(t, root)

	cfg.Days = 5
	cfg.MaxRuntime = 10 * time.Millisecond
	cfg.Walkers = 1
	cfg.QueueSize = 10

	total := 200
	for i := 0; i < total; i++ {
		p := filepath.Join(src, "f"+strconv.Itoa(i)+".txt")
		mustWriteFile(t, p, "x")
		mustSetAgeDays(t, p, 10)
	}

	pathconfig := []types.PathConfig{
		{Path: src, Backup: false, IsDir: true},
	}

	if err := Worker(pathconfig, backup, cfg, log); err != nil {
		t.Fatalf("worker error: %v", err)
	}

	remaining := countNonDirFiles(t, src)

	if remaining == 0 {
		t.Fatalf("expected some files to remain due to MaxRuntime, but none remain (processed all)")
	}
	if remaining == total {
		t.Fatalf("expected at least 1 file processed, but none were processed")
	}
}
