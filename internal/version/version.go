package version

import "fmt"

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func LongVersion() string {
	return fmt.Sprintf("%s commit=%s built=%s", Version, Commit, BuildDate)
}

func ShortVersion() string {
	return fmt.Sprintf("%s", Version)
}
