package provider

import (
	"os"
	"path/filepath"
	"strings"
)

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func readFileString(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(b), nil
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	_ = os.Chmod(path, mode)
	return nil
}

func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	parts := strings.Split(text, "\n")
	for len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return "\n"
	}
	return strings.Join(lines, "\n") + "\n"
}

func lineKey(line string) (string, bool) {
	trim := strings.TrimSpace(line)
	if trim == "" || strings.HasPrefix(trim, "#") || !strings.Contains(trim, "=") {
		return "", false
	}
	key := strings.TrimSpace(strings.SplitN(trim, "=", 2)[0])
	return key, key != ""
}

func parseQuotedAssign(line, key string) (string, bool) {
	trim := strings.TrimSpace(line)
	if !strings.HasPrefix(trim, key) {
		return "", false
	}
	rest := strings.TrimSpace(trim[len(key):])
	if !strings.HasPrefix(rest, "=") {
		return "", false
	}
	rest = strings.TrimSpace(rest[1:])
	if len(rest) >= 2 {
		q := rest[0]
		if q == '"' || q == '\'' {
			end := strings.IndexByte(rest[1:], q)
			if end >= 0 {
				return rest[1 : 1+end], true
			}
		}
	}
	return "", false
}
