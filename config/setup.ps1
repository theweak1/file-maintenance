# ============================================================================
# File Maintenance CLI - Setup Wizard
# ============================================================================
# This script launches a GUI to help users configure the file maintenance tool.
# Run this from PowerShell: .\setup.ps1
# ============================================================================

param(
    [string]$ConfigDir = "$PSScriptRoot\config"
)

$ErrorActionPreference = "Stop"

# Ensure config directory exists
if (-not (Test-Path $ConfigDir)) {
    New-Item -ItemType Directory -Path $ConfigDir -Force | Out-Null
}

# ============================================================================
# Load Windows Forms Assembly
# ============================================================================
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

# ============================================================================
# Form Configuration
# ============================================================================
$form = New-Object System.Windows.Forms.Form
$form.Text = "File Maintenance Tool - Setup Wizard"
$form.Size = New-Object System.Drawing.Size(750, 680)
$form.StartPosition = "CenterScreen"
$form.FormBorderStyle = "FixedDialog"
$form.MaximizeBox = $false
$form.MinimizeBox = $false
$form.BackColor = [System.Drawing.Color]::FromArgb(240, 240, 240)

# ============================================================================
# Title Label
# ============================================================================
$titleLabel = New-Object System.Windows.Forms.Label
$titleLabel.Location = New-Object System.Drawing.Point(20, 15)
$titleLabel.Size = New-Object System.Drawing.Size(700, 35)
$titleLabel.Text = "Welcome to File Maintenance Tool Setup"
$titleLabel.Font = New-Object System.Drawing.Font("Segoe UI", 16, [System.Drawing.FontStyle]::Bold)
$titleLabel.ForeColor = [System.Drawing.Color]::FromArgb(0, 120, 215)
$form.Controls.Add($titleLabel)

# ============================================================================
# Backup Location Section
# ============================================================================
$backupLabel = New-Object System.Windows.Forms.Label
$backupLabel.Location = New-Object System.Drawing.Point(20, 60)
$backupLabel.Size = New-Object System.Drawing.Size(200, 20)
$backupLabel.Text = "Backup Location:"
$backupLabel.Font = New-Object System.Drawing.Font("Segoe UI", 10, [System.Drawing.FontStyle]::Bold)
$form.Controls.Add($backupLabel)

$backupTextBox = New-Object System.Windows.Forms.TextBox
$backupTextBox.Location = New-Object System.Drawing.Point(20, 83)
$backupTextBox.Size = New-Object System.Drawing.Size(520, 25)
$backupTextBox.Text = "D:\backups"
$backupTextBox.Font = New-Object System.Drawing.Font("Segoe UI", 10)
$form.Controls.Add($backupTextBox)

$backupBrowseBtn = New-Object System.Windows.Forms.Button
$backupBrowseBtn.Location = New-Object System.Drawing.Point(550, 82)
$backupBrowseBtn.Size = New-Object System.Drawing.Size(80, 28)
$backupBrowseBtn.Text = "Browse..."
$backupBrowseBtn.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$form.Controls.Add($backupBrowseBtn)

$backupFolderBrowser = New-Object System.Windows.Forms.FolderBrowserDialog
$backupFolderBrowser.Description = "Select Backup Location"
$backupFolderBrowser.ShowNewFolderButton = $true

$backupBrowseBtn.Add_Click({
        if ($backupFolderBrowser.ShowDialog() -eq "OK") {
            $backupTextBox.Text = $backupFolderBrowser.SelectedPath
        }
    })

# ============================================================================
# Paths Section with Per-Path Backup Settings
# ============================================================================
$pathsLabel = New-Object System.Windows.Forms.Label
$pathsLabel.Location = New-Object System.Drawing.Point(20, 125)
$pathsLabel.Size = New-Object System.Drawing.Size(400, 20)
$pathsLabel.Text = "Paths to Clean:"
$pathsLabel.Font = New-Object System.Drawing.Font("Segoe UI", 10, [System.Drawing.FontStyle]::Bold)
$form.Controls.Add($pathsLabel)

