package mcp

import (
	"encoding/json"
	"os"
	"strings"

	"magic/internal/secrets"
)

// Inspect reports which canonical server names already exist in a harness
// target file, and whether that file exists. Read-only; a malformed file is
// reported as existing with nothing detected.
func Inspect(t Target, userHome string, names []string) (present map[string]bool, exists bool, err error) {
	present = map[string]bool{}
	path := t.Path(userHome)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return present, false, nil
		}
		return nil, false, err
	}
	switch t.Format {
	case FormatTOML:
		text := string(raw)
		for _, n := range names {
			if strings.Contains(text, "["+t.TopKey+"."+tomlKey(n)+"]") {
				present[n] = true
			}
		}
	default:
		root := map[string]any{}
		if json.Unmarshal(secrets.StripJSONC(raw), &root) == nil {
			if obj := nestedMap(root, t.TopKey); obj != nil {
				for _, n := range names {
					if _, ok := obj[n]; ok {
						present[n] = true
					}
				}
			}
		}
	}
	return present, true, nil
}

// nestedMap resolves a dotted key path against a decoded JSON object.
func nestedMap(root map[string]any, key string) map[string]any {
	cur := root
	for _, part := range strings.Split(key, ".") {
		next, _ := cur[part].(map[string]any)
		if next == nil {
			return nil
		}
		cur = next
	}
	return cur
}
