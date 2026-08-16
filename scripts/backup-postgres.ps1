param(
  [Parameter(Mandatory = $true)][string]$DatabaseUrl,
  [string]$BackupDirectory = "backups",
  [int]$RetentionDays = 30
)
$ErrorActionPreference = "Stop"
New-Item -ItemType Directory -Force -Path $BackupDirectory | Out-Null
$stamp = Get-Date -Format "yyyyMMdd_HHmmss"
$file = Join-Path $BackupDirectory "ispilolite_$stamp.dump"
& pg_dump --dbname=$DatabaseUrl --format=custom --compress=9 --no-owner --file=$file
if ($LASTEXITCODE -ne 0) { throw "pg_dump failed" }
Get-ChildItem -Path $BackupDirectory -Filter "ispilolite_*.dump" |
  Where-Object { $_.LastWriteTimeUtc -lt (Get-Date).ToUniversalTime().AddDays(-$RetentionDays) } |
  Remove-Item -Force
$file
