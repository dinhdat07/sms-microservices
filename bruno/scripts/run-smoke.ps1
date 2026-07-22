param([string]$Env = "local")
$ErrorActionPreference = "Stop"
$collectionDir = Resolve-Path "$PSScriptRoot\.."

Write-Host "=== Bruno API Smoke Tests - SMS ===" -ForegroundColor Cyan
Write-Host "Environment: $Env" -ForegroundColor Cyan

Push-Location $collectionDir
Write-Host "`n--- all folders ---" -ForegroundColor Yellow
bru run auth servers reporting health authorization --env $Env
$exitCode = $LASTEXITCODE
Pop-Location

if ($exitCode -eq 0) {
    Write-Host "`nALL TESTS PASSED" -ForegroundColor Green
    exit 0
} else {
    Write-Host "`nFAILED (exit code: $exitCode)" -ForegroundColor Red
    exit 1
}
