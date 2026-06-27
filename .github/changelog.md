# Changelog

## Release - 2026-06-27

### Changed

- worker: replace per-file backup destination space checks with batch-level validation.
- worker: collect jobs into an in-memory batch up to `queue-size`, check the total backup-enabled byte requirement once, then process the complete batch before accepting more work.
- worker: use an unbuffered job input channel so walkers are backpressured while the current batch is being validated and processed.
- docs: update README backup-space behavior, safety guarantees, worker flow, and queue-size description.
- docs: update execution-flow diagrams to show batch collection, one destination-space check per batch, serialized processing, and insufficient-space cancellation.

### Added

- tests: add integration coverage confirming backup space is checked once per batch instead of once per file.
- tests: add integration coverage confirming an insufficient-space batch cancels before source files are deleted.

### Validation notes

- The focused worker tests can be run with `go test ./internal/maintenance -run "BackupSpaceCheckedOncePerBatch|InsufficientBackupSpaceForBatch" -v -count=1`.
