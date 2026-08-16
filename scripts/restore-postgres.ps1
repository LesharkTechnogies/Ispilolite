param(
  [Parameter(Mandatory = $true)][string]$DatabaseUrl,
  [Parameter(Mandatory = $true)][string]$BackupFile,
  [switch]$Clean
)
$ErrorActionPreference = "Stop"
$args = @("--dbname=$DatabaseUrl", "--no-owner", "--exit-on-error")
if ($Clean) { $args += @("--clean", "--if-exists") }
$args += $BackupFile
& pg_restore @args
if ($LASTEXITCODE -ne 0) { throw "pg_restore failed" }
