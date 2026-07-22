param([string]$BaseURL = "http://localhost")
$ErrorActionPreference = "Continue"

$adminEmail = "admin@demo.com"
$adminPass = "admin"

Write-Host "=== Logging in as admin ===" -ForegroundColor Cyan
$loginBody = @{identifier=$adminEmail;password=$adminPass} | ConvertTo-Json
$login = Invoke-RestMethod -Uri "$BaseURL/api/v1/auth/login" -Method Post -Body $loginBody -ContentType "application/json"
$token = $login.accessToken
$headers = @{Authorization = "Bearer $token"}

Write-Host "=== Finding test servers ===" -ForegroundColor Cyan
$response = Invoke-RestMethod -Uri "$BaseURL/api/v1/servers?page=1&limit=100" -Headers $headers
$testServers = $response.servers | Where-Object {
    $_.serverName -like "import-srv-*" -or
    $_.serverName -like "bruno-test-*" -or
    $_.serverName -like "unique-*" -or
    $_.serverName -like "conflict-*"
}

if (-not $testServers) {
    Write-Host "No test servers found, DB is clean." -ForegroundColor Green
    exit 0
}

Write-Host "Deleting $($testServers.Count) test servers..." -ForegroundColor Yellow
foreach ($srv in $testServers) {
    $url = "$BaseURL/api/v1/servers/$($srv.serverId)"
    Invoke-RestMethod -Uri $url -Method Delete -Headers $headers | Out-Null
    Write-Host "  Deleted: $($srv.serverName) ($($srv.ipv4))"
}

Write-Host "Done. DB is clean." -ForegroundColor Green
