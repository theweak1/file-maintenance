package version

import "testing"

func TestVersion_Table(t *testing.T) {
	originalVersion := Version
	originalCommit := Commit
	originalBuildDate := BuildDate

	t.Cleanup(func() {
		Version = originalVersion
		Commit = originalCommit
		BuildDate = originalBuildDate
	})

	tests := []struct {
		name        string
		version     string
		versionType string
		commit      string
		buildDate   string
		want        string
	}{
		{
			name:        "default values with longVersion",
			version:     "dev",
			versionType: "longVersion",
			commit:      "unknown",
			buildDate:   "unknown",
			want:        "dev commit=unknown built=unknown",
		},
		{
			name:        "release values with longVersion",
			version:     "v0.1.0",
			versionType: "longVersion",
			commit:      "abc123def456",
			buildDate:   "2026-06-20T18:45:00Z",
			want:        "v0.1.0 commit=abc123def456 built=2026-06-20T18:45:00Z",
		},
		{
			name:        "dev build with short commit and longVersion",
			version:     "dev-abc1234",
			versionType: "longVersion",
			commit:      "abc1234",
			buildDate:   "2026-06-20T19:00:00Z",
			want:        "dev-abc1234 commit=abc1234 built=2026-06-20T19:00:00Z",
		},
		{
			name:        "dev build with short commit and shortVersion",
			version:     "dev-abc1234",
			versionType: "shortVersion",
			commit:      "abc1234",
			buildDate:   "2026-06-20T19:00:00Z",
			want:        "dev-abc1234",
		},
		{
			name:        "release values with shortVersion",
			version:     "v0.1.0",
			versionType: "shortVersion",
			commit:      "abc123def456",
			buildDate:   "2026-06-20T18:45:00Z",
			want:        "v0.1.0",
		},
		{
			name:        "default values with shortVersion",
			version:     "dev",
			versionType: "shortVersion",
			commit:      "unknown",
			buildDate:   "unknown",
			want:        "dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Version = tt.version
			Commit = tt.commit
			BuildDate = tt.buildDate

			var got string
			if tt.versionType == "longVersion" {
				got = LongVersion()
			} else {
				got = ShortVersion()
			}

			if got != tt.want {
				t.Fatalf("%s() = %q, want %q", tt.versionType, got, tt.want)
			}
		})
	}
}
