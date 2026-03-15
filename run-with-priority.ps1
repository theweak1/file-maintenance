# PowerShell script to run file-maintenance.exe with specified priority
# Usage: .\run-with-priority.ps1 -Priority BelowNormal

param(
	[ValidateSet('Idle', 'BelowNormal', 'Normal', 'AboveNormal', 'High', 'RealTime')]
	[string]$Priority = 'BelowNormal',
    
	[string]$ExePath = ".\file-maintenance.exe",
    
	[string]$Arguments = ""
)

# Get the directory where this script is located
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

# Build the full path to the executable
if (Test-Path (Join-Path $ScriptDir $ExePath)) {
	$FullExePath = Join-Path $ScriptDir $ExePath
}
else {
	$FullExePath = $ExePath
}

# Create process start info
$psi = New-Object System.Diagnostics.ProcessStartInfo
$psi.FileName = $FullExePath
$psi.Arguments = $Arguments
$psi.UseShellExecute = $false

# Set priority based on parameter
switch ($Priority) {
	'Idle' { $psi.PriorityClass = [System.Diagnostics.ProcessPriorityClass]::Idle }
	'BelowNormal' { $psi.PriorityClass = [System.Diagnostics.ProcessPriorityClass]::BelowNormal }
	'Normal' { $psi.PriorityClass = [System.Diagnostics.ProcessPriorityClass]::Normal }
	'AboveNormal' { $psi.PriorityClass = [System.Diagnostics.ProcessPriorityClass]::AboveNormal }
	'High' { $psi.PriorityClass = [System.Diagnostics.ProcessPriorityClass]::High }
	'RealTime' { $psi.PriorityClass = [System.Diagnostics.ProcessPriorityClass]::RealTime }
}

# Start the process
$process = [System.Diagnostics.Process]::Start($psi)

# Wait for the process to complete
$process.WaitForExit()

# Exit with the same exit code as the process
exit $process.ExitCode
