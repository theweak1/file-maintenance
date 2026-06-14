package setup

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed setup_script.gohtml
var setupScript string

// LaunchEmbeddedSetup launches the embedded setup wizard
func LaunchEmbeddedSetup(configDir, exeDir string) error {
	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create a temporary PowerShell script file
	scriptPath := filepath.Join(configDir, "setup_temp.ps1")

	// Write the embedded script to temp file
	if err := os.WriteFile(scriptPath, []byte(setupScript), 0644); err != nil {
		return fmt.Errorf("failed to write setup script: %w", err)
	}

	// Launch PowerShell with the setup script
	cmd := powershellCommand("-ExecutionPolicy", "Bypass", "-File", scriptPath, "-ConfigDir", configDir)
	cmd.Dir = configDir

	if err := cmd.Run(); err != nil {
		// Clean up temp file
		os.Remove(scriptPath)
		return fmt.Errorf("failed to launch setup wizard: %w", err)
	}

	// Clean up temp file
	os.Remove(scriptPath)

	return nil
}

// powershellCommand returns a command to run PowerShell
func powershellCommand(args ...string) *exec.Cmd {
	cmd := exec.Command("powershell.exe", args...)
	cmd.Env = os.Environ()
	return cmd
}
