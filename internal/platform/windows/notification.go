// internal/platform/windows/notification.go
package windows

import (
	"strings"
)

func (Platform) ShowCritical(title, message string) {
	// Escape quotes in the message and title for PowerShell
	escapedTitle := escapePowerShellString(title)
	escapedMessage := escapePowerShellString(message)

	command :=
		`Add-Type -AssemblyName System.Windows.Forms; ` +
			`[System.Windows.Forms.MessageBox]::Show("` +
			escapedMessage +
			`", "` +
			escapedTitle +
			`", [System.Windows.Forms.MessageBoxButtons]::OK, [System.Windows.Forms.MessageBoxIcon]::Error)`

	_ = runPowerShellDetached(command)
}

// escapePowerShellString escapes double quotes in a string for safe inclusion in a PowerShell command.
func escapePowerShellString(s string) string {
	return strings.ReplaceAll(s, `"`, "`\"")
}
