package envcompat

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	LegacyEnvPrefix = "BUGBOT_"
	NewEnvPrefix    = "REPOSITORY_DETECTIVE_"
)

var (
	deprecationOnce sync.Once
)

// Apply merges REPOSITORY_DETECTIVE_* environment variables into viper.
// When both legacy and new variables are set for the same key, the new prefix wins.
func Apply(v *viper.Viper, logger *logrus.Logger) {
	if v == nil {
		return
	}
	if logger == nil {
		logger = logrus.New()
	}

	conflictLogged := make(map[string]struct{})
	newPrefixUsed := false
	legacyOnlyUsed := false

	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			continue
		}

		switch {
		case strings.HasPrefix(key, NewEnvPrefix):
			newPrefixUsed = true
			suffix := strings.TrimPrefix(key, NewEnvPrefix)
			configKey := configKeyFromSuffix(suffix)
			legacyKey := LegacyEnvPrefix + suffix

			if legacyValue, legacySet := os.LookupEnv(legacyKey); legacySet && legacyValue != value {
				if _, logged := conflictLogged[configKey]; !logged {
					logger.Infof("Both %s and %s set; using %s", legacyKey, key, key)
					conflictLogged[configKey] = struct{}{}
				}
			}

			v.Set(configKey, value)

		case strings.HasPrefix(key, LegacyEnvPrefix):
			if _, newSet := os.LookupEnv(NewEnvPrefix + strings.TrimPrefix(key, LegacyEnvPrefix)); !newSet {
				legacyOnlyUsed = true
			}
		}
	}

	if legacyOnlyUsed && !newPrefixUsed {
		deprecationOnce.Do(func() {
			logger.Info("BUGBOT_* environment variables are supported but deprecated; prefer REPOSITORY_DETECTIVE_* (see docs/NAMING.md)")
		})
	}
}

// Resolve returns the effective value for a runner/core env key suffix,
// preferring REPOSITORY_DETECTIVE_* over BUGBOT_*.
func Resolve(suffix string) (string, bool) {
	if value, ok := os.LookupEnv(NewEnvPrefix + suffix); ok {
		return value, true
	}
	if value, ok := os.LookupEnv(LegacyEnvPrefix + suffix); ok {
		return value, true
	}
	return "", false
}

func configKeyFromSuffix(suffix string) string {
	return strings.ToLower(suffix)
}

// EnvExample returns a documented env name using the preferred prefix.
func EnvExample(key string) string {
	key = strings.TrimPrefix(strings.ToUpper(key), LegacyEnvPrefix)
	key = strings.TrimPrefix(key, NewEnvPrefix)
	return fmt.Sprintf("%s%s", NewEnvPrefix, key)
}
