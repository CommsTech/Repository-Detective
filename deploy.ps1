# Repository Detective - Deployment Script
# Run this script to deploy Repository Detective

Write-Host "🚀 Deploying Repository Detective..." -ForegroundColor Green

# Check if Docker is running
try {
    docker version | Out-Null
    Write-Host "✅ Docker is running" -ForegroundColor Green
} catch {
    Write-Host "❌ Docker is not running. Please start Docker Desktop first." -ForegroundColor Red
    exit 1
}

# Check if docker-compose is available
try {
    docker-compose version | Out-Null
    Write-Host "✅ Docker Compose is available" -ForegroundColor Green
} catch {
    Write-Host "❌ Docker Compose is not available. Please install Docker Compose." -ForegroundColor Red
    exit 1
}

# Create logs directory if it doesn't exist
if (!(Test-Path "logs")) {
    New-Item -ItemType Directory -Path "logs" | Out-Null
    Write-Host "✅ Created logs directory" -ForegroundColor Green
}

# Create config directory if it doesn't exist
if (!(Test-Path "config")) {
    New-Item -ItemType Directory -Path "config" | Out-Null
    Write-Host "✅ Created config directory" -ForegroundColor Green
}

# Copy config template if config.yaml doesn't exist
if (!(Test-Path "config/config.yaml")) {
    Copy-Item "config/config.yaml" "config/config.yaml.backup" -ErrorAction SilentlyContinue
    Write-Host "✅ Backed up existing config (if any)" -ForegroundColor Green
}

Write-Host "🔧 Building and starting Repository Detective..." -ForegroundColor Yellow

# Build and start the services
docker-compose up -d --build

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Repository Detective deployed successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "📋 Next steps:" -ForegroundColor Cyan
    Write-Host "1. Check the logs: docker-compose logs -f repository-detective" -ForegroundColor White
    Write-Host "2. Test the health endpoint: http://127.0.0.1:8080/health" -ForegroundColor White
    Write-Host "3. Configure webhooks in your Gitea repositories to point to:" -ForegroundColor White
    Write-Host "   http://your-server-ip:8080/webhook" -ForegroundColor White
    Write-Host ""
    Write-Host "🔗 Repository Detective is now running on port 8080" -ForegroundColor Green
} else {
    Write-Host "❌ Deployment failed. Check the error messages above." -ForegroundColor Red
    exit 1
}
