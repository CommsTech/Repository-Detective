# Gitea Bugbot Plugin

A powerful automated code review and bug detection plugin for Gitea that uses AI to analyze code, detect issues, and automatically create issues with fix proposals.

## Features

- **Automated Code Analysis**: Automatically analyzes code on every push and pull request
- **AI-Powered Review**: Uses OpenWebUI AI for intelligent code analysis
- **Multi-Language Support**: Supports Go, Python, JavaScript, Java, C++, and many more
- **Security Scanning**: Detects security vulnerabilities and best practice violations
- **Quality Assessment**: Identifies code quality issues and performance problems
- **Automatic Issue Creation**: Creates detailed Gitea issues with AI-generated content
- **Fix Proposals**: Provides specific suggestions and code examples for fixes
- **Webhook Integration**: Seamlessly integrates with Gitea webhook system
- **Configurable Analysis**: Customizable analysis depth, file size limits, and skip patterns

## Architecture

The plugin consists of several key components:

- **Webhook Handler**: Processes Gitea webhook events
- **Analysis Engine**: Coordinates code analysis and AI integration
- **OpenWebUI Client**: Communicates with your OpenWebUI AI server
- **Gitea Client**: Interacts with Gitea API for repository access and issue creation
- **Issue Manager**: Creates and manages Gitea issues based on analysis results

## Prerequisites

- Gitea server (self-hosted)
- OpenWebUI server with AI models
- Go 1.21+ (for building from source)
- Docker (for containerized deployment)

## Quick Start

### 1. Configuration

Create a configuration file `config/config.yaml`:

```yaml
# Gitea Configuration
gitea_url: "http://your-gitea-server:3000"
gitea_token: "your-gitea-access-token"
webhook_secret: "your-webhook-secret"

# OpenWebUI Configuration
openwebui_url: "http://your-openwebui-server:8080"
openwebui_token: "your-openwebui-api-token"

# Analysis Configuration
auto_create_issues: true
max_issues_per_run: 50
analysis_depth: 3
```

### 2. Docker Deployment

```bash
# Clone the repository
git clone https://github.com/yourusername/gitea-bugbot.git
cd gitea-bugbot

# Update configuration
# Edit config/config.yaml with your settings

# Build and run
docker-compose up -d
```

### 3. Gitea Webhook Setup

1. Go to your Gitea repository
2. Navigate to Settings → Webhooks
3. Add new webhook:
   - **Target URL**: `http://your-bugbot-server:8080/webhook`
   - **HTTP Method**: POST
   - **Post Content Type**: application/json
   - **Secret**: Your webhook secret
   - **Trigger On**: Push events, Pull request events
4. Save the webhook

### 4. Test the Integration

Make a push to your repository or create a pull request. The bugbot will automatically analyze the code and create issues for any problems found.

## Configuration

### Environment Variables

All configuration options can be set via environment variables with the `BUGBOT_` prefix:

```bash
BUGBOT_GITEA_URL=http://your-gitea-server:3000
BUGBOT_GITEA_TOKEN=your-token
BUGBOT_OPENWEBUI_URL=http://your-openwebui-server:8080
BUGBOT_OPENWEBUI_TOKEN=your-token
```

### Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `port` | `8080` | HTTP server port |
| `log_level` | `info` | Logging level (debug, info, warn, error) |
| `gitea_url` | - | Gitea server URL (required) |
| `gitea_token` | - | Gitea access token (required) |
| `webhook_secret` | - | Webhook secret for verification |
| `openwebui_url` | - | OpenWebUI server URL (required) |
| `openwebui_token` | - | OpenWebUI API token |
| `auto_create_issues` | `true` | Automatically create issues for problems |
| `max_issues_per_run` | `50` | Maximum issues to create per analysis |
| `analysis_depth` | `3` | Directory analysis depth |
| `max_file_size` | `1048576` | Maximum file size to analyze (1MB) |
| `skip_low_severity` | `false` | Skip low severity issues |
| `group_similar_issues` | `true` | Create summary issues for multiple problems |

## API Endpoints

### Health Check
```
GET /health
```

### Webhook Endpoint
```
POST /webhook
```

### Manual Analysis
```
POST /api/v1/analyze
Content-Type: application/json

{
  "owner": "username",
  "repository": "repo-name",
  "ref": "main",
  "type": "repository"
}
```

### Status
```
GET /api/v1/status
```

### Configuration Reload
```
POST /api/v1/config/reload
```

## Supported Languages

The plugin automatically detects and analyzes code in the following languages:

