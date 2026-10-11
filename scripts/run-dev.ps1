# Local dev launcher for Repository Detective (no Docker required)
$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot\..

if (-not (Test-Path logs)) {
    New-Item -ItemType Directory -Path logs | Out-Null
}

$env:REPOSITORY_DETECTIVE_SKIP_STARTUP_CHECKS = "true"
$env:REPOSITORY_DETECTIVE_GITEA_URL = "https://git.commsnet.org"
$env:REPOSITORY_DETECTIVE_GITEA_TOKEN = "demo-token"
$env:REPOSITORY_DETECTIVE_AI_PROVIDER = "ollama"
$env:REPOSITORY_DETECTIVE_AI_BASE_URL = "http://127.0.0.1:11434/v1"
$env:REPOSITORY_DETECTIVE_AI_MODEL = "llama3.2"
$env:REPOSITORY_DETECTIVE_API_KEY = "demo-repository-detective-key"
$env:REPOSITORY_DETECTIVE_WEBHOOK_SECRET = "demo-webhook-secret"
$env:REPOSITORY_DETECTIVE_AUTO_CREATE_ISSUES = "false"
$env:REPOSITORY_DETECTIVE_LOG_LEVEL = "info"

Write-Host "Starting Repository Detective on http://localhost:8080"
Write-Host "  Health:  http://localhost:8080/health"
Write-Host "  Status:  http://localhost:8080/api/v1/status  (header X-Repository-Detective-API-Key: demo-repository-detective-key)"
Write-Host ""

go run .
