param(
  [string]$DatabaseUrl = $env:DATABASE_URL
)

if ([string]::IsNullOrWhiteSpace($DatabaseUrl)) {
  Write-Error "DATABASE_URL is required (either pass -DatabaseUrl or set env var)."
  exit 1
}

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$migrationsDir = Join-Path $scriptDir "..\\db\\migrations"

docker run --rm `
  -v "${migrationsDir}:/migrations:ro" `
  migrate/migrate `
  -path /migrations `
  -database "$DatabaseUrl" `
  up

