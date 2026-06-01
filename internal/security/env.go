package security

import (
	"os"
	"strings"
)

// SensitiveEnvKeys must not be passed to subprocesses scanning untrusted code.
var SensitiveEnvKeys = []string{
	"BUGBOT_API_KEY", "BUGBOT_GITEA_TOKEN", "BUGBOT_AI_API_KEY",
	"BUGBOT_OPENWEBUI_TOKEN", "BUGBOT_QDRANT_API_KEY", "BUGBOT_EMBEDDING_API_KEY",
	"BUGBOT_DATABASE_DSN", "BUGBOT_WEBHOOK_SECRET",
	"REPOSITORY_DETECTIVE_API_KEY", "REPOSITORY_DETECTIVE_GITEA_TOKEN", "REPOSITORY_DETECTIVE_AI_API_KEY",
	"REPOSITORY_DETECTIVE_OPENWEBUI_TOKEN", "REPOSITORY_DETECTIVE_QDRANT_API_KEY", "REPOSITORY_DETECTIVE_EMBEDDING_API_KEY",
	"REPOSITORY_DETECTIVE_DATABASE_DSN", "REPOSITORY_DETECTIVE_WEBHOOK_SECRET",
	"GITEA_TOKEN", "OPENAI_API_KEY", "ANTHROPIC_API_KEY", "DATABASE_URL",
	"SSH_AUTH_SOCK", "SSH_AGENT_PID", "KUBECONFIG", "DOCKER_AUTH_CONFIG", "NETRC",
}

var sensitiveEnvPrefixes = []string{
	"AWS_", "AZURE_", "GCP_", "GOOGLE_", "GITHUB_", "GITLAB_", "NPM_", "PYPI_",
	"DOCKER_", "KUBE_", "K8S_", "BUGBOT_", "REPOSITORY_DETECTIVE_",
}

// MinimalSubprocessEnv returns a whitelist-only environment for scanner/git subprocesses.
func MinimalSubprocessEnv() []string {
	env := []string{
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=true",
		"GIT_SSH_COMMAND=disabled",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_NOGLOBAL=1",
	}
	if path := os.Getenv("PATH"); path != "" {
		env = append(env, "PATH="+path)
	}
	for _, key := range []string{"HOME", "USERPROFILE", "SystemRoot", "TEMP", "TMP", "APPDATA", "LOCALAPPDATA", "LANG", "LC_ALL"} {
		if v := os.Getenv(key); v != "" {
			env = append(env, key+"="+v)
		}
	}
	return env
}

func isSensitiveEnvKey(key string) bool {
	key = strings.ToUpper(strings.TrimSpace(key))
	for _, blocked := range SensitiveEnvKeys {
		if key == blocked {
			return true
		}
	}
	for _, prefix := range sensitiveEnvPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// SubprocessEnvExposesSecrets reports whether MinimalSubprocessEnv accidentally includes operator secrets.
func SubprocessEnvExposesSecrets() bool {
	minimal := MinimalSubprocessEnv()
	for _, entry := range minimal {
		key, _, ok := strings.Cut(entry, "=")
		if ok && isSensitiveEnvKey(key) {
			return true
		}
	}
	return false
}
