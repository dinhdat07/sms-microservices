param (
    [ValidateSet('swarm', 'ci')]
    [string]$Target = 'swarm'
)

$ErrorActionPreference = "Continue"

Push-Location "$PSScriptRoot"

if ($Target -eq 'ci') {
    Write-Host "=== Bringing up Docker Compose stack (CI Mode) ===" -ForegroundColor Cyan
    Push-Location ".."
    docker compose up -d
    Pop-Location
    
    Write-Host "Waiting for services to be ready..." -ForegroundColor Cyan
    Start-Sleep -Seconds 15
    
    $network = "sms_default"
} else {
    Write-Host "=== Using existing Swarm stack ===" -ForegroundColor Cyan
    $network = "sms_stack_default"
}

# ---- Cleanup stale test data via API ----
Write-Host "=== Cleaning up stale test data ===" -ForegroundColor Cyan
& ".\cleanup.ps1" -BaseURL "http://localhost"

# ---- Run tests ----
Write-Host "=== Running Bruno tests ===" -ForegroundColor Cyan
# Run Bruno in Docker container attached to the network so it can reach traefik
docker run --rm -v "${PWD}:/loc" -w /loc --network $network node:20-alpine sh -c "npx -y @usebruno/cli run auth servers reporting health authorization agent --env local --env-var baseURL=http://traefik"
$testExitCode = $LASTEXITCODE

if ($Target -eq 'ci') {
    Write-Host "=== Tearing down Docker Compose stack ===" -ForegroundColor Cyan
    Push-Location ".."
    docker compose down -v
    Pop-Location
}

Write-Host "`n=== Done ===" -ForegroundColor Green
exit $testExitCode
