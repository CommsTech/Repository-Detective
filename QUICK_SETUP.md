# 🚀 Quick Setup Guide - Gitea Bugbot Plugin

## Your Configuration is Ready! ✅

Your Docker Compose file is configured with placeholder values. You need to add your specific configuration:

- **Gitea Server**: `https://git.commsnet.org`
- **OpenWebUI Server**: `https://ai.commsnet.org/api`
- **AI Model**: `luna-tic-coder`

## 🐳 Deploy in 3 Simple Steps

### Step 1: Start Docker
Make sure Docker Desktop is running on your Windows machine.

### Step 2: Configure Your Settings
```powershell
# Copy the production configuration template
Copy-Item "docker-compose.prod.yml" "docker-compose.override.yml"

# Edit the override file with your actual values
notepad docker-compose.override.yml
```

### Step 3: Run the Deployment Script
```powershell
# In PowerShell, navigate to your project directory
cd C:\Users\commstech\Github\Gitea_AI_Bugbot

# Option 1: Use the build-and-deploy script (recommended for compatibility issues)
.\build-and-deploy.ps1

# Option 2: Use the original deployment script
.\deploy.ps1
```

### Step 3: Verify Deployment
The script will automatically:
- Build the Docker image
- Start the bugbot service
- Show you the next steps

## 🔧 Manual Deployment (Alternative)

If you prefer to deploy manually:

```powershell
# Option 1: Use the minimal configuration (recommended for compatibility issues)
docker-compose -f docker-compose.minimal.yml up -d

# Option 2: Use the simplified configuration
docker-compose -f docker-compose.simple.yml up -d --build

# Option 3: Use the full configuration
docker-compose up -d --build
```

# Check the logs
docker-compose logs -f gitea-bugbot

# Test the health endpoint
curl http://localhost:8080/health
```

## 🌐 Configure Webhooks

Once deployed, set up webhooks in your Gitea repositories:

1. Go to your repository settings in Gitea
2. Navigate to "Webhooks" → "Add webhook" → "Gitea"
3. Set the webhook URL to: `http://your-server-ip:8080/webhook`
4. Choose events: "Push" and "Pull Request"
5. Save the webhook

## 📊 Monitor Your Bugbot

- **Logs**: `docker-compose logs -f gitea-bugbot`
- **Status**: `http://localhost:8080/health`
- **Stop**: `docker-compose down`
- **Restart**: `docker-compose restart gitea-bugbot`

## 🔒 Security Note

**Important**: Change the webhook secret in `docker-compose.yml`:
```yaml
BUGBOT_WEBHOOKS_SECRET=your-secure-webhook-secret-here
```

Replace with a strong, random string for production use.

## 🎯 What Happens Next

Once deployed and webhooks are configured:
1. **Push code** to any repository → Bugbot automatically analyzes it
2. **Create pull requests** → Bugbot reviews the changes
3. **Security issues** are automatically detected and logged
4. **AI-powered suggestions** are provided for fixes
5. **Issues are created** in your Gitea repositories

## 🆘 Need Help?

- Check the logs: `docker-compose logs gitea-bugbot`
- Verify configuration: `docker-compose config`
- Restart the service: `docker-compose restart gitea-bugbot`

---

**Your Gitea Bugbot Plugin is ready to deploy!** 🎉
