package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"magic/internal/adapter"
	"magic/internal/secrets"
	"magic/internal/writefile"
)

// mergeLibraryFile writes library bytes to dst following decisions.md D5:
// structured config files merge by key; everything else (AGENTS.md-style whole
// files, one-file-per-item assets) is written as-is. User keys/items that the
// library does not define are never deleted.
//
// The target is recorded in the backup set only when a write is needed, and
// before the write, so an update entry holds the pre-apply content. A target
// whose content and permission already match is left untouched.
func mergeLibraryFile(backupDir string, id adapter.ID, dst, rel string, srcData []byte, mode os.FileMode) error {
	existing, readErr := os.ReadFile(dst)
	out, _, err := projectBytes(rel, srcData, existing, readErr == nil)
	if err != nil {
		return err
	}
	if !writefile.NeedsWrite(dst, out, mode) {
		return nil
	}
	if err := recordWrite(backupDir, id, rel, dst, mode, out); err != nil {
		return err
	}
	return writefile.Write(dst, out, mode)
}

// ProjectStrategy is the rule projectBytes actually used. A plan reports the
// reason from it, so the reason cannot claim a merge that did not happen.
type ProjectStrategy string

const (
	// StrategyCreate: the target does not exist, so the library bytes are written.
	StrategyCreate ProjectStrategy = "create"
	// StrategyJSONMerge: structured JSON deep merge, library value wins.
	StrategyJSONMerge ProjectStrategy = "json-merge"
	// StrategyJSONFallback: the file is named like a structured JSON config but a
	// side is not a parseable JSON object, so the whole file is replaced.
	StrategyJSONFallback ProjectStrategy = "json-fallback"
	// StrategyTOMLMerge: section-aware config.toml merge.
	StrategyTOMLMerge ProjectStrategy = "toml-merge"
	// StrategyWholeFile: the library bytes replace the target as-is.
	StrategyWholeFile ProjectStrategy = "whole-file"
)

// projectBytes returns the bytes that apply writes to dst and the strategy it
// used, following decisions.md D5. apply and plan both call it, so a plan cannot
// describe a write that apply would not make.
func projectBytes(rel string, srcData, dstData []byte, hasExisting bool) ([]byte, ProjectStrategy, error) {
	if !hasExisting {
		return srcData, StrategyCreate, nil
	}
	base := filepath.Base(rel)
	switch {
	case isMergeableJSON(base):
		merged, ok, err := mergeJSONBytes(srcData, dstData)
		if err != nil {
			return nil, "", err
		}
		if ok {
			return merged, StrategyJSONMerge, nil
		}
		return srcData, StrategyJSONFallback, nil
	case base == "config.toml":
		return []byte(mergeTOML(string(dstData), string(srcData))), StrategyTOMLMerge, nil
	}
	return srcData, StrategyWholeFile, nil
}

// isMergeableJSON reports whether a mirrored file is a structured config that
// must be merged by key rather than overwritten.
func isMergeableJSON(base string) bool {
	switch base {
	case "settings.json", "opencode.json", "opencode.jsonc", "hooks.json", "mcp.json", "mcp-config.json":
		return true
	}
	return strings.HasSuffix(base, ".partial.json")
}

// mergeJSONBytes deep-merges the library JSON object over the existing bytes.
// Returns ok=false when either side is not a JSON object.
func mergeJSONBytes(srcData, dstData []byte) ([]byte, bool, error) {
	var dstObj, srcObj map[string]any
	if json.Unmarshal(secrets.StripJSONC(dstData), &dstObj) != nil {
		return nil, false, nil
	}
	if json.Unmarshal(secrets.StripJSONC(srcData), &srcObj) != nil {
		return nil, false, nil
	}
	merged := deepMergeMap(dstObj, srcObj)
	out, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return nil, false, err
	}
	return append(out, '\n'), true, nil
}

// deepMergeMap returns dst ∪ src with src winning on conflicts. Maps recurse;
// scalars and arrays are replaced by the library value.
func deepMergeMap(dst, src map[string]any) map[string]any {
	out := make(map[string]any, len(dst)+len(src))
	for k, v := range dst {
		out[k] = v
	}
	for k, sv := range src {
		if dv, ok := out[k]; ok {
			dm, dok := dv.(map[string]any)
			sm, sok := sv.(map[string]any)
			if dok && sok {
				out[k] = deepMergeMap(dm, sm)
				continue
			}
		}
		out[k] = sv
	}
	return out
}

