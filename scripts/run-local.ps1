<#
.SYNOPSIS
  Bring up local Postgres + Redis (via Docker) and run the Ispilo Lite API.

.DESCRIPTION
  The API (cmd/auth) needs Postgres and Redis to start. This script starts two
  throwaway Docker containers matching the credentials in .env, waits for them
  to accept connections, then launches the API. Migrations run automatically on
  startup (see pkg/database.runMigrations).

  Requires: Docker Desktop and the Go toolchain on PATH.

.EXAMPLE
  ./scripts/run-local.ps1
#>
[CmdletBinding()]
param(
  [string]$PgContainer    = "ispilo-pg",
  [string]$RedisContainer = "ispilo-redis",
  [string]$PgUser         = "ispilo",
  [string]$PgPassword     = "ispilo-local",
  [string]$PgDb           = "ispilolite"
)

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

function Start-IfMissing($name, [scriptblock]$run) {
  $existing = docker ps -a --filter "name=^/$name$" --format "{{.Names}}"
  if ($existing -eq $name) {
    Write-Host "-> starting existing container $name"
    docker start $name | Out-Null
  } else {
    Write-Host "-> creating container $name"
    & $run
  }
}

Write-Host "== Ispilo Lite local runner =="

Start-IfMissing $PgContainer {
  docker run -d --name $PgContainer `
    -e POSTGRES_USER=$PgUser -e POSTGRES_PASSWORD=$PgPassword -e POSTGRES_DB=$PgDb `
    -p 5432:5432 postgres:16-alpine | Out-Null
}

Start-IfMissing $RedisContainer {
  docker run -d --name $RedisContainer -p 6379:6379 redis:7-alpine | Out-Null
}

Write-Host "-> waiting for Postgres to accept connections..."
for ($i = 0; $i -lt 30; $i++) {
  docker exec $PgContainer pg_isready -U $PgUser -d $PgDb 2>$null | Out-Null
  if ($LASTEXITCODE -eq 0) { break }
  Start-Sleep -Seconds 1
}

$apiPort = if ($env:PORT) { $env:PORT } else { "8001" }
Write-Host "-> starting API on http://localhost:$apiPort (Ctrl+C to stop)"
# .env is loaded by the app itself (pkg/database.loadDotEnv).
go run ./cmd/auth
