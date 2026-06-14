# File Maintenance Tool

A safe, scheduled file maintenance utility designed for Windows-first unattended cleanup while keeping the core logic isolated behind OS-specific platform abstractions.

The tool can:

- Scan configured file/folder paths.
- Identify files older than the configured retention period.
- Back up eligible files before deletion when backup is enabled for that path.
- Delete files after a successful backup, or delete directly when backup is intentionally disabled for that path.
- Clean up empty directories without crossing the configured path root.
- Write operational logs with retention cleanup.
- Launch a Windows setup wizard on first run when `config.ini` is missing.

---

## Current Default Layout

The application currently uses portable defaults beside the executable:

```text
fileMaintenance.exe
config/
  config.ini
  logging.json        # optional
logs/
  maintenance_YYYY-MM-DD.log
  errors_YYYY-MM-DD.log
  count_YYYY-MM-DD.log
```

This is intentional. The platform package still supports OS-conventional directories such as AppData on Windows, but `cmd/main/main.go` currently defaults to `<exe>/config` and `<exe>/logs` to make Task Scheduler deployments and portable runs easier to reason about.

Override paths with:

```powershell
fileMaintenance.exe -config-dir "C:\path\to\config" -log-dir "C:\path\to\logs"
```

---

## First-Time Setup

On Windows, startup checks whether this file exists:

```text
<config-dir>/config.ini
```

If it is missing, the Windows platform implementation launches the embedded PowerShell setup wizard. The wizard creates `config.ini` and the application continues only if the file was created successfully.

On Linux and macOS, no GUI setup wizard is currently provided. If `config.ini` is missing, the platform implementation returns `false` and the application exits safely without processing files.

---

## How It Works

1. **Startup**
   - Resolve the active OS platform implementation.
   - Resolve the executable directory.
   - Set portable default paths for `config/` and `logs/`.
   - Parse CLI flags.
   - Initialize logging.

2. **Configuration Check**
   - Call `platform.EnsureConfig(configDir, exeDir)`.
   - On Windows, launch the setup wizard if `config.ini` is missing.
   - On Linux/macOS, exit safely if `config.ini` is missing.

3. **Configuration Loading**
   - Read `config.ini`.
   - Parse `[backup]`, `[paths]`, and optional `[settings]` / `[advanced]` sections.
   - Apply non-zero config values over CLI/default values.

4. **Safety Checks**
   - Determine whether any configured path has backup enabled.
   - If backup is required, validate the backup destination before deleting anything.
   - Show a platform-specific critical notification if the backup path is inaccessible.

5. **Maintenance Worker**
   - Scan configured paths using bounded walkers.
   - Process file operations through one serialized processor.
   - Copy before delete when backup is enabled.
   - Delete directly only when backup is disabled for that path.
   - Track per-path delete counts.

6. **Cleanup and Exit**
   - Remove empty directories safely.
   - Prune old logs according to log retention.
   - Exit with an error code when fatal configuration or worker errors occur.

---

## Execution Flow

The high-level Mermaid source is maintained in:

```text
docs/diagrams/execution-flow-high-level.md
```

The detailed execution flow is maintained in:

```text
docs/diagrams/execution-flow.md
```

The execution flow uses concurrent path discovery but serialized file operations. This avoids uncontrolled SMB/network load and keeps backup/delete behavior predictable.

---

## Command-Line Flags

### Retention and Logging

| Flag             | Default | Description                                        |
| ---------------- | ------: | -------------------------------------------------- |
| `-days`          |     `7` | Only files older than this many days are eligible. |
| `-log-retention` |    `30` | Number of days to keep log files.                  |
| `-no-logs`       | `false` | Disable file logging and write to stdout/stderr.   |

### Paths

| Flag          | Default        | Description                                                    |
| ------------- | -------------- | -------------------------------------------------------------- |
| `-config-dir` | `<exe>/config` | Directory containing `config.ini` and optional `logging.json`. |
| `-log-dir`    | `<exe>/logs`   | Directory where log files are written.                         |

### Resource Controls

| Flag           | Default | Description                                                    |
| -------------- | ------: | -------------------------------------------------------------- |
| `-walkers`     |     `1` | Number of concurrent path walkers.                             |
| `-queue-size`  |   `300` | Buffered job queue size.                                       |
| `-max-files`   |     `0` | Maximum jobs to handle in one run. `0` means unlimited.        |
| `-max-runtime` |   `30m` | Maximum run duration. `0` means unlimited.                     |
| `-cooldown`    |     `0` | Delay after each processed job. Useful for SMB/network pacing. |
| `-retries`     |     `2` | Number of backup copy retries.                                 |

