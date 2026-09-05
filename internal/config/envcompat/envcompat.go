package envcompat

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const EnvPrefix = "REPOSITORY_DETECTIVE_"

// Apply merges REPOSITORY_DETECTIVE_* environment variables into viper.
func Apply(v *viper.Viper, logger *logrus.Logger) {
	if v == nil {
		return
	}
	_ = logger // reserved for future diagnostics

	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			continue
		}
		if !strings.HasPrefix(key, EnvPrefix) {
			continue
		}
		suffix := strings.TrimPrefix(key, EnvPrefix)
		configKey := configKeyFromSuffix(suffix)
		v.Set(configKey, value)
	}
}

// Resolve returns the effective value for a runner/core env key suffix from REPOSITORY_DETECTIVE_*.
func Resolve(suffix string) (string, bool) {
	if value, ok := os.LookupEnv(EnvPrefix + suffix); ok {
		return value, true
	}
	return "", false
}

func configKeyFromSuffix(suffix string) string {
	return strings.ToLower(suffix)
}

// EnvExample returns a documented env name using the product prefix.
func EnvExample(key string) string {
	key = strings.TrimPrefix(strings.ToUpper(key), EnvPrefix)
	return fmt.Sprintf("%s%s", EnvPrefix, key)
}