$pathHelpLabel = New-Object System.Windows.Forms.Label
$pathHelpLabel.Location = New-Object System.Drawing.Point(420, 125)
$pathHelpLabel.Size = New-Object System.Drawing.Size(300, 20)
$pathHelpLabel.Text = "Add paths and enable/disable backup per path"
$pathHelpLabel.Font = New-Object System.Drawing.Font("Segoe UI", 8)
$pathHelpLabel.ForeColor = [System.Drawing.Color]::Gray
$form.Controls.Add($pathHelpLabel)

# Path input row
$pathInputLabel = New-Object System.Windows.Forms.Label
$pathInputLabel.Location = New-Object System.Drawing.Point(20, 150)
$pathInputLabel.Size = New-Object System.Drawing.Size(60, 20)
$pathInputLabel.Text = "Path:"
$pathInputLabel.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$form.Controls.Add($pathInputLabel)

$pathInputTextBox = New-Object System.Windows.Forms.TextBox
$pathInputTextBox.Location = New-Object System.Drawing.Point(80, 148)
$pathInputTextBox.Size = New-Object System.Drawing.Size(420, 25)
$pathInputTextBox.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$form.Controls.Add($pathInputTextBox)

$pathBrowseBtn = New-Object System.Windows.Forms.Button
$pathBrowseBtn.Location = New-Object System.Drawing.Point(510, 147)
$pathBrowseBtn.Size = New-Object System.Drawing.Size(60, 26)
$pathBrowseBtn.Text = "..."
$pathBrowseBtn.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$form.Controls.Add($pathBrowseBtn)

$pathFolderBrowser = New-Object System.Windows.Forms.FolderBrowserDialog
$pathFolderBrowser.Description = "Select Path to Clean"

$pathBrowseBtn.Add_Click({
        if ($pathFolderBrowser.ShowDialog() -eq "OK") {
            $pathInputTextBox.Text = $pathFolderBrowser.SelectedPath
        }
    })

# Checkbox for backup enabled
$backupCheckBox = New-Object System.Windows.Forms.CheckBox
$backupCheckBox.Location = New-Object System.Drawing.Point(580, 150)
$backupCheckBox.Size = New-Object System.Drawing.Size(100, 20)
$backupCheckBox.Text = "Backup"
$backupCheckBox.Checked = $true
$backupCheckBox.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$form.Controls.Add($backupCheckBox)

$addPathBtn = New-Object System.Windows.Forms.Button
$addPathBtn.Location = New-Object System.Drawing.Point(680, 147)
$addPathBtn.Size = New-Object System.Drawing.Size(40, 26)
$addPathBtn.Text = "Add"
$addPathBtn.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$form.Controls.Add($addPathBtn)

# Paths list view with backup column
$pathsListView = New-Object System.Windows.Forms.ListView
$pathsListView.Location = New-Object System.Drawing.Point(20, 180)
$pathsListView.Size = New-Object System.Drawing.Size(700, 150)
$pathsListView.FullRowSelect = $true
$pathsListView.GridLines = $true
$pathsListView.View = [System.Windows.Forms.View]::Details
$pathsListView.CheckBoxes = $false
$form.Controls.Add($pathsListView)

# Add columns
$colPath = New-Object System.Windows.Forms.ColumnHeader
$colPath.Text = "Path"
$colPath.Width = 500
$pathsListView.Columns.Add($colPath)

$colBackup = New-Object System.Windows.Forms.ColumnHeader
$colBackup.Text = "Backup"
$colBackup.Width = 80
$pathsListView.Columns.Add($colBackup)

$colAction = New-Object System.Windows.Forms.ColumnHeader
$colAction.Text = ""
$colAction.Width = 100
$pathsListView.Columns.Add($colAction)

# Remove button
$removePathBtn = New-Object System.Windows.Forms.Button
$removePathBtn.Location = New-Object System.Drawing.Point(20, 335)
$removePathBtn.Size = New-Object System.Drawing.Size(100, 35)
$removePathBtn.Text = "Remove Selected"
$removePathBtn.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$form.Controls.Add($removePathBtn)

# Add default paths
$script:Paths = @()

