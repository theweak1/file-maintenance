package maintenance

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestBackupDestPath_Table(t *testing.T) {
	// backupDestPath is a safety helper:
	// - It preserves directory structure relative to srcRoot.
	// - It rejects any srcFull that would escape srcRoot (path traversal / wrong root),
	//   returning ErrPathEscapesRoot.
	tests := []struct {
		name       string
		backupRoot string
		srcRoot    string
		srcFull    string
		want       string
		wantErr    bool
	}{
		{
			name:       "preserves structure under srcRoot",
			backupRoot: filepath.Join("Z:", "Backups"),
			srcRoot:    filepath.Join("C:", "Data"),
			srcFull:    filepath.Join("C:", "Data", "Images", "2026", "img.jpg"),
			want:       filepath.Join("Z:", "Backups", "Images", "2026", "img.jpg"),
		},
		{
			name:       "rejects file outside srcRoot",
			backupRoot: filepath.Join("Z:", "Backups"),
			srcRoot:    filepath.Join("C:", "Data"),
			srcFull:    filepath.Join("C:", "Other", "x.txt"),
			wantErr:    true,
		},
		{
			name:       "cleans relative path safely",
			backupRoot: filepath.Join("Z:", "Backups"),
			srcRoot:    filepath.Join("C:", "Data"),
			srcFull:    filepath.Join("C:", "Data", "A", "..", "B", "file.txt"),
			want:       filepath.Join("Z:", "Backups", "B", "file.txt"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := backupDestPath(tt.backupRoot, tt.srcRoot, tt.srcFull)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (got=%q)", got)
				}
				// Contract: escaping srcRoot must be rejected with ErrPathEscapesRoot.
				if !errors.Is(err, ErrPathEscapesRoot) {
					t.Fatalf("expected ErrPathEscapesRoot, got %v", err)
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
