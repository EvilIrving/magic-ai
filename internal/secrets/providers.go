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

// ProvidersFileName is the encrypted map of provider API keys under secrets/.
// JSON plaintext is map[string]string keyed by key_env (e.g. "DEEPSEEK_API_KEY").
const ProvidersFileName = "providers.json.enc"

// ProvidersPlaintextFileName is the local-dev, unencrypted counterpart of
// ProvidersFileName. Same JSON shape and key_env keys, written 0600, never
// synced. It is used only when no encrypted store exists, or when the user
// forces it with MAGIC_SECRETS_MODE=plaintext.
const ProvidersPlaintextFileName = "providers.json"

// SecretMode selects which on-disk store is read and written.
type SecretMode string

const (
	// SecretModeEncrypted is the product default: AES-GCM + MAGIC_PASSPHRASE.
	SecretModeEncrypted SecretMode = "encrypted"
	// SecretModePlaintext is the local-dev store and needs no passphrase.
	SecretModePlaintext SecretMode = "plaintext"
)

func providersPath(secretsDir string) string {
	return filepath.Join(secretsDir, ProvidersFileName)
}

func providersPlaintextPath(secretsDir string) string {
	return filepath.Join(secretsDir, ProvidersPlaintextFileName)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ResolveSecretMode returns the active store mode for secretsDir.
//
// MAGIC_SECRETS_MODE=encrypted|plaintext wins. Otherwise:
//   - an existing providers.json.enc → encrypted (encrypted always wins, so an
//     installed product never silently downgrades to plaintext)
//   - else an existing providers.json → plaintext (local dev)
//   - else encrypted (default, so writes create .enc)
func ResolveSecretMode(secretsDir string) SecretMode {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MAGIC_SECRETS_MODE"))) {
	case string(SecretModePlaintext):
		return SecretModePlaintext
	case string(SecretModeEncrypted):
		return SecretModeEncrypted
	}
	if fileExists(providersPath(secretsDir)) {
		return SecretModeEncrypted
	}
	if fileExists(providersPlaintextPath(secretsDir)) {
		return SecretModePlaintext
	}
	return SecretModeEncrypted
}

// SecretStorePath returns the file the active mode reads and writes.
func SecretStorePath(secretsDir string) string {
	if ResolveSecretMode(secretsDir) == SecretModePlaintext {
		return providersPlaintextPath(secretsDir)
	}
	return providersPath(secretsDir)
}

// LoadProviderSecrets loads the active provider secret store.
// Missing file → empty map, nil error. In encrypted mode, a present file with
// an unset passphrase, or a decrypt failure, returns an error.
func LoadProviderSecrets(secretsDir, passphrase string) (map[string]string, error) {
	if ResolveSecretMode(secretsDir) == SecretModePlaintext {
		return loadPlaintextSecrets(secretsDir)
	}
	return loadEncryptedSecrets(secretsDir, passphrase)
}

func loadEncryptedSecrets(secretsDir, passphrase string) (map[string]string, error) {
	blob, err := os.ReadFile(providersPath(secretsDir))
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

func loadPlaintextSecrets(secretsDir string) (map[string]string, error) {
	blob, err := os.ReadFile(providersPlaintextPath(secretsDir))
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
		return nil, fmt.Errorf("parse %s: %w", ProvidersPlaintextFileName, err)
	}
	return out, nil
}

// SetProviderSecret merges keyEnv→value into the active store and rewrites it.
// Encrypted mode requires a non-empty passphrase; plaintext mode does not.
func SetProviderSecret(secretsDir, passphrase, keyEnv, value string) error {
	keyEnv = strings.TrimSpace(keyEnv)
	if keyEnv == "" {
		return fmt.Errorf("key_env must be non-empty")
	}
	if ResolveSecretMode(secretsDir) == SecretModePlaintext {
		return setPlaintextSecret(secretsDir, keyEnv, value)
	}
	if passphrase == "" {
		return fmt.Errorf("MAGIC_PASSPHRASE required to set provider secrets (or set MAGIC_SECRETS_MODE=plaintext for a local, unencrypted store)")
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

func setPlaintextSecret(secretsDir, keyEnv, value string) error {
	m, err := loadPlaintextSecrets(secretsDir)
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
	path := providersPlaintextPath(secretsDir)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// GetProviderSecret returns the value for keyEnv from the active providers map.
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
// Works in both encrypted and plaintext modes.
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
