// internal/platform/macos/notification.go
package macos

import (
	"fmt"
	"os"
	"strings"
)

func (Platform) ShowCritical(title, message string) {
	safeTitle := strings.ReplaceAll(title, `"`, `\"`)
	safeMessage := strings.ReplaceAll(message, `"`, `\"`)
	script := fmt.Sprintf(`display alert "%s" message "%s" as critical`, safeTitle, safeMessage)

	if err := runCommandDetached("osascript", "-e", script); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "CRITICAL [%s]: %s\n", title, message)
	}
}
