package maintenance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"file-maintenance/internal/logging"
	"file-maintenance/internal/types"
)

// Integration test helpers
//
// These helpers create real files/directories under t.TempDir() and are intended
// for Worker integration tests (not unit tests). They keep test bodies small and
// make failures easier to read.

// --- Sandbox / Setup ---

// newSandbox creates an isolated filesystem sandbox with:
//
//	<temp>/source
//	<temp>/backup
//
// Both directories are created before returning.
func newSandbox(t *testing.T) (root, src, backup string) {
	t.Helper()

	root = t.TempDir()
	src = filepath.Join(root, "source")
	backup = filepath.Join(root, "backup")

	mustMkdirAll(t, src)
	mustMkdirAll(t, backup)

	return root, src, backup
}

// newTestCfgAndLogger returns a baseline AppConfig and a real logger instance.
//
// Notes:
// - Uses a log directory under the sandbox root.
// - Relies on logger defaults if logging.json is not present in ConfigDir.
// - Tests can mutate returned cfg fields (NoBackup, Days, MaxFiles, etc.).
func newTestCfgAndLogger(t *testing.T, root string) (types.AppConfig, *logging.Logger) {
	t.Helper()

	logDir := filepath.Join(root, "logs")

	cfg := types.AppConfig{
		Days:      5,
		NoBackup:  false,
		Walkers:   1,
		QueueSize: 50,
		Retries:   0,
		ConfigDir: root,
		LogSettings: logging.LogSettings{
			NoLogs: false,
			LogDir: logDir,
		},
	}

	log, err := logging.New(cfg.ConfigDir, cfg.LogSettings)
	if err != nil {
		t.Fatalf("logging.New failed: %v", err)
	}
	return cfg, log
}

// --- File helpers ---

// mustMkdirAll creates directories and fails the test immediately on error.
func mustMkdirAll(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", p, err)
	}
}

// mustWriteFile writes a file and fails the test immediately on error.
func mustWriteFile(t *testing.T, p string, contents string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %q: %v", p, err)
	}
}

// mustSetAgeDays sets file timestamps so Worker can treat it as "old" or "recent".
func mustSetAgeDays(t *testing.T, p string, ageDays int) {
	t.Helper()
	mt := time.Now().AddDate(0, 0, -ageDays)
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatalf("chtimes %q: %v", p, err)
	}
}

// --- Assertions ---

// assertExists fails the test if p does not exist (or is inaccessible).
func assertExists(t *testing.T, p string) {
	t.Helper()
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("expected file to exist: %q (err=%v)", p, err)
	}
}

// assertNotExists fails the test if p exists (or errors for reasons other than not-exist).
func assertNotExists(t *testing.T, p string) {
	t.Helper()
	_, err := os.Stat(p)
	if !os.IsNotExist(err) {
		t.Fatalf("expected file to NOT exist: %q (stat err=%v)", p, err)
	}
}

// countBackupsWithBase counts how many files under backupRoot have the given base name.
// (This searches all date folders.)
func countBackupsWithBase(t *testing.T, backupRoot string, base string) int {
	t.Helper()

	count := 0
	_ = filepath.WalkDir(backupRoot, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if filepath.Base(p) == base {
			count++
		}
		return nil
	})
	return count
}

// anyBackupHasSuffix checks whether any file under backupRoot has a relative path that
// ends with the provided suffix.
//
// This is useful because Worker writes into a date folder, so the full backup path is:
//
//	<backupRoot>/<dateFolder>/<suffix>
func anyBackupHasSuffix(t *testing.T, backupRoot string, suffix string) bool {
	t.Helper()

	suffix = filepath.Clean(suffix)

	found := false
	_ = filepath.WalkDir(backupRoot, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(backupRoot, p)
		if rerr != nil {
			return nil
		}
		// rel is: <dateFolder>\<suffix>
		rel = filepath.Clean(rel)

		// Portable "ends with" check for path segments.
		if strings.HasSuffix(rel, suffix) {
			found = true
		}
		return nil
	})
	return found
}

// countNonDirFiles counts only immediate (top-level) non-directory entries in dir.
func countNonDirFiles(t *testing.T, dir string) int {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir %q: %v", dir, err)
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() {
			n++
		}
	}
	return n
}

// countFilesWithPrefixSuffix counts files under root whose base name matches prefix/suffix.
// Useful for "file0.txt..fileN.txt" style integration tests.
func countFilesWithPrefixSuffix(t *testing.T, root, prefix, suffix string) int {
	t.Helper()

	n := 0
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		base := filepath.Base(p)
		if strings.HasPrefix(base, prefix) && strings.HasSuffix(base, suffix) {
			n++
		}
		return nil
	})
	return n
}
