package security

import (
	"os"
	"strings"
)

// SensitiveEnvKeys must not be passed to subprocesses scanning untrusted code.
var SensitiveEnvKeys = []string{
	"REPOSITORY_DETECTIVE_API_KEY", "REPOSITORY_DETECTIVE_GITEA_TOKEN", "REPOSITORY_DETECTIVE_AI_API_KEY",
	"REPOSITORY_DETECTIVE_OPENWEBUI_TOKEN",
	"REPOSITORY_DETECTIVE_DATABASE_DSN", "REPOSITORY_DETECTIVE_WEBHOOK_SECRET",
	"GITEA_TOKEN", "OPENAI_API_KEY", "ANTHROPIC_API_KEY", "DATABASE_URL",
	"SSH_AUTH_SOCK", "SSH_AGENT_PID", "KUBECONFIG", "DOCKER_AUTH_CONFIG", "NETRC",
}

var sensitiveEnvPrefixes = []string{
	"AWS_", "AZURE_", "GCP_", "GOOGLE_", "GITHUB_", "GITLAB_", "NPM_", "PYPI_",
	"DOCKER_", "KUBE_", "K8S_", "REPOSITORY_DETECTIVE_",
}

// MinimalSubprocessEnv returns a whitelist-only environment for scanner/git subprocesses.
func MinimalSubprocessEnv() []string {
	path := os.Getenv("PATH")
	if _, err := os.Stat("/usr/local/go/bin"); err == nil {
		path = "/usr/local/go/bin:" + path
	}
	env := []string{
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=true",
		"GIT_SSH_COMMAND=disabled",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_NOGLOBAL=1",
		"GOTOOLCHAIN=auto",
	}
	if path != "" {
		env = append(env, "PATH="+path)
	}
	home := os.Getenv("HOME")
	if home == "" {
		// Scanner tools (grype/trivy/syft) store DBs under $HOME/.cache.
		// Without HOME they fall back to /.cache and fail in the container.
		if st, err := os.Stat("/home/repositorydetective"); err == nil && st.IsDir() {
			home = "/home/repositorydetective"
		}
	}
	for _, key := range []string{
		"USERPROFILE", "SystemRoot", "TEMP", "TMP", "TMPDIR",
		"XDG_CACHE_HOME", "XDG_CONFIG_HOME", "APPDATA", "LOCALAPPDATA",
		"LANG", "LC_ALL",
	} {
		if v := os.Getenv(key); v != "" {
			env = append(env, key+"="+v)
		}
	}
	if home != "" {
		env = append(env, "HOME="+home)
		if os.Getenv("XDG_CACHE_HOME") == "" {
			env = append(env, "XDG_CACHE_HOME="+home+"/.cache")
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
