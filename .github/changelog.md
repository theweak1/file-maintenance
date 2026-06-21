# Changelog

## Release - 2026-06-21

### Added

- docs: add the current Windows Task Scheduler action fields used for the scheduled maintenance run.
- docs: add a GitHub Actions release-build section covering test/build matrix behavior, release assets, and injected version metadata.
- docs: add `-version` usage and clarify build metadata output.

### Changed

- docs: clarify that portable defaults use `<exe>/config` and `<exe>/logs` for scheduled/portable deployments.
- docs: clarify that non-zero `config.ini` values override CLI/default values after startup flag parsing.
- docs: clarify that CLI duration flags use Go duration strings such as `55m` and `50ms`, while current `[advanced]` config durations are parsed as numeric milliseconds.
- docs: update the platform abstraction section to include backup destination free-space checks through `AvailableBytes`.
- docs: remove unsupported README claims about empty directory cleanup and narrow the backup-layout description to the active worker behavior.
- docs: clarify that Windows currently implements backup space validation, while Linux/macOS still return a not-implemented error for backup-enabled runs.

### Existing release-workflow updates already in this release

- Fix typo in build workflow name (9b41db1).
- ci: checkout repository before creating release (c0d66d6).
- ci: collect release assets safely (e4089c0).
- ci: use platform-specific archive commands (ddadd8e).
- ci: add version metadata for release builds (140cba1).
- feat: add backup destination disk space validation (72b2f23).

### Review notes

- `build.ps1 smoke` still checks for legacy `config/folders.txt` and `config/backup.txt`, while the runtime now uses `config/config.ini`.
- Empty directory cleanup is mentioned in worker comments, but no active cleanup call was found in the current worker path.
- The stricter `backupDestPath` helper exists, but the active worker currently uses `buildBackupPath`.
- The Windows setup wizard labels max runtime as minutes, but the current config parser reads `[advanced] max-runtime` as milliseconds.

If this file is present at release time and was updated in this release commit, its contents will be used as the release notes. Otherwise the workflow will fall back to the default automated note.
