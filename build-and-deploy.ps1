# Repository Detective - Build and Deploy Script
# This script builds the Docker image first, then deploys

Write-Host "🚀 Building and Deploying Repository Detective..." -ForegroundColor Green

# Check if Docker is running
try {
    docker version | Out-Null
    Write-Host "✅ Docker is running" -ForegroundColor Green
} catch {
    Write-Host "❌ Docker is not running. Please start Docker Desktop first." -ForegroundColor Red
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

Write-Host "🔧 Building Docker image..." -ForegroundColor Yellow

# Build the Docker image
docker build -t repository-detective:latest .

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Docker image built successfully!" -ForegroundColor Green
    
    Write-Host "🚀 Deploying Repository Detective..." -ForegroundColor Yellow
    
    # Deploy using the minimal compose file
    docker-compose -f docker-compose.minimal.yml up -d
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Repository Detective deployed successfully!" -ForegroundColor Green
        Write-Host ""
        Write-Host "📋 Next steps:" -ForegroundColor Cyan
        Write-Host "1. Check the logs: docker-compose -f docker-compose.minimal.yml logs -f repository-detective" -ForegroundColor White
        Write-Host "2. Test the health endpoint: http://localhost:8080/health" -ForegroundColor White
        Write-Host "3. Configure webhooks in your Gitea repositories to point to:" -ForegroundColor White
        Write-Host "   http://your-server-ip:8080/webhook" -ForegroundColor White
        Write-Host ""
        Write-Host "🔗 Repository Detective is now running on port 8080" -ForegroundColor Green
    } else {
        Write-Host "❌ Deployment failed. Check the error messages above." -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "❌ Docker build failed. Check the error messages above." -ForegroundColor Red
    exit 1
}