- **Go** (.go)
- **Python** (.py)
- **JavaScript** (.js, .jsx)
- **TypeScript** (.ts, .tsx)
- **Java** (.java)
- **C++** (.cpp, .cc, .cxx)
- **C** (.c)
- **C#** (.cs)
- **PHP** (.php)
- **Ruby** (.rb)
- **Rust** (.rs)
- **Swift** (.swift)
- **Kotlin** (.kt)
- **Scala** (.scala)
- **Shell** (.sh, .bash)
- **PowerShell** (.ps1)
- **SQL** (.sql)
- **HTML** (.html, .htm)
- **CSS** (.css, .scss, .sass)
- **Data** (.xml, .yaml, .yml, .json)
- **Markdown** (.md, .txt)

## Issue Types

The plugin detects and categorizes issues into several types:

### Security Issues
- SQL injection vulnerabilities
- XSS vulnerabilities
- CSRF vulnerabilities
- Insecure authentication
- Data exposure risks

### Code Quality Issues
- Code complexity
- Maintainability problems
- Readability issues
- Code style violations
- Best practice violations

### Performance Issues
- Inefficient algorithms
- Memory leaks
- Resource management
- Optimization opportunities

### Bug Detection
- Logic errors
- Edge cases
- Error handling
- Input validation

## Customization

### Issue Templates

Customize issue titles and bodies using template variables:

```yaml
issue_title_template: "[{{severity}}] {{title}} in {{file}}"
issue_body_template: |
  ## Problem
  
  {{description}}
  
  ## Context
  
  - **File:** {{file}}
  - **Severity:** {{severity}}
  - **Category:** {{category}}
  
  ## Suggested Fix
  
  {{suggestion}}
```

### Skip Patterns

Configure which files and directories to skip:

```yaml
skip_patterns:
  - "node_modules"
  - "vendor"
  - ".git"
  - "build"
  - "dist"
  - "coverage"
  - "*.min.js"
  - "*.bundle.js"
```

### Language Mapping

Customize language detection:

```yaml
language_mapping:
  ".vue": "vue"
  ".svelte": "svelte"
  ".elm": "elm"
  ".clj": "clojure"
```

## Development

### Building from Source

```bash
# Clone repository
git clone https://github.com/yourusername/gitea-bugbot.git
cd gitea-bugbot

# Install dependencies
go mod download

# Build
go build -o gitea-bugbot .

# Run
./gitea-bugbot
```

### Project Structure

```
gitea-bugbot/
├── main.go                 # Main application entry point
├── config/                 # Configuration files
├── handlers/               # Webhook and HTTP handlers
├── analyzers/              # Code analysis engine
├── ai/                     # OpenWebUI AI integration
├── gitea/                  # Gitea API client
├── issues/                 # Issue management
├── Dockerfile              # Docker build file
├── docker-compose.yml      # Docker deployment
└── README.md               # This file
```

## Troubleshooting

### Common Issues

1. **Connection Failed to Gitea**
   - Verify Gitea URL and token
   - Check network connectivity
   - Ensure token has appropriate permissions

2. **Connection Failed to OpenWebUI**
   - Verify OpenWebUI URL and token
   - Check if OpenWebUI is running
   - Verify API endpoint availability

3. **Webhook Not Triggering**
   - Check webhook URL configuration
   - Verify webhook secret
   - Check Gitea webhook logs

4. **No Issues Created**
   - Verify `auto_create_issues` is enabled
   - Check analysis logs for errors
   - Verify repository permissions

### Logs

Enable debug logging to troubleshoot issues:

```yaml
log_level: "debug"
```

### Health Check

Monitor the plugin health:

```bash
curl http://localhost:8080/health
```

## Security Considerations

- Store sensitive tokens securely
- Use HTTPS for production deployments
- Implement proper webhook secret verification
- Limit API access to necessary endpoints
- Monitor and log all activities

## Performance

- Configure appropriate file size limits
- Set reasonable analysis depth
- Use concurrent analysis limits
- Monitor memory and CPU usage
- Implement rate limiting for large repositories

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions:

- Create an issue on GitHub
- Check the troubleshooting section
- Review the configuration examples
- Consult the API documentation

## Roadmap

- [ ] Support for more programming languages
- [ ] Advanced issue deduplication
- [ ] Custom analysis rules
- [ ] Integration with CI/CD pipelines
- [ ] Web-based configuration interface
- [ ] Analytics and reporting
- [ ] Team collaboration features
- [ ] Automated fix suggestions
- [ ] Performance benchmarking
- [ ] Security compliance reporting
