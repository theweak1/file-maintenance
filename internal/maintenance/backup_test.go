package maintenance

import (
	"path/filepath"
	"testing"
	"time"
)

func TestBuildBackupPath_Table(t *testing.T) {
	// buildBackupPath() uses a date-based folder. We compute the expected
	// date folder using the same format as production code to keep this
	// test stable within a single run.
	dateFolder := time.Now().Format("02Jan06")

	tests := []struct {
		name       string
		backupRoot string
		folderRoot string
		srcPath    string
		want       string
		wantErr    bool
	}{
		{
			name:       "adds date folder + preserves relative structure",
			backupRoot: filepath.Join("D:", "backup"),
			folderRoot: filepath.Join("C:", "data"),
			srcPath:    filepath.Join("C:", "data", "sub", "file.txt"),
			want:       filepath.Join("D:", "backup", dateFolder, "data", "sub", "file.txt"),
		},
		{
			name:       "src outside folder root may produce '..' via filepath.Rel",
			backupRoot: filepath.Join("D:", "backup"),
			folderRoot: filepath.Join("C:", "data"),
			srcPath:    filepath.Join("C:", "other", "file.txt"),

			// NOTE:
			// buildBackupPath() is a path-construction helper; it does not enforce
			// "must be under folderRoot" safety. If callers need to prevent path
			// traversal / escaping, that must be enforced at a higher level (e.g.,
			// by validating rel paths or using the guarded backupDestPath logic).
			want: filepath.Join("D:", "backup", dateFolder, "other", "file.txt"),
		},
		{
			name:       "error on different drives",
			backupRoot: filepath.Join("D:", "backup"),
			folderRoot: filepath.Join("C:", "data"),
			srcPath:    filepath.Join("D:", "other", "file.txt"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildBackupPath(tt.backupRoot, tt.folderRoot, tt.srcPath)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (got=%q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("want %q, got %q", tt.want, got)
			}
		})
	}
}

func TestBackoffForAttempt_Table(t *testing.T) {
	tests := []struct {
		name    string
		attempt int
		want    time.Duration
	}{
		{"first", 0, 250 * time.Millisecond},
		{"second", 1, 1 * time.Second},
		{"third", 2, 3 * time.Second},
		{"beyond", 10, 3 * time.Second},
		{"negative", -1, 3 * time.Second},
		{"large", 100, 3 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := backoffForAttempt(tt.attempt)
			if got != tt.want {
				t.Fatalf("want %v, got %v", tt.want, got)
			}
		})
	}
}
