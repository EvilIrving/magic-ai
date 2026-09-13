package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"magic/internal/adapter"
	"magic/internal/secrets"
)

// Options controls Apply.
type Options struct {
	MagicHome string
	UserHome  string
	DryRun    bool
	Only      string
	BackupDir string
	Adapters  []adapter.Adapter
}

// CanonicalRelPath is the packs/mcp file relative to MAGIC_HOME.
const CanonicalRelPath = "packs/mcp/mcp.yaml"

// Apply projects packs/mcp/mcp.yaml into every MCP-capable, detected adapter.
func Apply(opts Options) ([]string, error) {
	path := filepath.Join(opts.MagicHome, filepath.FromSlash(CanonicalRelPath))
	f, err := Load(path)
	if err != nil {
		return nil, err
	}
	if len(f.MCP) == 0 {
		return []string{"[mcp] skip (no packs/mcp/mcp.yaml entries)"}, nil
	}
	var log []string
	for _, a := range opts.Adapters {
		id := string(a.ID())
		if !a.Implemented() {
			continue
		}
		if opts.Only != "" && id != opts.Only {
			continue
		}
		if !a.Caps().MCP {
			log = append(log, fmt.Sprintf("[%s] mcp: n/a", id))
			continue
		}
		if !a.Detect(opts.UserHome) && opts.Only != id {
			log = append(log, fmt.Sprintf("[%s] mcp: skip (not detected)", id))
			continue
		}
		t, ok := TargetFor(id)
		if !ok {
			log = append(log, fmt.Sprintf("[%s] mcp: n/a", id))
			continue
		}
		logs, err := Project(t, opts.UserHome, f, opts.DryRun, opts.BackupDir)
		if err != nil {
			return log, fmt.Errorf("[%s] mcp: %w", id, err)
		}
		log = append(log, logs...)
	}
	return log, nil
}

// Project writes the enabled canonical servers into one harness target.
func Project(t Target, userHome string, f *File, dryRun bool, backupDir string) ([]string, error) {
	var log []string
	path := t.Path(userHome)
	entries := map[string]map[string]any{}
	skipped := 0
	for _, name := range f.Names() {
		s := f.MCP[name]
		if !s.IsEnabled() && !styleSupportsEnabled(t.Style) {
			log = append(log, fmt.Sprintf("[%s] mcp: n/a %s (disabled; %s has no enabled flag)", t.Harness, name, t.Harness))
			skipped++
			continue
		}
		entry, ok, reason := Translate(s, t.Style)
		if !ok {
			log = append(log, fmt.Sprintf("[%s] mcp: n/a %s — %s", t.Harness, name, reason))
			skipped++
			continue
		}
		entries[name] = entry
	}
	if len(entries) == 0 {
		if skipped > 0 {
			return log, nil
		}
		log = append(log, fmt.Sprintf("[%s] mcp: skip (nothing to write)", t.Harness))
		return log, nil
	}
	if dryRun {
		log = append(log, fmt.Sprintf("[%s] mcp: would merge %d server(s) → %s", t.Harness, len(entries), path))
		return log, nil
	}
	if backupDir != "" && fileExists(path) {
		if err := backupFile(path, backupDir, filepath.Join(t.Harness, filepath.Base(path))); err != nil {
			return log, err
		}
	}
	switch t.Format {
	case FormatTOML:
		text, err := readText(path)
		if err != nil {
			return log, err
		}
		out := upsertTOMLSections(text, t.TopKey, entries)
		if err := writeFile(path, []byte(out)); err != nil {
			return log, err
		}
	case FormatYAML:
		text, err := readText(path)
		if err != nil {
			return log, err
		}
		out, err := upsertYAMLSection(text, t.TopKey, entries)
		if err != nil {
			return log, fmt.Errorf("parse %s: %w", path, err)
		}
		if err := writeFile(path, out); err != nil {
			return log, err
		}
	default:
		if t.LenientJSON && !jsonConfigParseable(path) {
			log = append(log, fmt.Sprintf("[%s] mcp: n/a (config is not plain JSON; left untouched)", t.Harness))
			return log, nil
		}
		out, err := mergeJSONConfig(path, t.TopKey, entries)
		if err != nil {
			return log, err
		}
		if err := writeFile(path, out); err != nil {
			return log, err
		}
	}
	log = append(log, fmt.Sprintf("[%s] mcp: merged %d server(s) → %s", t.Harness, len(entries), path))
	return log, nil
}

func styleSupportsEnabled(style string) bool {
	switch style {
	case StyleCodex, StyleOpenCode, StylePi, StyleKiro, StyleZCode,
		StyleOMP, StyleHermes, StyleDimAgent:
		return true
	}
	return false
}

// jsonConfigParseable reports whether an existing file parses as plain JSON or
// JSONC. A missing/empty file counts as parseable (a fresh one is written).
func jsonConfigParseable(path string) bool {
	raw, err := os.ReadFile(path)
	if err != nil || len(strings.TrimSpace(string(raw))) == 0 {
		return true
	}
	var root map[string]any
	return json.Unmarshal(secrets.StripJSONC(raw), &root) == nil
}

// mergeJSONConfig sets root[topKey][name]=entry, preserving all other content.
// topKey may be a dotted path (e.g. "mcp.servers") for nested configs.
func mergeJSONConfig(path, topKey string, entries map[string]map[string]any) ([]byte, error) {
	root := map[string]any{}
	if raw, err := os.ReadFile(path); err == nil && len(strings.TrimSpace(string(raw))) > 0 {
		if err := json.Unmarshal(secrets.StripJSONC(raw), &root); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
	}
	obj := ensureNestedMap(root, topKey)
	for name, entry := range entries {
		obj[name] = entry
	}
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// ensureNestedMap walks/creates a dotted key path and returns the leaf object.
func ensureNestedMap(root map[string]any, key string) map[string]any {
	parts := strings.Split(key, ".")
	cur := root
	for _, part := range parts {
		next, _ := cur[part].(map[string]any)
		if next == nil {
			next = map[string]any{}
			cur[part] = next
		}
		cur = next
	}
	return cur
}

func readText(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(raw), nil
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func backupFile(path, backupDir, rel string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dst := filepath.Join(backupDir, rel)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, raw, 0o600)
}
