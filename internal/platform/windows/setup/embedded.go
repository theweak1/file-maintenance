package setup

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"file-maintenance/internal/types"
)

//go:embed setup_script.gohtml
var setupScript string

// LaunchEmbeddedSetup launches the embedded setup wizard.
func LaunchEmbeddedSetup(configDir, exeDir string) (types.SetupAction, error) {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return types.SetupActionCancelled, fmt.Errorf("failed to create config directory: %w", err)
	}

	scriptPath := filepath.Join(configDir, "setup_temp.ps1")
	if err := os.WriteFile(scriptPath, []byte(setupScript), 0644); err != nil {
		return types.SetupActionCancelled, fmt.Errorf("failed to write setup script: %w", err)
	}
	defer os.Remove(scriptPath)

	cmd := powershellCommand("-ExecutionPolicy", "Bypass", "-File", scriptPath, "-ConfigDir", configDir)
	cmd.Dir = configDir

	exitCode := 0
	if err := cmd.Run(); err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			return types.SetupActionCancelled, fmt.Errorf("failed to launch setup wizard: %w", err)
		}
		exitCode = exitErr.ExitCode()
	}

	action, ok := SetupActionFromExitCode(exitCode)
	if !ok {
		return types.SetupActionCancelled, fmt.Errorf("setup wizard exited with unexpected code %d", exitCode)
	}

	if action != types.SetupActionCancelled && !ConfigExists(configDir) {
		return types.SetupActionCancelled, fmt.Errorf("setup completed but config.ini was not created")
	}

	return action, nil
}

// powershellCommand returns a command to run PowerShell.
func powershellCommand(args ...string) *exec.Cmd {
	cmd := exec.Command("powershell.exe", args...)
	cmd.Env = os.Environ()
	return cmd
}
