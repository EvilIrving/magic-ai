package secrets

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var keyRE = regexp.MustCompile(`(?i)((?:api[_-]?key)|(?:(?:auth|access|refresh)[_-]?token)|password|secret|^key$|^access$|^refresh$|^token$|authorization)`)

// PlaceholderPrefix marks reinjectable secret slots in mirrored files.
const PlaceholderPrefix = "${MAGIC_SECRET:"
const PlaceholderSuffix = "}"

// RedactResult holds redacted body and extracted secrets.
type RedactResult struct {
	Data     []byte
	Secrets  map[string]string
	Redacted bool
	Warned   bool
}

// RedactBytes redacts known key fields in JSON or TOML-ish text.
func RedactBytes(path string, data []byte) RedactResult {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json", ".jsonc":
		if ext == ".jsonc" {
			data = StripJSONC(data)
		}
		return redactJSON(data)
	case ".toml":
		return redactTOML(data)
	default:
		stripped := StripJSONC(data)
		trim := strings.TrimSpace(string(stripped))
		if strings.HasPrefix(trim, "{") || strings.HasPrefix(trim, "[") {
			return redactJSON(stripped)
		}
		return RedactResult{Data: data}
	}
}

// StripJSONC removes // and /* */ comments outside of strings.
func StripJSONC(data []byte) []byte {
	s := string(data)
	var b strings.Builder
	b.Grow(len(s))
	inStr, escape := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			b.WriteByte(c)
			if escape {
				escape = false
			} else if c == '\\' {
				escape = true
			} else if c == '"' {
				inStr = false
			}
			continue
		}
		if c == '"' {
			inStr = true
			b.WriteByte(c)
			continue
		}
		if c == '/' && i+1 < len(s) {
			switch s[i+1] {
			case '/':
				i += 2
				for i < len(s) && s[i] != '\n' {
					i++
				}
				if i < len(s) {
					b.WriteByte('\n')
				}
				continue
			case '*':
				i += 2
				for i+1 < len(s) && !(s[i] == '*' && s[i+1] == '/') {
					i++
				}
				if i+1 < len(s) {
					i++
				}
				continue
			}
		}
		b.WriteByte(c)
	}
	return []byte(b.String())
}

func isEnvPlaceholder(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	if strings.HasPrefix(s, PlaceholderPrefix) {
		return true
	}
	if strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") {
		return true
	}
	if len(s) >= 2 && s[0] == '$' {
		for _, r := range s[1:] {
			if (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
				return false
			}
		}
		return true
	}
	return false
}

func redactJSON(data []byte) RedactResult {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return RedactResult{Data: data}
	}
	secrets := map[string]string{}
	changed := walkRedact(v, "", secrets)
	if !changed {
		return RedactResult{Data: data}
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return RedactResult{Data: data}
	}
	out = append(out, '\n')
	return RedactResult{Data: out, Secrets: secrets, Redacted: true}
}

func walkRedact(v any, prefix string, secrets map[string]string) bool {
	changed := false
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			path := k
			if prefix != "" {
				path = prefix + "." + k
			}
			if keyRE.MatchString(k) {
				if s, ok := val.(string); ok && s != "" && !isEnvPlaceholder(s) {
					secrets[path] = s
					t[k] = PlaceholderPrefix + path + PlaceholderSuffix
					changed = true
				}
			} else if walkRedact(val, path, secrets) {
				changed = true
			}
		}
	case []any:
		for i, val := range t {
			path := fmt.Sprintf("%s[%d]", prefix, i)
			if walkRedact(val, path, secrets) {
				changed = true
			}
		}
	}
	return changed
}

func redactTOML(data []byte) RedactResult {
	lines := strings.Split(string(data), "\n")
	secrets := map[string]string{}
	changed := false
	keyLine := regexp.MustCompile(`(?i)^(\s*)([A-Za-z0-9_.-]*(?:api[_-]?key|auth[_-]?token|password|secret)[A-Za-z0-9_.-]*)\s*=\s*"(.*)"\s*$`)
	for i, line := range lines {
		m := keyLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		key, val := m[2], m[3]
		if val == "" || isEnvPlaceholder(val) {
			continue
		}
		secrets[key] = val
		lines[i] = m[1] + key + ` = "` + PlaceholderPrefix + key + PlaceholderSuffix + `"`
		changed = true
	}
	if !changed {
		return RedactResult{Data: data}
	}
	return RedactResult{Data: []byte(strings.Join(lines, "\n")), Secrets: secrets, Redacted: true}
}

// StoreSecrets writes extracted secrets if passphrase set; else returns warn.
func StoreSecrets(secretsDir, adapterID string, secrets map[string]string, passphrase string) (warned bool, err error) {
	if len(secrets) == 0 {
		return false, nil
	}
	if passphrase == "" {
		return true, nil
	}
	if err := os.MkdirAll(secretsDir, 0o700); err != nil {
		return false, err
	}
	payload, _ := json.MarshalIndent(secrets, "", "  ")
	enc, err := Encrypt(payload, passphrase)
	if err != nil {
		return false, err
	}
	path := filepath.Join(secretsDir, adapterID+".json.enc")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, enc, 0o600); err != nil {
		return false, err
	}
	return false, os.Rename(tmp, path)
}

// LoadSecrets decrypts secrets/<adapterID>.json.enc.
func LoadSecrets(secretsDir, adapterID, passphrase string) (map[string]string, error) {
	path := filepath.Join(secretsDir, adapterID+".json.enc")
	blob, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if passphrase == "" {
		return nil, fmt.Errorf("secrets file present but MAGIC_PASSPHRASE unset")
	}
	pt, err := Decrypt(blob, passphrase)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	if err := json.Unmarshal(pt, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ReinjectBytes replaces ${MAGIC_SECRET:path} placeholders with values.
func ReinjectBytes(data []byte, secrets map[string]string) []byte {
	if len(secrets) == 0 {
		return data
	}
	s := string(data)
	for k, v := range secrets {
		ph := PlaceholderPrefix + k + PlaceholderSuffix
		s = strings.ReplaceAll(s, ph, v)
	}
	return []byte(s)
}

// RedactReinjectRoundTrip is a convenience for tests: redact then reinject.
func RedactReinjectRoundTrip(path string, data []byte) (redacted, restored []byte, secrets map[string]string, ok bool) {
	r := RedactBytes(path, data)
	if !r.Redacted {
		return data, data, nil, false
	}
	return r.Data, ReinjectBytes(r.Data, r.Secrets), r.Secrets, true
}
