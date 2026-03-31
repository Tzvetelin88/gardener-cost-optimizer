param(
    [string]$ApiAddress = ":8080",
    [ValidateSet("mock", "real", "auto")]
    [string]$DataSource = "mock",
    [int]$FrontendPort = 4173,
    [string]$ViteApiBaseUrl = "http://localhost:8080/api/v1"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot

Write-Host "Installing frontend dependencies..."
Push-Location "$root\frontend"
npm install
Pop-Location

$goCommand = Get-Command go -ErrorAction SilentlyContinue
if (-not $goCommand) {
    Write-Warning "Go is not installed or not on PATH. Backend was not launched."
    Write-Host "Install Go and run: cd $root\backend; go run ./cmd/api"
    exit 0
}

Write-Host "Starting Smart Cost Optimizer backend..."
$backendJob = Start-Process -FilePath "go" -ArgumentList "run", "./cmd/api" -WorkingDirectory "$root\backend" -PassThru -NoNewWindow -Environment @{
    "API_ADDR" = $ApiAddress
    "DATA_SOURCE" = $DataSource
}

Write-Host "Building frontend bundle..."
Push-Location "$root\frontend"
$env:VITE_API_BASE_URL = $ViteApiBaseUrl
npm run build
Remove-Item Env:VITE_API_BASE_URL -ErrorAction SilentlyContinue
Pop-Location

Write-Host "Starting Smart Cost Optimizer frontend preview..."
$frontendJob = Start-Process -FilePath "npm" -ArgumentList "run", "preview", "--", "--host=0.0.0.0", "--port=$FrontendPort" -WorkingDirectory "$root\frontend" -PassThru

Write-Host "Frontend PID: $($frontendJob.Id)"
Write-Host "Backend PID: $($backendJob.Id)"
Write-Host "Backend DATA_SOURCE: $DataSource"
Write-Host "Frontend URL: http://localhost:$FrontendPort/"
Write-Host "Press Ctrl+C in this terminal to stop waiting. Processes continue running."
