# Gitea Bugbot Plugin Architecture

## Overview
This plugin integrates with Gitea to automatically review code, detect bugs, and propose fixes using OpenWebUI AI assistance.

## Core Components

### 1. Gitea Plugin Integration
- **Plugin Type**: Gitea webhook plugin
- **Trigger**: Repository push events, pull request creation/updates
- **Integration Points**: Webhook system, issue tracker, comment system

### 2. Code Analysis Engine
- **Language Support**: Multi-language support (Go, Python, JavaScript, etc.)
- **Analysis Methods**: 
  - Static code analysis
  - Pattern recognition
  - Security vulnerability detection
  - Code quality metrics

### 3. AI Integration Layer
- **Multi-provider support**: OpenAI, Anthropic, OpenRouter, Ollama, Open WebUI, OpenClaw
- **Transport abstraction**: OpenAI-compatible and Anthropic Messages APIs
- **Configuration**: `ai_provider`, `ai_base_url`, `ai_api_key`, `ai_model`
- **Legacy compatibility**: `openwebui_url` / `openwebui_token` auto-map to Open WebUI provider

### 4. Issue Management
- **Automatic Issue Creation**: Create issues for detected bugs
- **Fix Proposals**: Generate and attach fix suggestions
- **Priority Classification**: Categorize issues by severity

## Technical Architecture

### Plugin Structure
```
gitea-bugbot/
├── main.go                 # Main plugin entry point
├── config/                 # Configuration management
├── handlers/               # Webhook and event handlers
├── analyzers/              # Code analysis modules
├── ai/                     # AI integration layer
├── issues/                 # Issue management
└── templates/              # Issue and comment templates
```

### Data Flow
1. Gitea webhook triggers `handlers.WebhookHandler` (rate limited + secret verified)
2. `AnalysisProcessor` in main.go runs CAH pipeline via `analyzers.Engine`
3. **Prepare** — map attack surface via Gitea file tree + AI
4. **Scan** — parallel auditor agents (SQL, XSS, auth, injection, crypto, config)
5. **Validate** — advocate/counsel debater agents filter false positives
6. **Dedup** — collapse findings by root cause
7. **Prove** — generate PoC commands for validated findings
8. Issues are created in Gitea via `issues.Manager`

## Module Path

```
git.commsnet.org/commstech/bugbot
```

## Configuration
- Gitea server connection details
- OpenWebUI server configuration
- Repository inclusion/exclusion rules
- Analysis depth and frequency settings
- Issue creation policies
