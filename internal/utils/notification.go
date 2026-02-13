package utils

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// ShowPopup displays a message box popup notification to alert the user.
//
// This function is primarily used for critical errors such as inaccessible
// backup locations, ensuring users are notified even when running unattended
// via Task Scheduler.
//
// Platform support:
//   - Windows: Uses PowerShell to show a native Windows message box with OK button
//   - Other platforms: Falls back to printing to stderr (CLI context)
//
// Non-blocking behavior:
//
//	On Windows, the popup is launched as a separate background process using
//	PowerShell. This allows the message to appear even after the application
//	terminates (e.g., due to Fatal log or os.Exit).
//
// Parameters:
//   - title: The popup window title (e.g., "Backup Location Error")
//   - message: The message to display in the popup body
//
// Example:
//
//	utils.ShowPopup("Backup Location Error", "Cannot access backup path: \\server\share\backups")
func ShowPopup(title, message string) {
	switch runtime.GOOS {
	case "windows":
		showWindowsPopup(title, message)
	default:
		// For non-Windows platforms, print to stderr as fallback
		// since there's no native popup mechanism in a CLI context
		_, _ = os.Stderr.Write([]byte("POPUP [" + title + "]: " + message + "\n"))
	}
}

// showWindowsPopup uses PowerShell to display a native Windows message box.
// This works even when running as a console application or from Task Scheduler.
//
// The command uses Add-Type to load System.Windows.Forms and calls MessageBox::Show
// with an Error icon to indicate a critical issue.
//
// Implementation notes:
//   - WindowStyle Hidden prevents the PowerShell window from briefly appearing
//   - NoProfile speeds up execution by skipping profile scripts
//   - cmd.Start() is used (not cmd.Run()) to avoid blocking the calling process
func showWindowsPopup(title, message string) {
	// Escape quotes in the message and title for PowerShell
	escapedTitle := strings.ReplaceAll(title, `"`, "`\"")
	escapedMessage := strings.ReplaceAll(message, `"`, "`\"")

	// PowerShell command to show a message box with OK button
	// -WindowStyle Hidden hides the PowerShell window briefly
	// -ArgumentList passes arguments to the script
	powershell := `powershell`
	args := []string{
		"-WindowStyle", "Hidden",
		"-NoProfile",
		"-Command",
		`Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.MessageBox]::Show("` + escapedMessage + `", "` + escapedTitle + `", [System.Windows.Forms.MessageBoxButtons]::OK, [System.Windows.Forms.MessageBoxIcon]::Error)`,
	}

	cmd := exec.Command(powershell, args...)

	// Execute without waiting for completion
	// This allows the popup to appear while the app continues or exits
	_ = cmd.Start()
}
