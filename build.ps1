param(
	[ValidateSet("build", "run", "run-nologs", "clean", "smoke", "test", "test-integration", "test-all", "coverage")]
	[string]$Task = "build",

	[int]$Days = 7,

	# Pass-through knobs (optional)
	[switch]$Verbose,
	[switch]$Race
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$binDir = Join-Path $root ".bin"
$exe = Join-Path $binDir "fileMaintenance.exe"
$cmdDir = Join-Path $root "cmd\main"
$configDir = Join-Path $root "configs"
$logDir = Join-Path $root "logs"

function Ensure-Dir([string]$Path) {
	if (-not (Test-Path $Path)) {
		New-Item -ItemType Directory -Path $Path -Force | Out-Null
	}
}

function Require-File([string]$Path) {
	if (-not (Test-Path $Path)) {
		throw "Missing required file: $Path"
	}
}

function Go-TestCommonArgs {
	$args = @("./...")
	if ($Verbose) { $args = @("-v") + $args }
	if ($Race) { $args = @("-race") + $args }
	return $args
}

function Test-Unit {
	Write-Host "Running unit tests..."
	$testArgs = Go-TestCommonArgs
	go test @testArgs
}

function Test-Integration {
	Write-Host "Running integration tests..."
	$testArgs = @("-tags=integration") + (Go-TestCommonArgs)
	go test @testArgs
}

function Test-All {
	Test-Unit
	Test-Integration
}

function Coverage {
	Ensure-Dir $binDir
	$covFile = Join-Path $binDir "coverage.out"
	$covHtml = Join-Path $binDir "coverage.html"

	Write-Host "Running coverage -> $covFile"

	$testArgs = @("-coverprofile=$covFile") + (Go-TestCommonArgs)
	go test @testArgs

	if (-not (Test-Path $covFile)) {
		throw "Coverage file not found: $covFile"
	}

	Write-Host "Generating HTML -> $covHtml"
	$coverArgs = @("-html=$covFile", "-o=$covHtml")
	go tool cover @coverArgs

	Write-Host "Coverage report: $covHtml"
}


function Build {
	Ensure-Dir $binDir
	Write-Host "Building -> $exe"
	go build -o $exe $cmdDir
}

function Run([switch]$NoLogs) {
	Build
	if (-not $NoLogs) { Ensure-Dir $logDir }

	$args = @(
		"-days", "$Days",
		"-config-dir", $configDir
	)

	if ($NoLogs) {
		$args += "-no-logs"
	}
 else {
		$args += @("-log-dir", $logDir)
	}

	Write-Host "Running: $exe $($args -join ' ')"
	& $exe @args
	if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

function Clean {
	if (Test-Path $binDir) {
		Write-Host "Removing $binDir"
		Remove-Item -Recurse -Force $binDir
	}
}

function Smoke {
	Require-File (Join-Path $configDir "folders.txt")
	Require-File (Join-Path $configDir "backup.txt")
	Require-File (Join-Path $configDir "logging.json")

	Write-Host "Smoke test: build + run -no-logs -days 0"
	$script:Days = 0
	Run -NoLogs
}

switch ($Task) {
	"build" { Build }
	"run" { Run }
	"run-nologs" { Run -NoLogs }
	"clean" { Clean }
	"smoke" { Smoke }
	"test" { Test-Unit }
	"test-integration" { Test-Integration }
	"test-all" { Test-All }
	"coverage" { Coverage }
}
