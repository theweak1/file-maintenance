// internal/platform/macos/command.go
package macos

import "os/exec"

func runCommandDetached(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Start()
}
