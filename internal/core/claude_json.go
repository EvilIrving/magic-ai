package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"magic/internal/backup"
	"magic/internal/writefile"
)

// claudePartialFile is the per-harness Claude custom partial relative to MAGIC_HOME.
var claudePartialFile = filepath.Join("packs", "harness", "claude", "claude.json.partial.json")

// applyClaudePartial merges the per-harness Claude custom partial
// (packs/harness/claude/claude.json.partial.json) into ~/.claude.json.
// The library copy is the final content (no placeholder reinjection).
func applyClaudePartial(opts ApplyOptions, backupDir string) ([]string, error) {
	var log []string
	src := filepath.Join(opts.MagicHome, claudePartialFile)
	partialData, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			return log, nil
		}
		return log, err
	}
	dst := filepath.Join(opts.UserHome, ".claude.json")
	dstData, readErr := os.ReadFile(dst)
	hasExisting := readErr == nil
	out, err := claudePartialProjected(partialData, dstData, hasExisting)
	if err != nil {
		return log, err
	}
	if opts.DryRun {
		log = append(log, "[claude] would merge claude.json.partial.json → ~/.claude.json")
		return log, nil
	}
	const mode = os.FileMode(0o600)
	if writefile.NeedsWrite(dst, out, mode) {
		if err := recordWrite(backupDir, "claude", backup.ClaudeJSONRel, dst, mode, out); err != nil {
			return log, err
		}
		if err := writefile.Write(dst, out, mode); err != nil {
			return log, err
		}
	}
	log = append(log, "[claude] merged claude.json.partial.json → ~/.claude.json")
	return log, nil
}

// claudePartialProjected returns the bytes apply writes to ~/.claude.json.
// applyClaudePartial and plan both call it (architecture.md I5), so the plan
// cannot predict a different merge than apply performs.
//
// The merge is field-level (decisions.md D5): nested objects such as mcpServers
// keep the user's entries, and a library value only wins on the same key path.
func claudePartialProjected(partialData, dstData []byte, hasExisting bool) ([]byte, error) {
	var partial map[string]any
	if err := json.Unmarshal(partialData, &partial); err != nil {
		return nil, fmt.Errorf("parse %s: %w", claudePartialFile, err)
	}
	existing := map[string]any{}
	if hasExisting {
		if err := json.Unmarshal(dstData, &existing); err != nil {
			// Never drop a user's ~/.claude.json because it stopped parsing.
			return nil, fmt.Errorf("parse ~/.claude.json: %w", err)
		}
	}
	out, err := json.MarshalIndent(deepMergeMap(existing, partial), "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}
