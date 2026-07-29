param (
    [string]$Username = "whaleplay710",
    [string]$Tag = "v1.0.0"
)

$ErrorActionPreference = "Stop"

$services = @(
    @{ Name = "sms-identity"; Path = "sms-identity" },
    @{ Name = "sms-management"; Path = "sms-management" },
    @{ Name = "sms-simulator"; Path = "sms-simulator" },
    @{ Name = "sms-monitoring-worker"; Path = "sms-monitoring" },
    @{ Name = "sms-agent-handler"; Path = "sms-agent-handler" },
    @{ Name = "sms-reporting"; Path = "sms-reporting" },
    @{ Name = "sms-notification"; Path = "sms-notification" },
    @{ Name = "sms-frontend"; Path = "sms-frontend" }
)

Write-Host "=== Building and Pushing Docker Images ===" -ForegroundColor Cyan
Write-Host "Docker Hub Username: $Username" -ForegroundColor Cyan
Write-Host "Image Tag: $Tag" -ForegroundColor Cyan
Write-Host "------------------------------------------"

foreach ($service in $services) {
    $imageName = "$Username/$($service.Name):$Tag"
    Write-Host "Building $imageName from $($service.Path)..." -ForegroundColor Yellow
    
    # Build the image
    docker build -t $imageName "./$($service.Path)"
    
    Write-Host "Pushing $imageName to Docker Hub..." -ForegroundColor Yellow
    
    # Push the image
    docker push $imageName
    
    Write-Host "Successfully pushed $imageName" -ForegroundColor Green
    Write-Host "------------------------------------------"
}

Write-Host "All images have been built and pushed successfully!" -ForegroundColor Green