// --- minimal section-aware TOML merge (top-level keys + [section] upsert) ---
//
// This is intentionally conservative: simple `key = value` pairs in matching
// sections are upserted (library wins), user keys not in the library are kept,
// and array-of-tables `[[name]]` from the library replace the destination's
// blocks of that name wholesale. It is not a full TOML implementation.

type tomlBlock struct {
	header  string
	isArray bool
	lines   []string
}

func mergeTOML(dst, src string) string {
	if strings.TrimSpace(dst) == "" {
		return src
	}
	dstBlocks := parseTOML(dst)
	srcBlocks := parseTOML(src)

	for _, sb := range srcBlocks {
		if sb.isArray && sb.header != "" {
			dstBlocks = removeTOMLBlocks(dstBlocks, sb.header, true)
		}
	}
	for _, sb := range srcBlocks {
		if sb.isArray && sb.header != "" {
			// Array-of-tables: destination blocks were removed above; append every
			// library block so repeated entries are preserved.
			dstBlocks = append(dstBlocks, sb)
			continue
		}
		idx := findTOMLBlock(dstBlocks, sb)
		if idx < 0 {
			dstBlocks = append(dstBlocks, sb)
			continue
		}
		dstBlocks[idx] = mergeTOMLBlock(dstBlocks[idx], sb)
	}
	return renderTOML(dstBlocks)
}

func parseTOML(text string) []tomlBlock {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	var blocks []tomlBlock
	cur := tomlBlock{}
	for _, raw := range lines {
		trim := strings.TrimSpace(raw)
		if h, isArray, ok := tomlHeader(trim); ok {
			blocks = append(blocks, cur)
			cur = tomlBlock{header: h, isArray: isArray}
			continue
		}
		cur.lines = append(cur.lines, raw)
	}
	blocks = append(blocks, cur)
	return blocks
}

func tomlHeader(trim string) (header string, isArray bool, ok bool) {
	switch {
	case strings.HasPrefix(trim, "[[") && strings.HasSuffix(trim, "]]"):
		return strings.TrimSpace(trim[2 : len(trim)-2]), true, true
	case strings.HasPrefix(trim, "[") && strings.HasSuffix(trim, "]"):
		return strings.TrimSpace(trim[1 : len(trim)-1]), false, true
	}
	return "", false, false
}

func removeTOMLBlocks(blocks []tomlBlock, header string, array bool) []tomlBlock {
	var out []tomlBlock
	for _, b := range blocks {
		if b.header == header && b.isArray == array {
			continue
		}
		out = append(out, b)
	}
	return out
}

func findTOMLBlock(blocks []tomlBlock, want tomlBlock) int {
	for i, b := range blocks {
		if b.header == want.header && b.isArray == want.isArray {
			return i
		}
	}
	return -1
}

// mergeTOMLBlock upserts src's `key = value` lines into dst, keeping dst-only keys.
func mergeTOMLBlock(dst, src tomlBlock) tomlBlock {
	srcByKey := map[string]string{}
	var srcKeys []string
	for _, line := range src.lines {
		k, ok := lineKeyMinimal(line)
		if !ok {
			continue
		}
		if _, seen := srcByKey[k]; !seen {
			srcKeys = append(srcKeys, k)
		}
		srcByKey[k] = line
	}
	seen := map[string]bool{}
	var body []string
	for _, line := range dst.lines {
		k, ok := lineKeyMinimal(line)
		if !ok {
			body = append(body, line)
			continue
		}
		if repl, ok := srcByKey[k]; ok {
			body = append(body, repl)
			seen[k] = true
			continue
		}
		body = append(body, line)
	}
	for _, k := range srcKeys {
		if !seen[k] {
			body = append(body, srcByKey[k])
		}
	}
	return tomlBlock{header: dst.header, isArray: dst.isArray, lines: body}
}

// lineKeyMinimal returns the assignment key for a simple `key = value` line.
func lineKeyMinimal(line string) (string, bool) {
	trim := strings.TrimSpace(line)
	if trim == "" || strings.HasPrefix(trim, "#") || !strings.Contains(trim, "=") {
		return "", false
	}
	key := strings.TrimSpace(strings.SplitN(trim, "=", 2)[0])
	if key == "" {
		return "", false
	}
	return key, true
}

func renderTOML(blocks []tomlBlock) string {
	var b strings.Builder
	for _, blk := range blocks {
		if blk.header != "" || blk.isArray {
			if blk.isArray {
				b.WriteString("[[" + blk.header + "]]\n")
			} else {
				b.WriteString("[" + blk.header + "]\n")
			}
		}
		for _, line := range blk.lines {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	out := b.String()
	// Collapse an accidental leading blank from an empty preamble block.
	out = strings.TrimPrefix(out, "\n")
	if out != "" && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out
}
