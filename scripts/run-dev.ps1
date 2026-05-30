# Local dev launcher for Gitea Bugbot (no Docker required)
$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot\..

if (-not (Test-Path logs)) {
    New-Item -ItemType Directory -Path logs | Out-Null
}

$env:BUGBOT_SKIP_STARTUP_CHECKS = "true"
$env:BUGBOT_GITEA_URL = "https://git.commsnet.org"
$env:BUGBOT_GITEA_TOKEN = "demo-token"
$env:BUGBOT_AI_PROVIDER = "ollama"
$env:BUGBOT_AI_BASE_URL = "http://127.0.0.1:11434/v1"
$env:BUGBOT_AI_MODEL = "llama3.2"
$env:BUGBOT_API_KEY = "demo-bugbot-key"
$env:BUGBOT_WEBHOOK_SECRET = "demo-webhook-secret"
$env:BUGBOT_AUTO_CREATE_ISSUES = "false"
$env:BUGBOT_LOG_LEVEL = "info"

Write-Host "Starting Gitea Bugbot on http://localhost:8080"
Write-Host "  Health:  http://localhost:8080/health"
Write-Host "  Status:  http://localhost:8080/api/v1/status  (header X-Bugbot-API-Key: demo-bugbot-key)"
Write-Host ""

go run .
