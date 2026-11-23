
package config

import (
	"fmt"
	"os"
)

// Env returns environment variable or fallback value
func Env(key, fallback string) string {

	value := os.Getenv(key)

	if value != "" {
		return value
	}

	return fallback
}

// RequiredEnv returns an error if environment variable is missing
func RequiredEnv(key string) (string, error) {

	value := os.Getenv(key)

	if value == "" {
		return "", fmt.Errorf("required environment variable %s is missing", key)
	}

	return value, nil
}
