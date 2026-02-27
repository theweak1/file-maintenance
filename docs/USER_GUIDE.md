# File Maintenance Tool - User Guide

A safe, scheduled file maintenance utility for Windows that automatically cleans up old files while optionally backing them up first.

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [First-Time Setup](#first-time-setup)
3. [Running the Application](#running-the-application)
4. [Configuration](#configuration)
5. [Command-Line Options](#command-line-options)
6. [Scheduling with Windows Task Scheduler](#scheduling-with-windows-task-scheduler)
7. [Viewing Logs](#viewing-logs)
8. [Troubleshooting](#troubleshooting)

---

## Quick Start

1. **Run the application** - Double-click `fileMaintenance.exe`
2. **Complete the setup wizard** - Configure your backup location and paths to clean
3. **The application will** - Scan configured paths, backup old files (if enabled), and delete them

---

## First-Time Setup

When you first run the application, a Setup Wizard will appear:

![Setup Wizard](Images/File%20Maintenance%20Tool%20-%20Setup%20Wizard.png)


### Setup Wizard Fields

| Field | Description |
|-------|-------------|
| **Backup Location** | Where to store backups before deletion (local drive or network share) |
| **Paths to Clean** | List of folders/files to process |
| **Backup** checkbox | Enable/disable backup for each path (checked = backup enabled) |

### How to Use the Setup Wizard

1. **Backup Location**
   - Enter the path where backups should be stored
   - Click **Browse...** to select a folder using the file browser
   - Example: `D:\backups` or `\\server\share\backups`

2. **Add Paths to Clean**
   - Enter a folder or file path in the "Path" field
   - Click **Browse...** to select a folder
   - Click **Add** to add it to the list
   - Use the **Backup** checkbox to enable/disable backup for that path

3. **Save Configuration**
   - Click **Save & Exit** to save your configuration and run the application

![filled setup wizard](Images/Setup%20Wizard.png)

---

## Running the Application

### Basic Usage

```powershell
fileMaintenance.exe -days 7
```

This will delete files older than 7 days (after backing them up if enabled).

### Interactive Run

Simply double-click the executable or run from command prompt:

```powershell
fileMaintenance.exe
```

The application will:
1. Read configuration from `config/config.ini`
2. Validate backup location is accessible
3. Scan configured paths for files older than the specified days
4. Backup files (if enabled for that path)
5. Delete original files
6. Clean up empty directories
7. Generate logs

---

## Configuration

Configuration is stored in `config/config.ini` in the same directory as the executable.

### Example Configuration

```ini
[backup]
path=D:\backups

[paths]
C:\Temp\OldFiles, yes
C:\Temp\ToDelete, no
\\server\share\incoming, yes
C:\Logs\debug.log, no
```

### Configuration Format

| Section | Key | Description |
|---------|-----|-------------|
| `[backup]` | `path` | Backup destination root path |
| `[paths]` | (lines) | Paths to process with per-path backup control |

### Path Entry Format

```
path, yes|no
```

- `path` - The file or folder to process
- `yes` - Enable backup before deletion
- `no` - Delete without backup (use with caution)

You can also use comments:

```ini
; This is a comment
[paths]
; Folder with backup
C:\Temp\OldFiles, yes
; Folder without backup - CAUTION!
C:\Temp\ToDelete, no
```

---

## Command-Line Options

| Option | Default | Description |
|--------|---------|-------------|
| `-days` | 7 | Only files older than this many days are eligible for deletion |
| `-log-retention` | 30 | Log retention in days |
| `-config-dir` | `<exe>/config` | Config directory |
| `-log-dir` | `<exe>/logs` | Log directory |
| `-no-logs` | false | Console-only logging (no file logs) |
| `-walkers` | 1 | Concurrent path walkers |
| `-queue-size` | 300 | Job queue size |
| `-max-files` | 0 | Max files per run (0 = unlimited) |
| `-max-runtime` | 30m | Max runtime |
| `-cooldown` | 0 | Cooldown between files |
| `-retries` | 2 | Copy retries |

### Recommended Settings for Network Shares

```powershell
fileMaintenance.exe -days 7 -walkers 1 -queue-size 300 -max-runtime 30m -cooldown 50ms -retries 2
```

### Console-Only Mode (No File Logs)

```powershell
fileMaintenance.exe -days 0 -no-logs
```

---

## Scheduling with Windows Task Scheduler

### Recommended Schedule

Run twice daily (e.g., 6:30 AM / 6:30 PM)

### Creating a Scheduled Task

1. Open **Task Scheduler** (search in Start menu)
2. Click **Create Basic Task**
3. Follow the wizard:
   - **Name**: File Maintenance
   - **Trigger**: Daily (or your preferred schedule)
   - **Action**: Start a program
   - **Program**: `powershell.exe`
   - **Arguments**: `-NoProfile -ExecutionPolicy Bypass -Command "Start-Process -FilePath 'C:\path\to\fileMaintenance.exe' -ArgumentList '-days 7 -walkers 1 -max-runtime 30m -cooldown 50ms' -Priority BelowNormal -WindowStyle Hidden -Wait"`

### Task Settings

- ✅ Run whether user is logged on or not
- ✅ Run as soon as possible after a missed start
- ✅ Stop task if running longer than 1 hour

---

## Viewing Logs

Logs are stored in the `logs` directory next to the executable.

### Log Files

| File | Description |
|------|-------------|
| `maintenance_YYYY-MM-DD.log` | All log levels |
| `errors_YYYY-MM-DD.log` | Errors only |
| `count_YYYY-MM-DD.log` | Summary counts (files deleted per folder) |

![Logs folder](Images/Logs%20folder.png)

### Sample Log Entry

```
[2026-02-27 10:30:15] [INFO] Starting maintenance run
[2026-02-27 10:30:15] [INFO] Path: C:\Temp\OldFiles (backup: yes)
[2026-02-27 10:30:16] [INFO] Backup location: D:\backups
[2026-02-27 10:30:20] [SUCCESS] Backed up: C:\Temp\OldFiles\old-file.txt -> D:\backups\27Feb26\OldFiles\old-file.txt
[2026-02-27 10:30:21] [SUCCESS] Deleted: C:\Temp\OldFiles\old-file.txt
[2026-02-27 10:30:25] [COUNT] C:\Temp\OldFiles: 15 files deleted
```

---

## Troubleshooting

### Error: Backup path is not accessible

**Symptom**: Popup notification appears saying backup location is inaccessible

**Solution**:
1. Verify the backup path exists and is accessible
2. Check network share permissions (if using UNC path)
3. Ensure the backup drive is connected

### Error: Configuration not found

**Symptom**: Application launches setup wizard every time

**Solution**: Run the setup wizard and save your configuration

### Files Not Being Deleted

**Possible causes**:
1. Files are not older than the `-days` threshold
2. Path is not correctly specified in config.ini
3. Application ran with errors - check logs

### Backup not accessible

[Backup Location Error Pop-up](Images/Backup%20Location%20Error%20pop-up.png)

### How to Modify Configuration

1. Edit `config/config.ini` directly with a text editor, OR
2. Delete `config/config.ini` and run the application to launch the setup wizard again

---

## Safety Features

- ✅ **Backup before delete** - Files are backed up before deletion (when enabled)
- ✅ **No deletion if backup fails** - Original files are kept if backup fails
- ✅ **Path safety** - Prevents directory traversal attacks
- ✅ **Bounded resources** - Limited concurrency and memory usage
- ✅ **Network retry** - Automatic retry with backoff for SMB issues
- ✅ **Empty directory cleanup** - Automatically removes empty folders

---

## Backup Folder Structure

Backups are organized by date for easy restore:

```
backupRoot/
└── 27Feb26/                  (Date of the run)
    └── OldFiles/              (Source folder name)
        └── subfolder/
            └── file.txt      (Preserves original structure)
```

---

## Support

For issues or questions, check:
1. Log files in the `logs` directory
2. Configuration in `config/config.ini`
3. Re-run setup wizard to reconfigure
