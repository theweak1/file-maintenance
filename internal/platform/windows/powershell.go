package windows

import "os/exec"

func runPowerShellDetached(command string) error {
	cmd := exec.Command("powershell.exe",
		"-NoProfile",
		"-WindowStyle", "Hidden",
		"-Command",
		command)
	return cmd.Start()
}
