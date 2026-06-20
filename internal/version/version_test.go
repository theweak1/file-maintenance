package version

import "testing"

func TestString_Table(t *testing.T) {
	originalVersion := Version
	originalCommit := Commit
	originalBuildDate := BuildDate

	t.Cleanup(func() {
		Version = originalVersion
		Commit = originalCommit
		BuildDate = originalBuildDate
	})

	tests := []struct {
		name      string
		version   string
		commit    string
		buildDate string
		want      string
	}{
		{
			name:      "default values",
			version:   "dev",
			commit:    "unknown",
			buildDate: "unknown",
			want:      "dev commit=unknown built=unknown",
		},
		{
			name:      "release values",
			version:   "v0.1.0",
			commit:    "abc123def456",
			buildDate: "2026-06-20T18:45:00Z",
			want:      "v0.1.0 commit=abc123def456 built=2026-06-20T18:45:00Z",
		},
		{
			name:      "dev build with short commit",
			version:   "dev-abc1234",
			commit:    "abc1234",
			buildDate: "2026-06-20T19:00:00Z",
			want:      "dev-abc1234 commit=abc1234 built=2026-06-20T19:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Version = tt.version
			Commit = tt.commit
			BuildDate = tt.buildDate

			got := String()

			if got != tt.want {
				t.Fatalf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}
