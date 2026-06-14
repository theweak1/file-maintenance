// internal/platform/linux/notification.go
package linux

import (
	"fmt"
	"os"
	"os/exec"
)

func (Platform) ShowCritical(title, message string) {
	if _, err := exec.LookPath("notify-send"); err == nil {
		_ = runCommandDetached("notify-send", title, message)
		return
	}

	_, _ = fmt.Fprintf(os.Stderr, "CRITICAL [%s]: %s\n", title, message)
}