$addPathBtn.Add_Click({
        $path = $pathInputTextBox.Text.Trim()
        if ([string]::IsNullOrEmpty($path)) {
            [System.Windows.Forms.MessageBox]::Show("Please enter a path.", "Validation Error", [System.Windows.Forms.MessageBoxButtons]::OK, [System.Windows.Forms.MessageBoxIcon]::Warning)
            return
        }
    
        if (-not (Test-Path $path)) {
            [System.Windows.Forms.MessageBox]::Show("The specified path does not exist.", "Validation Error", [System.Windows.Forms.MessageBoxButtons]::OK, [System.Windows.Forms.MessageBoxIcon]::Warning)
            return
        }
    
        $backupEnabled = if ($backupCheckBox.Checked) { "Yes" } else { "No" }
        $script:Paths += @{Path = $path; Backup = $backupCheckBox.Checked }
    
        $item = New-Object System.Windows.Forms.ListViewItem($path)
        $item.SubItems.Add($backupEnabled)
        $item.SubItems.Add("Remove")
        $item.Tag = $path
        $pathsListView.Items.Add($item)
    
        $pathInputTextBox.Text = ""
    })

$removePathBtn.Add_Click({
        if ($pathsListView.SelectedItems.Count -gt 0) {
            $selectedPath = $pathsListView.SelectedItems[0].Tag
            $script:Paths = $script:Paths | Where-Object { $_.Path -ne $selectedPath }
            $pathsListView.Items.Remove($pathsListView.SelectedItems[0])
        }
    })

# ============================================================================
# Basic Settings Section
# ============================================================================
$settingsLabel = New-Object System.Windows.Forms.Label
$settingsLabel.Location = New-Object System.Drawing.Point(20, 375)
$settingsLabel.Size = New-Object System.Drawing.Size(200, 20)
$settingsLabel.Text = "Settings:"
$settingsLabel.Font = New-Object System.Drawing.Font("Segoe UI", 10, [System.Drawing.FontStyle]::Bold)
$form.Controls.Add($settingsLabel)

# Days to retain
$daysLabel = New-Object System.Windows.Forms.Label
$daysLabel.Location = New-Object System.Drawing.Point(20, 400)
$daysLabel.Size = New-Object System.Drawing.Size(150, 20)
$daysLabel.Text = "Files older than (days):"
$daysLabel.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$form.Controls.Add($daysLabel)

$daysNumeric = New-Object System.Windows.Forms.NumericUpDown
$daysNumeric.Location = New-Object System.Drawing.Point(180, 398)
$daysNumeric.Size = New-Object System.Drawing.Size(80, 25)
$daysNumeric.Minimum = 0
$daysNumeric.Maximum = 365
$daysNumeric.Value = 7
$daysNumeric.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$form.Controls.Add($daysNumeric)

# Log retention
$logRetentionLabel = New-Object System.Windows.Forms.Label
$logRetentionLabel.Location = New-Object System.Drawing.Point(300, 400)
$logRetentionLabel.Size = New-Object System.Drawing.Size(140, 20)
$logRetentionLabel.Text = "Log retention (days):"
$logRetentionLabel.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$form.Controls.Add($logRetentionLabel)

$logRetentionNumeric = New-Object System.Windows.Forms.NumericUpDown
$logRetentionNumeric.Location = New-Object System.Drawing.Point(450, 398)
$logRetentionNumeric.Size = New-Object System.Drawing.Size(80, 25)
$logRetentionNumeric.Minimum = 1
$logRetentionNumeric.Maximum = 365
$logRetentionNumeric.Value = 30
$logRetentionNumeric.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$form.Controls.Add($logRetentionNumeric)

# ============================================================================
# Advanced Options (Collapsible)
# ============================================================================
$advancedCheck = New-Object System.Windows.Forms.CheckBox
$advancedCheck.Location = New-Object System.Drawing.Point(20, 435)
$advancedCheck.Size = New-Object System.Drawing.Size(200, 20)
$advancedCheck.Text = "Show Advanced Options"
$advancedCheck.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$form.Controls.Add($advancedCheck)

# Advanced options panel (hidden by default)
$advancedPanel = New-Object System.Windows.Forms.Panel
$advancedPanel.Location = New-Object System.Drawing.Point(20, 460)
$advancedPanel.Size = New-Object System.Drawing.Size(700, 130)
$advancedPanel.Visible = $false
$form.Controls.Add($advancedPanel)

