package options

import (
	"os"
)

const (
	namespaceEnvVar        = "NAMESPACE"
	namespaceEnvVarDefault = "default"
)

// GetNamespace returns the namespace from the environment variable NAMESPACE or the default value "default".
func GetNamespace() string {
	return getStringFromEnvVar(namespaceEnvVar, namespaceEnvVarDefault)
}

// getStringFromEnvVar returns the value of the environment variable with the given name.
// If the environment variable is not set, it returns the default value.
func getStringFromEnvVar(name, defaultVal string) string {
	value, exists := os.LookupEnv(name)
	if !exists {
		value = defaultVal
	}
	return value
}
