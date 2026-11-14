package provider

import (
	"fmt"
	"strings"
)

// TOML helpers for the dialect writers that manage whole tables (Kimi Code's
// providers/models, Vibe's [[providers]]/[[models]]). They work on lines, like
// Grok's managed-block editor, because the files belong to the user: a full
// parse-and-re-emit would drop their comments and key order.
//
// Scope: Magic only ever adds or removes *whole tables* it owns, plus one root
// scalar it points at them. Nothing here edits a table Magic does not own.

// tomlTablePath returns the dotted table path a header line opens, with quotes
// removed from its segments: `[providers."magic-x"]` → providers.magic-x,
// `[[models]]` → models, `[providers."magic-x".env]` → providers.magic-x.env.
func tomlTablePath(line string) (string, bool) {
	trim := strings.TrimSpace(line)
	if !strings.HasPrefix(trim, "[") {
		return "", false
	}
	trim = strings.TrimPrefix(trim, "[")
	trim = strings.TrimPrefix(trim, "[")
	end := strings.LastIndex(trim, "]")
	if end < 0 {
		return "", false
	}
	inner := strings.TrimSpace(trim[:end])
	inner = strings.ReplaceAll(inner, `"`, "")
	inner = strings.ReplaceAll(inner, `'`, "")
	if inner == "" {
		return "", false
	}
	return inner, true
}

// tomlTableKey quotes a table path segment the way the dialect examples do.
func tomlTableKey(key string) string {
	return `"` + key + `"`
}

// tomlDropTables removes every table whose path satisfies drop, together with
// the lines under it, and leaves everything else untouched.
func tomlDropTables(lines []string, drop func(path string) bool) []string {
	out := make([]string, 0, len(lines))
	skipping := false
	for _, line := range lines {
		if path, ok := tomlTablePath(line); ok {
			skipping = drop(path)
			if skipping {
				continue
			}
		}
		if skipping {
			continue
		}
		out = append(out, line)
	}
	return out
}

// tomlDropCommentBlock removes the lines between two comment markers, inclusive.
// Magic writes that block; an unterminated one reads as "Magic's block to the
// end of the file", which is what an interrupted write leaves behind.
func tomlDropCommentBlock(lines []string, startMark, endMark string) []string {
	out := make([]string, 0, len(lines))
	skipping := false
	for _, line := range lines {
		switch strings.TrimSpace(line) {
		case startMark:
			skipping = true
			continue
		case endMark:
			if skipping {
				skipping = false
				continue
			}
		}
		if skipping {
			continue
		}
		out = append(out, line)
	}
	return out
}

// tomlHasTableMatching reports whether any table header satisfies match.
func tomlHasTableMatching(lines []string, match func(path string) bool) bool {
	for _, line := range lines {
		if path, ok := tomlTablePath(line); ok && match(path) {
			return true
		}
	}
	return false
}

// tomlAppendBlock appends table lines at the end of the document, keeping one
// blank line between the user's last table and Magic's block.
func tomlAppendBlock(lines []string, block []string) []string {
	out := append([]string{}, lines...)
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}
	if len(out) > 0 {
		out = append(out, "")
	}
	return append(out, block...)
}

// tomlRootRegionEnd is the index where the first table header starts; root
// scalars can only live before it (a key after a header belongs to that table).
func tomlRootRegionEnd(lines []string) int {
	for i, line := range lines {
		if _, ok := tomlTablePath(line); ok {
			return i
		}
	}
	return len(lines)
}

// tomlRootValue reads a root-level string scalar.
func tomlRootValue(lines []string, key string) (string, bool) {
	end := tomlRootRegionEnd(lines)
	for i := 0; i < end; i++ {
		if v, ok := parseQuotedAssign(lines[i], key); ok {
			return v, true
		}
	}
	return "", false
}

// tomlSetRootValue writes key = "value" into the root region, replacing the
// existing line in place or inserting one before the first table header.
func tomlSetRootValue(lines []string, key, value string) []string {
	settingLine := fmt.Sprintf("%s = %q", key, value)
	end := tomlRootRegionEnd(lines)
	for i := 0; i < end; i++ {
		if _, ok := parseQuotedAssign(lines[i], key); ok {
			out := append([]string{}, lines...)
			out[i] = settingLine
			return out
		}
	}
	out := make([]string, 0, len(lines)+2)
	out = append(out, lines[:end]...)
	out = append(out, settingLine)
	if end < len(lines) {
		out = append(out, "")
		out = append(out, lines[end:]...)
	}
	return out
}

// tomlDropRootValue removes key from the root region.
func tomlDropRootValue(lines []string, key string) []string {
	end := tomlRootRegionEnd(lines)
	out := make([]string, 0, len(lines))
	for i, line := range lines {
		if i < end {
			if k, ok := lineKey(line); ok && k == key {
				continue
			}
		}
		out = append(out, line)
	}
	// A key that was alone in the root region leaves the file starting with a
	// blank line; drop leading blanks so a cleared file looks hand-written.
	for len(out) > 0 && strings.TrimSpace(out[0]) == "" {
		out = out[1:]
	}
	return out
}
