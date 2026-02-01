package maintenance

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckBackupPath_Table(t *testing.T) {
	// CheckBackupPath is a safety gate used before any destructive operations:
	// - true only for existing directories
	// - false for missing paths or non-directory paths (e.g., a file)
	root := t.TempDir()

	validDir := filepath.Join(root, "valid")
	if err := os.MkdirAll(validDir, 0o755); err != nil {
		t.Fatalf("setup: mkdir validDir: %v", err)
	}

	aFile := filepath.Join(root, "file.txt")
	if err := os.WriteFile(aFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: write aFile: %v", err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "valid directory", path: validDir, want: true},
		{name: "path does not exist", path: filepath.Join(root, "missing"), want: false},
		{name: "path is a file", path: aFile, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckBackupPath(tt.path)
			if got != tt.want {
				t.Fatalf("want %v, got %v", tt.want, got)
			}
		})
	}
}

type fakeFileInfo struct{ mt time.Time }

func (f fakeFileInfo) Name() string       { return "x" }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return 0 }
func (f fakeFileInfo) ModTime() time.Time { return f.mt }
func (f fakeFileInfo) IsDir() bool        { return false }
func (f fakeFileInfo) Sys() any           { return nil }

func TestIsFileOlder_Table(t *testing.T) {
	// IsFileOlder uses time.Now() internally, so tests must avoid the exact cutoff
	// boundary (equal timestamps can be flaky due to time.Now() resolution / jitter).
	now := time.Now()

	tests := []struct {
		name string
		mt   time.Time
		days int
		want bool
	}{
		{"old file", now.AddDate(0, 0, -10), 5, true},
		{"recent file", now.AddDate(0, 0, -2), 5, false},

		// Cutoff behavior:
		// cutoff := time.Now().AddDate(0, 0, -days)
		// returns mt.Before(cutoff)
		{"just newer than cutoff", now.AddDate(0, 0, -5).Add(1 * time.Second), 5, false},
		{"just older than cutoff", now.AddDate(0, 0, -5).Add(-1 * time.Second), 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := fakeFileInfo{mt: tt.mt}
			got := IsFileOlder(info, tt.days)
			if got != tt.want {
				t.Fatalf("want %v, got %v", tt.want, got)
			}
		})
	}
}
