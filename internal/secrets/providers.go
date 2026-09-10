package secrets

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ProvidersFileName is the encrypted map of provider API keys under secrets/.
// JSON plaintext is map[string]string keyed by key_env (e.g. "DEEPSEEK_API_KEY").
const ProvidersFileName = "providers.json.enc"

func providersPath(secretsDir string) string {
	return filepath.Join(secretsDir, ProvidersFileName)
}

// LoadProviderSecrets decrypts secrets/providers.json.enc.
// Missing file → empty map, nil error. Decrypt failure or unset passphrase
// when the file exists → error.
func LoadProviderSecrets(secretsDir, passphrase string) (map[string]string, error) {
	path := providersPath(secretsDir)
	blob, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	if passphrase == "" {
		return nil, fmt.Errorf("%s present but MAGIC_PASSPHRASE unset — export MAGIC_PASSPHRASE to decrypt provider keys", ProvidersFileName)
	}
	pt, err := Decrypt(blob, passphrase)
	if err != nil {
		return nil, fmt.Errorf("decrypt %s: %w", ProvidersFileName, err)
	}
	out := map[string]string{}
	if err := json.Unmarshal(pt, &out); err != nil {
		return nil, fmt.Errorf("parse %s: %w", ProvidersFileName, err)
	}
	return out, nil
}

// SetProviderSecret merges keyEnv→value into the encrypted providers map and rewrites the file.
// Requires a non-empty passphrase.
func SetProviderSecret(secretsDir, passphrase, keyEnv, value string) error {
	keyEnv = strings.TrimSpace(keyEnv)
	if keyEnv == "" {
		return fmt.Errorf("key_env must be non-empty")
	}
	if passphrase == "" {
		return fmt.Errorf("MAGIC_PASSPHRASE required to set provider secrets")
	}
	m, err := LoadProviderSecrets(secretsDir, passphrase)
	if err != nil {
		return err
	}
	if m == nil {
		m = map[string]string{}
	}
	m[keyEnv] = value
	if err := os.MkdirAll(secretsDir, 0o700); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	enc, err := Encrypt(payload, passphrase)
	if err != nil {
		return err
	}
	path := providersPath(secretsDir)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, enc, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// GetProviderSecret returns the value for keyEnv from the encrypted providers map.
// ok is false when the key is absent (file missing or key not set).
func GetProviderSecret(secretsDir, passphrase, keyEnv string) (string, bool, error) {
	keyEnv = strings.TrimSpace(keyEnv)
	if keyEnv == "" {
		return "", false, nil
	}
	m, err := LoadProviderSecrets(secretsDir, passphrase)
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
func ListProviderSecretNames(secretsDir, passphrase string) ([]string, error) {
	m, err := LoadProviderSecrets(secretsDir, passphrase)
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
