package setup

import (
	"strings"
	"testing"
)

func TestEmbeddedSetupScriptLoadsExistingConfig(t *testing.T) {
	markers := []string{
		"function Read-IniFile",
		"function Load-ExistingConfiguration",
		"Load-ExistingConfiguration",
		"Add-PathRow -Path $path -Backup $backupEnabled",
		"Convert-DurationValue -Value $existingConfig[\"advanced\"][\"cooldown\"] -TargetUnit \"Milliseconds\"",
		"Convert-DurationValue -Value $existingConfig[\"advanced\"][\"max-runtime\"] -TargetUnit \"Minutes\"",
	}

	for _, marker := range markers {
		if !strings.Contains(setupScript, marker) {
			t.Fatalf("embedded setup script is missing %q", marker)
		}
	}
}