# Row 1
$walkersLabel = New-Object System.Windows.Forms.Label
$walkersLabel.Location = New-Object System.Drawing.Point(0, 5)
$walkersLabel.Size = New-Object System.Drawing.Size(130, 20)
$walkersLabel.Text = "Concurrent walkers:"
$walkersLabel.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$advancedPanel.Controls.Add($walkersLabel)

$walkersNumeric = New-Object System.Windows.Forms.NumericUpDown
$walkersNumeric.Location = New-Object System.Drawing.Point(140, 3)
$walkersNumeric.Size = New-Object System.Drawing.Size(60, 25)
$walkersNumeric.Minimum = 1
$walkersNumeric.Maximum = 10
$walkersNumeric.Value = 1
$walkersNumeric.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$advancedPanel.Controls.Add($walkersNumeric)

$queueLabel = New-Object System.Windows.Forms.Label
$queueLabel.Location = New-Object System.Drawing.Point(240, 5)
$queueLabel.Size = New-Object System.Drawing.Size(80, 20)
$queueLabel.Text = "Queue size:"
$queueLabel.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$advancedPanel.Controls.Add($queueLabel)

$queueNumeric = New-Object System.Windows.Forms.NumericUpDown
$queueNumeric.Location = New-Object System.Drawing.Point(330, 3)
$queueNumeric.Size = New-Object System.Drawing.Size(70, 25)
$queueNumeric.Minimum = 10
$queueNumeric.Maximum = 1000
$queueNumeric.Value = 300
$queueNumeric.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$advancedPanel.Controls.Add($queueNumeric)

$retriesLabel = New-Object System.Windows.Forms.Label
$retriesLabel.Location = New-Object System.Drawing.Point(440, 5)
$retriesLabel.Size = New-Object System.Drawing.Size(60, 20)
$retriesLabel.Text = "Retries:"
$retriesLabel.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$advancedPanel.Controls.Add($retriesLabel)

$retriesNumeric = New-Object System.Windows.Forms.NumericUpDown
$retriesNumeric.Location = New-Object System.Drawing.Point(510, 3)
$retriesNumeric.Size = New-Object System.Drawing.Size(60, 25)
$retriesNumeric.Minimum = 0
$retriesNumeric.Maximum = 10
$retriesNumeric.Value = 2
$retriesNumeric.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$advancedPanel.Controls.Add($retriesNumeric)

$cooldownLabel = New-Object System.Windows.Forms.Label
$cooldownLabel.Location = New-Object System.Drawing.Point(600, 5)
$cooldownLabel.Size = New-Object System.Drawing.Size(80, 20)
$cooldownLabel.Text = "Cooldown (ms):"
$cooldownLabel.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$advancedPanel.Controls.Add($cooldownLabel)

$cooldownNumeric = New-Object System.Windows.Forms.NumericUpDown
$cooldownNumeric.Location = New-Object System.Drawing.Point(630, 35)
$cooldownNumeric.Size = New-Object System.Drawing.Size(60, 25)
$cooldownNumeric.Minimum = 0
$cooldownNumeric.Maximum = 5000
$cooldownNumeric.Value = 0
$cooldownNumeric.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$advancedPanel.Controls.Add($cooldownNumeric)

# Row 2 - More advanced options
$maxFilesLabel = New-Object System.Windows.Forms.Label
$maxFilesLabel.Location = New-Object System.Drawing.Point(0, 40)
$maxFilesLabel.Size = New-Object System.Drawing.Size(130, 20)
$maxFilesLabel.Text = "Max files (0=unlimited):"
$maxFilesLabel.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$advancedPanel.Controls.Add($maxFilesLabel)

$maxFilesNumeric = New-Object System.Windows.Forms.NumericUpDown
$maxFilesNumeric.Location = New-Object System.Drawing.Point(140, 38)
$maxFilesNumeric.Size = New-Object System.Drawing.Size(80, 25)
$maxFilesNumeric.Minimum = 0
$maxFilesNumeric.Maximum = 100000
$maxFilesNumeric.Value = 0
$maxFilesNumeric.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$advancedPanel.Controls.Add($maxFilesNumeric)

$maxRuntimeLabel = New-Object System.Windows.Forms.Label
$maxRuntimeLabel.Location = New-Object System.Drawing.Point(260, 40)
$maxRuntimeLabel.Size = New-Object System.Drawing.Size(130, 20)
$maxRuntimeLabel.Text = "Max runtime (minutes):"
$maxRuntimeLabel.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$advancedPanel.Controls.Add($maxRuntimeLabel)