---

## Configuration

### `config/config.ini`

Required sections:

```ini
[backup]
path=D:\backups

[paths]
C:\Temp\OldFiles, yes
C:\Temp\ToDelete, no
```

Optional sections:

```ini
[settings]
days=7
log-retention=30

[advanced]
walkers=1
queue-size=300
retries=2
cooldown=50
max-files=0
max-runtime=30
```

### `[backup]`

| Key    | Description                                                |
| ------ | ---------------------------------------------------------- |
| `path` | Backup destination root path. Can be local or network/SMB. |

### `[paths]`

Each standalone line is a file or folder path followed by optional backup behavior:

```ini
path, yes|no
```

| Value   | Meaning                                              |
| ------- | ---------------------------------------------------- |
| `yes`   | Back up eligible files before deletion.              |
| `no`    | Delete eligible files without backup. Use carefully. |
| omitted | Backup defaults to enabled.                          |

Supported path types:

| Type   | Example                 | Behavior                                       |
| ------ | ----------------------- | ---------------------------------------------- |
| Folder | `C:\Temp\OldFiles, yes` | Recursively evaluates files inside the folder. |
| File   | `C:\Logs\old.log, no`   | Evaluates that file directly.                  |

Comments and blank lines are ignored. Lines beginning with `;` or `#` are treated as comments.

### `config/logging.json`

Optional logging configuration:

```json
{
  "DEBUG": false,
  "COUNT": true,
  "INFO": true,
  "WARN": true,
  "ERROR": true,
  "SUCCESS": true,
  "FATAL": true
}
```

Unknown log levels default to enabled.

---

## Backup Layout

Backups are written under a run-date folder:

```text
<backupRoot>/<DDMmmYY>/<folder-name>/<relative-path>/<filename>
```

Example:

```text
Source:
C:\Data\Images\2024\Camera\IMG001.jpg

Backup:
D:\backups\30Jan26\Images\2024\Camera\IMG001.jpg
```

The backup path builder validates that files stay within the configured source root and prevents directory traversal into unintended destinations.

---

## Logging

Default file logs:

```text
logs/maintenance_YYYY-MM-DD.log   # all enabled levels
logs/errors_YYYY-MM-DD.log        # ERROR/FATAL-focused output
logs/count_YYYY-MM-DD.log         # summary counts
```

Per-path delete counts are logged after processing completes so totals reflect actual successful deletions.

---

## OS-Specific Platform Abstraction

The platform layer lives under `internal/platform`:

```text
internal/platform/
  platform.go
  current_windows.go
  current_linux.go
  current_darwin.go
  windows/
  linux/
  macos/
```

The core application depends on the `Platform` interface instead of importing OS-specific packages directly.

Current responsibilities:

- `ShowCritical(title, message string)`
- `DefaultConfigDir(appName string) (string, error)`
- `DefaultLogDir(appName string) (string, error)`
- `EnsureConfig(configDir string, exeDir string) (bool, error)`

Windows provides the setup wizard implementation. Linux and macOS currently perform a safe config existence check only.

---

## Windows Task Scheduler Example

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command ^
Start-Process -FilePath "C:\path\fileMaintenance.exe" `
-ArgumentList "-days 7 -walkers 1 -max-runtime 30m -cooldown 50ms" `
-Priority BelowNormal -WindowStyle Hidden -Wait
```

Recommended task options:

- Run whether user is logged on or not.
- Run as soon as possible after a missed start.
- Stop task if running longer than expected.

---

## Safety Guarantees

- No deletion occurs if backup is enabled and the backup root is inaccessible.
- No deletion occurs if backup copy fails.
- File operations are serialized to reduce network and disk contention.
- Directory cleanup does not cross the configured path root.
- Resource controls prevent unbounded walking or job queue growth.
- Critical backup-location failures trigger platform-specific user notification.

---

## Development and Testing

Run tests:

```powershell
go test ./...
```

Smoke test:

```powershell
.\build.ps1 smoke
```

Build:

```powershell
.\build.ps1 build
```

Note: tests involving Windows drive-letter behavior should be run on Windows or guarded with Windows-specific build tags.

---

## License

Internal / private use.
