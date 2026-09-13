package secrets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ProvidersFileName is the local, always-plaintext provider key store under
// secrets/. The file is a JSON object keyed by key_env (e.g. "DEEPSEEK_API_KEY").
//
// Per decisions.md D2 the local store is never encrypted: `.enc` exists only as
// the cloud object name produced by `magic sync push` and consumed by
// `magic sync pull`. There is no local encrypted state and no MAGIC_SECRETS_MODE.
const ProvidersFileName = "providers.json"

// ProvidersPath returns $MAGIC_HOME/secrets/providers.json.
func ProvidersPath(secretsDir string) string {
	return filepath.Join(secretsDir, ProvidersFileName)
}

// LoadProviderSecrets loads the local plaintext provider key map.
// A missing file yields an empty map and a nil error.
func LoadProviderSecrets(secretsDir string) (map[string]string, error) {
	blob, err := os.ReadFile(ProvidersPath(secretsDir))
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	out := map[string]string{}
	if len(bytes.TrimSpace(blob)) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(blob, &out); err != nil {
		return nil, fmt.Errorf("parse %s: %w", ProvidersFileName, err)
	}
	return out, nil
}

// SetProviderSecret merges keyEnv→value into the plaintext store and rewrites it (0600).
func SetProviderSecret(secretsDir, keyEnv, value string) error {
	keyEnv = strings.TrimSpace(keyEnv)
	if keyEnv == "" {
		return fmt.Errorf("key_env must be non-empty")
	}
	m, err := LoadProviderSecrets(secretsDir)
	if err != nil {
		return err
	}
	m[keyEnv] = value
	if err := os.MkdirAll(secretsDir, 0o700); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	path := ProvidersPath(secretsDir)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// GetProviderSecret returns the value for keyEnv.
// ok is false when the key is absent (file missing or key not set).
func GetProviderSecret(secretsDir, keyEnv string) (string, bool, error) {
	keyEnv = strings.TrimSpace(keyEnv)
	if keyEnv == "" {
		return "", false, nil
	}
	m, err := LoadProviderSecrets(secretsDir)
	if err != nil {
		return "", false, err
	}
	v, ok := m[keyEnv]
	if !ok || v == "" {
		return "", false, nil
	}
	return v, true, nil
}

// ListProviderSecretNames returns sorted key_env names (values never returned).
func ListProviderSecretNames(secretsDir string) ([]string, error) {
	m, err := LoadProviderSecrets(secretsDir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	sort.Strings(names)
	return names, nil
}
