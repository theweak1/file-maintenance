// internal/platform/linux/command.go
package linux

import "os/exec"

func runCommandDetached(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Start()
}
