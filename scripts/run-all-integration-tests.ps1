param(
  [string]$GoTestTags = "integration"
)

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

$paths = @(
  "./services/upload/test/integration",
  "./services/metadata/test/integration",
  "./services/orchestrator/test/integration",
  "./services/transcode/test/integration",
  "./services/segment/test/integration",
  "./services/thumbnail/test/integration",
  "./services/publish/test/integration",
  "./services/expirer/cmd/server",
  "./services/outbox-dispatcher/cmd/server"
)

foreach ($p in $paths) {
  $args = @("test", "-tags", $GoTestTags, "-v", "-count=1", $p)
  Write-Host "==> go $($args -join ' ')"
  & go @args
  if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
  }
}

