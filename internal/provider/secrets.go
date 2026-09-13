package provider

import (
	"fmt"
	"os"
	"strings"

	"magic/internal/secrets"
)

// ResolveSecret loads keyEnv from Magic provider secrets.
// Returns error instructing magic secrets set when missing.
func ResolveSecret(secretsDir, keyEnv string) (string, error) {
	keyEnv = strings.TrimSpace(keyEnv)
	if keyEnv == "" {
		return "", nil
	}
	pass := os.Getenv("MAGIC_PASSPHRASE")
	v, ok, err := secrets.GetProviderSecret(secretsDir, pass, keyEnv)
	if err != nil {
		return "", err
	}
	if !ok || strings.TrimSpace(v) == "" {
		return "", fmt.Errorf("%s is not set — run: magic secrets set %s --value … (requires MAGIC_PASSPHRASE)", keyEnv, keyEnv)
	}
	return v, nil
}

// SecretPresent reports whether keyEnv exists in Magic secrets (no error if file missing).
func SecretPresent(secretsDir, keyEnv string) bool {
	keyEnv = strings.TrimSpace(keyEnv)
	if keyEnv == "" {
		return true // official / no key required
	}
	pass := os.Getenv("MAGIC_PASSPHRASE")
	v, ok, err := secrets.GetProviderSecret(secretsDir, pass, keyEnv)
	if err != nil || !ok {
		return false
	}
	return strings.TrimSpace(v) != ""
}

// SetSecret wraps secrets.SetProviderSecret. The passphrase is required only
// when the active store is encrypted; the plaintext local store needs none.
func SetSecret(secretsDir, keyEnv, value string) error {
	return secrets.SetProviderSecret(secretsDir, os.Getenv("MAGIC_PASSPHRASE"), keyEnv, value)
}
