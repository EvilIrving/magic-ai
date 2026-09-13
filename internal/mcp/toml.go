package mcp

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// upsertTOMLSections upserts `[section.<name>]` tables into TOML text.
// Managed keys in an existing table are replaced; the user's other keys and
// comments inside that table are preserved. Other tables are untouched.
func upsertTOMLSections(text, section string, entries map[string]map[string]any) string {
	lines := splitTOML(text)
	for _, name := range sortedKeys(entries) {
		header := "[" + section + "." + tomlKey(name) + "]"
		body := encodeTOMLBody(entries[name])
		lines = upsertTOMLTable(lines, header, body)
	}
	return joinTOML(lines)
}

func upsertTOMLTable(lines []string, header string, body []string) []string {
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == header {
			start = i
			break
		}
	}
	if start < 0 {
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			lines = append(lines, "")
		}
		out := append([]string{}, lines...)
		out = append(out, header)
		out = append(out, body...)
		return out
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
			end = i
			break
		}
	}
	merged := mergeTOMLBody(lines[start+1:end], body)
	out := append([]string{}, lines[:start+1]...)
	out = append(out, merged...)
	if end < len(lines) {
		out = append(out, lines[end:]...)
	}
	return out
}

// mergeTOMLBody puts managed lines first, then keeps the user's other keys.
func mergeTOMLBody(existing, managed []string) []string {
	managedKeys := map[string]bool{}
	for _, l := range managed {
		if k, ok := tomlLineKey(l); ok {
			managedKeys[k] = true
		}
	}
	out := append([]string{}, managed...)
	for _, l := range existing {
		if k, ok := tomlLineKey(l); ok && managedKeys[k] {
			continue
		}
		out = append(out, l)
	}
	return out
}

func encodeTOMLBody(entry map[string]any) []string {
	keys := make([]string, 0, len(entry))
	for k := range entry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, tomlKey(k)+" = "+encodeTOMLValue(entry[k]))
	}
	return out
}

func encodeTOMLValue(v any) string {
	switch t := v.(type) {
	case string:
		return strconv.Quote(t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case []string:
		parts := make([]string, 0, len(t))
		for _, e := range t {
			parts = append(parts, strconv.Quote(e))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case []any:
		parts := make([]string, 0, len(t))
		for _, e := range t {
			parts = append(parts, encodeTOMLValue(e))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, tomlKey(k)+" = "+encodeTOMLValue(t[k]))
		}
		return "{ " + strings.Join(parts, ", ") + " }"
	default:
		return strconv.Quote(fmt.Sprint(v))
	}
}

func tomlKey(name string) string {
	if isBareTOMLKey(name) {
		return name
	}
	return strconv.Quote(name)
}

func isBareTOMLKey(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

// tomlLineKey extracts the key from a `key = value` line (quoted or bare).
func tomlLineKey(line string) (string, bool) {
	trim := strings.TrimSpace(line)
	if trim == "" || strings.HasPrefix(trim, "#") || !strings.Contains(trim, "=") {
		return "", false
	}
	key := strings.TrimSpace(strings.SplitN(trim, "=", 2)[0])
	key = strings.Trim(key, `"'`)
	return key, key != ""
}

func splitTOML(text string) []string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func joinTOML(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