$maxRuntimeNumeric = New-Object System.Windows.Forms.NumericUpDown
$maxRuntimeNumeric.Location = New-Object System.Drawing.Point(400, 38)
$maxRuntimeNumeric.Size = New-Object System.Drawing.Size(70, 25)
$maxRuntimeNumeric.Minimum = 0
$maxRuntimeNumeric.Maximum = 1440
$maxRuntimeNumeric.Value = 30
$maxRuntimeNumeric.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$advancedPanel.Controls.Add($maxRuntimeNumeric)

# Walkers info
$walkersInfoLabel = New-Object System.Windows.Forms.Label
$walkersInfoLabel.Location = New-Object System.Drawing.Point(0, 100)
$walkersInfoLabel.Size = New-Object System.Drawing.Size(350, 30)
$walkersInfoLabel.Text = "Tip: More walkers = faster but more system load. Use 1 for network shares."
$walkersInfoLabel.Font = New-Object System.Drawing.Font("Segoe UI", 8)
$walkersInfoLabel.ForeColor = [System.Drawing.Color]::Gray
$advancedPanel.Controls.Add($walkersInfoLabel)

$queueInfoLabel = New-Object System.Windows.Forms.Label
$queueInfoLabel.Location = New-Object System.Drawing.Point(370, 100)
$queueInfoLabel.Size = New-Object System.Drawing.Size(320, 30)
$queueInfoLabel.Text = "Tip: Larger queue = more memory but better throughput for large folders."
$queueInfoLabel.Font = New-Object System.Drawing.Font("Segoe UI", 8)
$queueInfoLabel.ForeColor = [System.Drawing.Color]::Gray
$advancedPanel.Controls.Add($queueInfoLabel)

$advancedCheck.Add_CheckedChanged({
        $advancedPanel.Visible = $advancedCheck.Checked
        # Keep form at max 680 height to always show buttons
        $form.Size = New-Object System.Drawing.Size(750, 680)
    })

# ============================================================================
# Buttons
# ============================================================================
$cancelBtn = New-Object System.Windows.Forms.Button
$cancelBtn.Location = New-Object System.Drawing.Point(380, 600)
$cancelBtn.Size = New-Object System.Drawing.Size(120, 40)
$cancelBtn.Text = "Cancel"
$cancelBtn.Font = New-Object System.Drawing.Font("Segoe UI", 10)
$cancelBtn.DialogResult = "Cancel"
$form.Controls.Add($cancelBtn)

$saveBtn = New-Object System.Windows.Forms.Button
$saveBtn.Location = New-Object System.Drawing.Point(510, 600)
$saveBtn.Size = New-Object System.Drawing.Size(180, 40)
$saveBtn.Text = "Save & Exit"
$saveBtn.Font = New-Object System.Drawing.Font("Segoe UI", 11, [System.Drawing.FontStyle]::Bold)
$saveBtn.BackColor = [System.Drawing.Color]::FromArgb(0, 120, 215)
$saveBtn.ForeColor = [System.Drawing.Color]::White
$saveBtn.FlatStyle = "Flat"
$form.Controls.Add($saveBtn)

# ============================================================================
# Save Configuration Function
# ============================================================================
function Save-Configuration {
    param(
        [string]$BackupPath,
        [object[]]$Paths,
        [int]$Days,
        [int]$LogRetention,
        [int]$Walkers,
        [int]$QueueSize,
        [int]$Retries,
        [int]$Cooldown,
        [int]$MaxFiles,
        [int]$MaxRuntime,
        [bool]$NoBackup
    )
    
    # Build paths section with per-path backup settings
    $pathsContent = ""
    foreach ($p in $Paths) {
        $backupSetting = if ($p.Backup) { "yes" } else { "no" }
        $pathsContent += "$($p.Path), $backupSetting`n"
    }
    
    $configContent = @"
; File Maintenance Tool Configuration
; Generated by Setup Wizard
; ====================================

[backup]
path=$BackupPath

[paths]
; Paths to clean (one per line)
; Format: path, yes|no (yes = backup enabled, no = backup disabled)
$($pathsContent.TrimEnd())

[settings]
; File retention: only files older than this many days will be processed
days=$Days

; Log retention: how many days to keep log files
log-retention=$LogRetention

[advanced]
; Number of concurrent folder walkers
walkers=$Walkers

; Size of the job queue
queue-size=$QueueSize

; Number of retries on copy failure
retries=$Retries

; Cooldown in milliseconds between file operations
cooldown=$Cooldown

; Maximum files to process (0 = unlimited)
max-files=$MaxFiles

; Maximum runtime in minutes (0 = unlimited)
max-runtime=$MaxRuntime
"@
    
    if ($NoBackup) {
        $configContent += @"

; Disable all backups - delete files only
no-backup=true
"@
    }
    
    $configFile = Join-Path $ConfigDir "config.ini"
    $configContent | Out-File -FilePath $configFile -Encoding UTF8 -Force
    
    return $configFile
}

