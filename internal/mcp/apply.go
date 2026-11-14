package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"magic/internal/adapter"
	"magic/internal/backup"
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
		logs, err := Project(t, opts.UserHome, a.MirrorRoot(opts.UserHome), f, opts.DryRun, opts.BackupDir)
		if err != nil {
			return log, fmt.Errorf("[%s] mcp: %w", id, err)
		}
		log = append(log, logs...)
	}
	return log, nil
}

// Project writes the enabled canonical servers into one harness target.
//
// mirrorRoot is that harness's configuration root. Backup entries record paths
// relative to it, so a target below a subdirectory (Kiro's settings/mcp.json)
// keeps its subdirectory in the backup set and rollback can put it back.
//
// The content comes from ProjectBytes: plan stacks its layers with the same
// computation, so a plan cannot describe a projection apply would not make.
func Project(t Target, userHome, mirrorRoot string, f *File, dryRun bool, backupDir string) ([]string, error) {
	// targetEntries is the same call ProjectBytes makes; here it also produces
	// the notes for the servers this dialect cannot hold.
	entries, log := targetEntries(t, f)
	if len(entries) == 0 {
		if len(log) > 0 {
			return log, nil
		}
		log = append(log, fmt.Sprintf("[%s] mcp: skip (nothing to write)", t.Harness))
		return log, nil
	}
	if dryRun {
		log = append(log, fmt.Sprintf("[%s] mcp: would merge %d server(s) → %s", t.Harness, len(entries), t.Path(userHome)))
		return log, nil
	}
	path, out, _, err := ProjectBytes(t, userHome, f)
	if err != nil {
		return log, err
	}
	if out == nil {
		log = append(log, fmt.Sprintf("[%s] mcp: n/a (config is not plain JSON; left untouched)", t.Harness))
		return log, nil
	}
	if err := recordProjection(t, userHome, mirrorRoot, path, out, backupDir); err != nil {
		return log, err
	}
	if err := writeFile(path, out); err != nil {
		return log, err
	}
	log = append(log, fmt.Sprintf("[%s] mcp: merged %d server(s) → %s", t.Harness, len(entries), path))
	return log, nil
}

// targetEntries translates every server of f that this target's dialect can
// hold. The notes explain the ones it cannot: a disabled server on a dialect
// without an enabled flag, or a transport the dialect has no field for.
func targetEntries(t Target, f *File) (entries map[string]map[string]any, notes []string) {
	entries = map[string]map[string]any{}
	for _, name := range f.Names() {
		s := f.MCP[name]
		if !s.IsEnabled() && !styleSupportsEnabled(t.Style) {
			notes = append(notes, fmt.Sprintf("[%s] mcp: n/a %s (disabled; %s has no enabled flag)", t.Harness, name, t.Harness))
			continue
		}
		entry, ok, reason := Translate(s, t.Style)
		if !ok {
			notes = append(notes, fmt.Sprintf("[%s] mcp: n/a %s — %s", t.Harness, name, reason))
			continue
		}
		entries[name] = entry
	}
	return entries, notes
}

// ProjectBytes computes what Project would write into t without writing it:
// the target path, the merged bytes, and how many servers went in.
//
// out is nil when there is nothing to write: no server reaches this dialect, or
// (LenientJSON) the target is not plain JSON and must be left untouched.
func ProjectBytes(t Target, userHome string, f *File) (path string, out []byte, entries int, err error) {
	path = t.Path(userHome)
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return path, nil, 0, err
	}
	translated, _ := targetEntries(t, f)
	if len(translated) == 0 {
		return path, nil, 0, nil
	}
	out, err = mergeEntries(t, path, translated, existing)
	if err != nil {
		return path, nil, len(translated), err
	}
	return path, out, len(translated), nil
}

// MergeProjection applies t's merge rules to existing — the content the target
// holds when this layer runs — and returns the bytes Project would write.
//
// It is how plan stacks layers: each projection merges into the previous layer's
// output instead of re-reading the file. out is nil when there is nothing to
// write (see ProjectBytes).
func MergeProjection(t Target, path string, f *File, existing []byte) (out []byte, err error) {
	translated, _ := targetEntries(t, f)
	if len(translated) == 0 {
		return nil, nil
	}
	return mergeEntries(t, path, translated, existing)
}

// mergeEntries merges translated servers into existing content. out is nil when
// the target must be left untouched: a LenientJSON file that is not plain JSON.
func mergeEntries(t Target, path string, entries map[string]map[string]any, existing []byte) ([]byte, error) {
	switch t.Format {
	case FormatTOML:
		return []byte(upsertTOMLSections(string(existing), t.TopKey, entries)), nil
	case FormatYAML:
		out, err := upsertYAMLSection(string(existing), t.TopKey, entries)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		return out, nil
	}
	if t.LenientJSON && !jsonParseable(existing) {
		return nil, nil
	}
	return mergeJSONConfig(path, t.TopKey, entries, existing)
}

func styleSupportsEnabled(style string) bool {
	switch style {
	case StyleCodex, StyleOpenCode, StylePi, StyleKiro, StyleZCode,
		StyleOMP, StyleHermes, StyleDimAgent:
		return true
	}
	return false
}

// jsonParseable reports whether content parses as plain JSON or JSONC. Missing
// or empty content counts as parseable (a fresh file is written).
func jsonParseable(content []byte) bool {
	if len(strings.TrimSpace(string(content))) == 0 {
		return true
	}
	var root map[string]any
	return json.Unmarshal(secrets.StripJSONC(content), &root) == nil
}

// mergeJSONConfig sets root[topKey][name]=entry, preserving all other content.
// topKey may be a dotted path (e.g. "mcp.servers") for nested configs.
func mergeJSONConfig(path, topKey string, entries map[string]map[string]any, existing []byte) ([]byte, error) {
	root := map[string]any{}
	if len(strings.TrimSpace(string(existing))) > 0 {
		if err := json.Unmarshal(secrets.StripJSONC(existing), &root); err != nil {
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

// recordProjection adds the target to the backup set's manifest. An existing
// target is copied into the set first; a missing one is recorded as create so
// rollback deletes it. out is the content about to be written: its fingerprint
// lets rollback tell this projection's output from a later user edit.
func recordProjection(t Target, userHome, mirrorRoot, path string, out []byte, backupDir string) error {
	if backupDir == "" {
		return nil
	}
	rel := projectionRel(t, userHome, mirrorRoot, path)
	action, backupRel := backup.ActionCreate, ""
	if fileExists(path) {
		action, backupRel = backup.ActionUpdate, filepath.Join(t.Harness, rel)
	}
	return backup.Record(backupDir, backup.Entry{
		Adapter: t.Harness,
		Rel:     rel,
		Target:  path,
		Backup:  backupRel,
		Action:  action,
		Mode:    backup.ModeString(0o600),
		After:   backup.FingerprintOf(out),
	})
}

// projectionRel is where a projection target is recorded inside a backup set:
// relative to the harness's mirror root, falling back to the user home.
func projectionRel(t Target, userHome, mirrorRoot, path string) string {
	// ~/.claude.json lives outside Claude's mirror root, so it is addressed by
	// the name apply uses for the same file.
	if t.Harness == "claude" && filepath.Base(path) == ".claude.json" {
		return backup.ClaudeJSONRel
	}
	if mirrorRoot != "" {
		if rel, err := filepath.Rel(mirrorRoot, path); err == nil && !escapesUp(rel) {
			return rel
		}
	}
	if rel, err := filepath.Rel(userHome, path); err == nil {
		return rel
	}
	return filepath.Base(path)
}

func escapesUp(rel string) bool {
	return rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
