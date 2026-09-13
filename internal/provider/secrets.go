package provider

import (
	"fmt"
	"strings"

	"magic/internal/secrets"
)

// ResolveSecret loads keyEnv from the local plaintext Magic provider secrets.
// Returns an error instructing `magic secrets set` when missing.
func ResolveSecret(secretsDir, keyEnv string) (string, error) {
	keyEnv = strings.TrimSpace(keyEnv)
	if keyEnv == "" {
		return "", nil
	}
	v, ok, err := secrets.GetProviderSecret(secretsDir, keyEnv)
	if err != nil {
		return "", err
	}
	if !ok || strings.TrimSpace(v) == "" {
		return "", fmt.Errorf("%s is not set — run: magic secrets set %s --value …", keyEnv, keyEnv)
	}
	return v, nil
}

// SecretPresent reports whether keyEnv exists in Magic secrets (no error if file missing).
func SecretPresent(secretsDir, keyEnv string) bool {
	keyEnv = strings.TrimSpace(keyEnv)
	if keyEnv == "" {
		return true // official / no key required
	}
	v, ok, err := secrets.GetProviderSecret(secretsDir, keyEnv)
	if err != nil || !ok {
		return false
	}
	return strings.TrimSpace(v) != ""
}

// SetSecret wraps secrets.SetProviderSecret. The local store is always plaintext.
func SetSecret(secretsDir, keyEnv, value string) error {
	return secrets.SetProviderSecret(secretsDir, keyEnv, value)
}