# ============================================================================
# Save Button Click Handler
# ============================================================================
$saveBtn.Add_Click({
        # Validate backup path
        if ([string]::IsNullOrWhiteSpace($backupTextBox.Text)) {
            [System.Windows.Forms.MessageBox]::Show("Please specify a backup location.", "Validation Error", [System.Windows.Forms.MessageBoxButtons]::OK, [System.Windows.Forms.MessageBoxIcon]::Warning)
            $form.ActiveControl = $backupTextBox
            return
        }
    
        # Validate paths
        if ($script:Paths.Count -eq 0) {
            [System.Windows.Forms.MessageBox]::Show("Please add at least one path to clean.", "Validation Error", [System.Windows.Forms.MessageBoxButtons]::OK, [System.Windows.Forms.MessageBoxIcon]::Warning)
            return
        }
    
        # Save configuration
        try {
            $walkers = if ($advancedCheck.Checked) { $walkersNumeric.Value } else { 1 }
            $queueSize = if ($advancedCheck.Checked) { $queueNumeric.Value } else { 300 }
            $retries = if ($advancedCheck.Checked) { $retriesNumeric.Value } else { 2 }
            $cooldown = if ($advancedCheck.Checked) { $cooldownNumeric.Value } else { 0 }
            $maxFiles = if ($advancedCheck.Checked) { $maxFilesNumeric.Value } else { 0 }
            $maxRuntime = if ($advancedCheck.Checked) { $maxRuntimeNumeric.Value } else { 30 }
            $noBackup = if ($advancedCheck.Checked) { $noBackupCheck.Checked } else { $false }
        
            $configFile = Save-Configuration -BackupPath $backupTextBox.Text -Paths $script:Paths -Days $daysNumeric.Value -LogRetention $logRetentionNumeric.Value -Walkers $walkers -QueueSize $queueSize -Retries $retries -Cooldown $cooldown -MaxFiles $maxFiles -MaxRuntime $maxRuntime -NoBackup $noBackup
        
            [System.Windows.Forms.MessageBox]::Show("Configuration saved successfully to:`n$configFile", "Success", [System.Windows.Forms.MessageBoxButtons]::OK, [System.Windows.Forms.MessageBoxIcon]::Information)
            $form.DialogResult = "OK"
            $form.Close()
        }
        catch {
            [System.Windows.Forms.MessageBox]::Show("Failed to save configuration: $_", "Error", [System.Windows.Forms.MessageBoxButtons]::OK, [System.Windows.Forms.MessageBoxIcon]::Error)
        }
    })

# ============================================================================
# Cancel Button Handler
# ============================================================================
$cancelBtn.Add_Click({
        $result = [System.Windows.Forms.MessageBox]::Show("Are you sure you want to cancel setup?", "Confirm Cancel", [System.Windows.Forms.MessageBoxButtons]::YesNo, [System.Windows.Forms.MessageBoxIcon]::Question)
        if ($result -eq "Yes") {
            $form.DialogResult = "Cancel"
            $form.Close()
        }
    })

# ============================================================================
# Show Form
# ============================================================================
$result = $form.ShowDialog()

if ($result -eq "OK") {
    Write-Host "Setup completed successfully!"
    Write-Host "You can now run the file maintenance tool."
}
else {
    Write-Host "Setup cancelled."
    exit 1
}
