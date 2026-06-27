# Changelog

## Release - 2026-06-27

### Changed

- startup: make setup/configuration mode the default when the executable is launched without `-run`.
- startup: require explicit `-run` before the background backup/delete maintenance process executes from CLI or Task Scheduler.
- startup: when `-run` is used and `config.ini` is missing, exit with an error instead of opening the setup GUI.
- config: split configuration into a file maintenance plan (`FilePlanConfig`) and runtime execution settings (`RuntimeConfig`).
- config: change runtime precedence to defaults -> `config.ini` -> explicitly passed CLI flags.
- config: preserve explicit zero values such as `days=0`, `max-files=0`, and `max-runtime=0` by using pointer-based runtime overrides.
- config: support Go duration strings in `config.ini`, such as `cooldown=50ms` and `max-runtime=55m`, while retaining numeric millisecond compatibility.
- setup: replace the single Save & Exit action with Cancel, Save & Close, and Save & Run.
- setup: write duration strings for cooldown and max runtime from the Windows setup wizard.
- worker: replace per-file backup destination space checks with batch-level validation.
- worker: collect jobs into an in-memory batch up to `queue-size`, check the total backup-enabled byte requirement once, then process the complete batch before accepting more work.
- worker: use an unbuffered job input channel so walkers are backpressured while the current batch is being validated and processed.
- docs: update README and execution-flow diagrams for setup-first mode, explicit `-run`, config split, CLI precedence, and batch-level backup-space validation.

### Added

- cli: add `-run` as the explicit maintenance execution flag.
- cli: add `-setup` as an explicit setup/configuration mode flag.
- cli: add `-no-backup` to force delete-only behavior for a run.
- tests: add config coverage confirming plan/runtime split, duration-string parsing, legacy millisecond parsing, required paths, and explicit zero-value handling.
- tests: add setup exit-code mapping coverage for Cancel, Save & Close, and Save & Run.
- tests: add integration coverage confirming backup space is checked once per batch instead of once per file.
- tests: add integration coverage confirming an insufficient-space batch cancels before source files are deleted.

### Validation notes

- Focused validation commands:
  - `go test ./internal/config ./internal/platform/windows/setup`
  - `go test ./internal/maintenance -run "BackupSpaceCheckedOncePerBatch|InsufficientBackupSpaceForBatch" -v -count=1`
- Full `go test ./...` still has the existing Linux-only path expectation issue in `TestBuildBackupPath_Table/error_on_different_drives`.
