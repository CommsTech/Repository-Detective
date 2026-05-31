package security

import "os"

// SensitiveEnvKeys must not be passed to subprocesses scanning untrusted code.
var SensitiveEnvKeys = []string{
	"BUGBOT_API_KEY", "BUGBOT_GITEA_TOKEN", "BUGBOT_AI_API_KEY",
	"BUGBOT_OPENWEBUI_TOKEN", "BUGBOT_QDRANT_API_KEY", "BUGBOT_EMBEDDING_API_KEY",
	"BUGBOT_DATABASE_DSN", "BUGBOT_WEBHOOK_SECRET",
	"REPOSITORY_DETECTIVE_API_KEY", "REPOSITORY_DETECTIVE_GITEA_TOKEN", "REPOSITORY_DETECTIVE_AI_API_KEY",
	"REPOSITORY_DETECTIVE_OPENWEBUI_TOKEN", "REPOSITORY_DETECTIVE_QDRANT_API_KEY", "REPOSITORY_DETECTIVE_EMBEDDING_API_KEY",
	"REPOSITORY_DETECTIVE_DATABASE_DSN", "REPOSITORY_DETECTIVE_WEBHOOK_SECRET",
	"GITEA_TOKEN", "OPENAI_API_KEY", "ANTHROPIC_API_KEY", "DATABASE_URL",
}

// MinimalSubprocessEnv returns a safe environment for scanner/git subprocesses.
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
	for _, key := range []string{"HOME", "USERPROFILE", "SystemRoot", "TEMP", "TMP", "APPDATA", "LOCALAPPDATA"} {
		if v := os.Getenv(key); v != "" {
			env = append(env, key+"="+v)
		}
	}
	return env
}

// SubprocessEnvExposesSecrets reports whether env would leak operator secrets.
func SubprocessEnvExposesSecrets() bool {
	minimal := MinimalSubprocessEnv()
	for _, key := range SensitiveEnvKeys {
		if os.Getenv(key) == "" {
			continue
		}
		prefix := key + "="
		for _, entry := range minimal {
			if len(entry) >= len(prefix) && entry[:len(prefix)] == prefix {
				return true
			}
		}
	}
	return false
}
