package core

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// applyClaudePartial merges the per-harness Claude custom partial
// (packs/harness/claude/claude.json.partial.json) into ~/.claude.json.
// The library copy is the final content (no placeholder reinjection).
func applyClaudePartial(opts ApplyOptions, backupDir string) ([]string, error) {
	var log []string
	src := filepath.Join(opts.MagicHome, "packs", "harness", "claude", "claude.json.partial.json")
	data, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			return log, nil
		}
		return log, err
	}
	var partial map[string]any
	if err := json.Unmarshal(data, &partial); err != nil {
		return log, err
	}
	dst := filepath.Join(opts.UserHome, ".claude.json")
	existing := map[string]any{}
	if raw, err := os.ReadFile(dst); err == nil {
		_ = json.Unmarshal(raw, &existing)
	}
	for k, v := range partial {
		existing[k] = v
	}
	out, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return log, err
	}
	out = append(out, '\n')
	if opts.DryRun {
		log = append(log, "[claude] would merge claude.json.partial.json → ~/.claude.json")
		return log, nil
	}
	if _, err := os.Stat(dst); err == nil && backupDir != "" {
		_ = BackupPath(dst, backupDir, "claude/_claude.json")
	}
	if err := AtomicWriteFile(dst, out, 0o600); err != nil {
		return log, err
	}
	log = append(log, "[claude] merged claude.json.partial.json → ~/.claude.json")
	return log, nil
}
