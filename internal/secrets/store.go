package secrets

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PlaceholderPrefix marks reinjectable secret slots in per-harness custom files.
// apply replaces `${MAGIC_SECRET:<path>}` with the value from secrets/<adapter>.json.enc.
const PlaceholderPrefix = "${MAGIC_SECRET:"
const PlaceholderSuffix = "}"

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

// LoadSecrets decrypts secrets/<adapterID>.json.enc.
// Missing file → nil, nil. Present file with unset passphrase → error.
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

// ReinjectBytes replaces `${MAGIC_SECRET:<path>}` placeholders with values.
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
