package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"magic/internal/adapter"
	"magic/internal/policy"
)

// Keys whose string values are treated as commands or paths when walking settings JSON.
var settingsRefKeys = map[string]bool{
	"command": true,
	"cmd":     true,
	"script":  true,
	"path":    true,
}

// pathTokenRe finds path-like tokens: ~/..., $HOME/..., or absolute paths (optionally quoted).
var pathTokenRe = regexp.MustCompile(`(?:"([^"]+)"|'([^']+)'|((?:~|\$HOME|\$\{HOME\})?/[^\s"';]+))`)

// tomlQuotedStringRe extracts double- or single-quoted string literals from TOML-ish text.
var tomlQuotedStringRe = regexp.MustCompile(`"([^"\\]|\\.)*"|'([^'\\]|\\.)*'`)

// ExtractLocalRefs walks settings JSON and returns absolute paths under harnessRoot
// referenced by command/cmd/script/path values (and any string that embeds a path
// under the harness root). Existence is not checked here.
// harnessRoot is the absolute harness directory (e.g. /Users/x/.claude).
func ExtractLocalRefs(settingsJSON []byte, userHome, harnessRoot string) []string {
	var root any
	if err := json.Unmarshal(settingsJSON, &root); err != nil {
		return nil
	}
	userHome = filepath.Clean(userHome)
	harnessRoot = filepath.Clean(harnessRoot)
	seen := map[string]bool{}
	var out []string

	add := func(abs string) {
		abs = filepath.Clean(abs)
		if abs == "" || abs == harnessRoot {
			return
		}
		if !pathUnderRoot(abs, harnessRoot) {
			return
		}
		if seen[abs] {
			return
		}
		seen[abs] = true
		out = append(out, abs)
	}

	var walk func(v any, key string)
	walk = func(v any, key string) {
		switch t := v.(type) {
		case map[string]any:
			for k, child := range t {
				walk(child, k)
			}
		case []any:
			for _, child := range t {
				walk(child, key)
			}
		case string:
			for _, p := range pathsFromSettingsString(t, key, userHome, harnessRoot) {
				add(p)
			}
		}
	}
	walk(root, "")
	return out
}

// ExtractConfigLocalRefs extracts path-like quoted strings from config text
// (e.g. Codex config.toml) that resolve under harnessRoot. Conservative:
// only paths under the harness root; ignores URLs and bare binaries.
// Existence is not checked here.
func ExtractConfigLocalRefs(configText []byte, userHome, harnessRoot string) []string {
	userHome = filepath.Clean(userHome)
	harnessRoot = filepath.Clean(harnessRoot)
	seen := map[string]bool{}
	var out []string

	add := func(abs string) {
		abs = filepath.Clean(abs)
		if abs == "" || abs == harnessRoot {
			return
		}
		if !pathUnderRoot(abs, harnessRoot) {
			return
		}
		if seen[abs] {
			return
		}
		seen[abs] = true
		out = append(out, abs)
	}

	text := string(configText)
	for _, m := range tomlQuotedStringRe.FindAllString(text, -1) {
		tok := unquote(m)
		tok = unescapeSimple(tok)
		tok = strings.TrimSpace(tok)
		if tok == "" || looksLikeURL(tok) {
			continue
		}
		if !looksLikePathToken(tok) {
			continue
		}
		// Skip bare binaries (no path separator / home prefix).
		if !strings.ContainsAny(tok, `/\`) && !strings.HasPrefix(tok, "~") && !strings.HasPrefix(tok, "$") {
			continue
		}
		if exp := expandUserPath(tok, userHome); exp != "" {
			add(exp)
		}
	}

	// Also pick up unquoted ~/… / $HOME/… tokens that may appear in comments or bare values.
	for _, m := range pathTokenRe.FindAllStringSubmatch(text, -1) {
		tok := m[1]
		if tok == "" {
			tok = m[2]
		}
		if tok == "" {
			tok = m[3]
		}
		tok = strings.TrimSpace(tok)
		if tok == "" || looksLikeURL(tok) {
			continue
		}
		if !strings.ContainsAny(tok, `/\`) && !strings.HasPrefix(tok, "~") && !strings.HasPrefix(tok, "$") {
			continue
		}
		if exp := expandUserPath(tok, userHome); exp != "" {
			add(exp)
		}
	}
	return out
}

func unescapeSimple(s string) string {
	// Minimal TOML/JSON-ish unescapes for path strings.
	replacer := strings.NewReplacer(
		`\\`, `\`,
		`\"`, `"`,
		`\'`, `'`,
	)
	return replacer.Replace(s)
}

func pathsFromSettingsString(s, key, userHome, harnessRoot string) []string {
	s = strings.TrimSpace(s)
	if s == "" || looksLikeURL(s) {
		return nil
	}
	interested := settingsRefKeys[strings.ToLower(key)] || stringMentionsHarness(s, userHome, harnessRoot)
	if !interested {
		return nil
	}

	var raw []string

	// Whole-string path (common for "path" / "script").
	if u := unquote(s); looksLikePathToken(u) {
		if expanded := expandUserPath(u, userHome); expanded != "" {
			raw = append(raw, expanded)
		}
	}

	// Tokenize shell-ish commands: bash ~/.claude/foo.sh, "quoted paths", etc.
	for _, m := range pathTokenRe.FindAllStringSubmatch(s, -1) {
		tok := m[1]
		if tok == "" {
			tok = m[2]
		}
		if tok == "" {
			tok = m[3]
		}
		tok = strings.TrimSpace(tok)
		if tok == "" || looksLikeURL(tok) {
			continue
		}
		// Skip bare binaries (no path separator / home prefix).
		if !strings.ContainsAny(tok, `/\`) && !strings.HasPrefix(tok, "~") && !strings.HasPrefix(tok, "$") {
			continue
		}
		if exp := expandUserPath(tok, userHome); exp != "" {
			raw = append(raw, exp)
		}
	}

	var out []string
	for _, p := range raw {
		p = filepath.Clean(p)
		if pathUnderRoot(p, harnessRoot) {
			out = append(out, p)
		}
	}
	return out
}

func stringMentionsHarness(s, userHome, harnessRoot string) bool {
	slashRoot := filepath.ToSlash(harnessRoot)
	sSlash := filepath.ToSlash(s)
	if strings.Contains(sSlash, slashRoot) {
		return true
	}
	rel, err := filepath.Rel(userHome, harnessRoot)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	return strings.Contains(sSlash, "~/"+rel) ||
		strings.Contains(sSlash, "$HOME/"+rel) ||
		strings.Contains(sSlash, "${HOME}/"+rel)
}

func looksLikePathToken(s string) bool {
	if strings.HasPrefix(s, "~/") || strings.HasPrefix(s, "$HOME/") || strings.HasPrefix(s, "${HOME}/") {
		return true
	}
	if filepath.IsAbs(s) {
		return true
	}
	return strings.ContainsAny(s, `/\`)
}

func looksLikeURL(s string) bool {
	ls := strings.ToLower(s)
	return strings.HasPrefix(ls, "http://") || strings.HasPrefix(ls, "https://") ||
		strings.HasPrefix(ls, "ftp://") || strings.HasPrefix(ls, "file://")
}

func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func expandUserPath(p, userHome string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	switch {
	case p == "~":
		return userHome
	case strings.HasPrefix(p, "~/"):
		return filepath.Join(userHome, filepath.FromSlash(p[2:]))
	case strings.HasPrefix(p, "$HOME/"):
		return filepath.Join(userHome, filepath.FromSlash(p[len("$HOME/"):]))
	case strings.HasPrefix(p, "${HOME}/"):
		return filepath.Join(userHome, filepath.FromSlash(p[len("${HOME}/"):]))
	default:
		if filepath.IsAbs(p) {
			return filepath.Clean(p)
		}
		// Relative tokens are not resolved against cwd.
		return ""
	}
}

func pathUnderRoot(abs, root string) bool {
	abs = filepath.Clean(abs)
	root = filepath.Clean(root)
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return false
	}
	if rel == "." {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func isSettingsConfigRel(rel string) bool {
	base := filepath.Base(rel)
	return base == "settings.json" || base == "settings.local.json"
}

func isTomlConfigRel(rel string) bool {
	return filepath.Base(rel) == "config.toml"
}

// followLocalConfigRefs appends allow verdicts for local files referenced by
// allow-listed settings.json / settings.local.json (reason settings.ref) and
// config.toml (reason config.ref). Missing refs are skipped.
func followLocalConfigRefs(report *policy.Report, userHome string, a adapter.Adapter, overlay *policy.OverlayHarness) {
	type candidate struct {
		abs, root, harness, rel, reason string
		extract                         func([]byte, string, string) []string
	}
	var cands []candidate
	for _, v := range report.Allow {
		switch {
		case isSettingsConfigRel(v.Rel):
			cands = append(cands, candidate{
				abs: v.AbsPath, root: v.Root, harness: v.Harness, rel: v.Rel,
				reason: "settings.ref", extract: ExtractLocalRefs,
			})
		case isTomlConfigRel(v.Rel):
			cands = append(cands, candidate{
				abs: v.AbsPath, root: v.Root, harness: v.Harness, rel: v.Rel,
				reason: "config.ref", extract: ExtractConfigLocalRefs,
			})
		}
	}
	if len(cands) == 0 {
		return
	}

	inAllow := map[string]bool{}
	for _, v := range report.Allow {
		inAllow[filepath.Clean(v.AbsPath)] = true
	}

	for _, c := range cands {
		if c.abs == "" || c.root == "" {
			continue
		}
		harness := c.harness
		if harness == "" {
			harness = string(a.ID())
		}
		data, err := os.ReadFile(c.abs)
		if err != nil {
			continue
		}
		refs := c.extract(data, userHome, c.root)
		for _, abs := range refs {
			abs = filepath.Clean(abs)
			if inAllow[abs] {
				continue
			}
			st, err := os.Stat(abs)
			if err != nil || st.IsDir() {
				continue
			}
			rel, err := filepath.Rel(c.root, abs)
			if err != nil {
				continue
			}
			rel = filepath.ToSlash(rel)
			if overlay != nil && policy.MatchAny(rel, overlay.Exclude) {
				continue
			}
			report.Unknown = removeVerdictByAbs(report.Unknown, abs)
			report.Deny = removeVerdictByAbs(report.Deny, abs)

			report.Allow = append(report.Allow, policy.Verdict{
				Harness:  harness,
				AbsPath:  abs,
				Rel:      rel,
				Decision: "allow",
				Reason:   c.reason,
				Root:     c.root,
			})
			inAllow[abs] = true
		}
	}
}

// followSettingsRefs is kept as an alias name used historically; delegates to followLocalConfigRefs.
func followSettingsRefs(report *policy.Report, userHome string, a adapter.Adapter, overlay *policy.OverlayHarness) {
	followLocalConfigRefs(report, userHome, a, overlay)
}

func removeVerdictByAbs(vs []policy.Verdict, abs string) []policy.Verdict {
	abs = filepath.Clean(abs)
	var out []policy.Verdict
	for _, v := range vs {
		if filepath.Clean(v.AbsPath) != abs {
			out = append(out, v)
		}
	}
	return out
}

// FoldHomePathsInSettings rewrites absolute / $HOME prefixes under
// userHome/<harnessRel> to ~/… inside JSON string values so the library copy
// is portable across machines. harnessRel is the adapter RootSpec.RelHome
// (e.g. ".claude", ".pi/agent"). Invalid JSON is returned unchanged.
func FoldHomePathsInSettings(settingsJSON []byte, userHome, harnessRel string) []byte {
	var root any
	if err := json.Unmarshal(settingsJSON, &root); err != nil {
		return settingsJSON
	}
	userHome = filepath.Clean(userHome)
	harnessRel = filepath.ToSlash(strings.Trim(filepath.ToSlash(harnessRel), "/"))
	if harnessRel == "" {
		return settingsJSON
	}
	harnessRoot := filepath.Join(userHome, filepath.FromSlash(harnessRel))
	tildePrefix := "~/" + harnessRel
	changed := false

	var walk func(v any) any
	walk = func(v any) any {
		switch t := v.(type) {
		case map[string]any:
			for k, child := range t {
				t[k] = walk(child)
			}
			return t
		case []any:
			for i, child := range t {
				t[i] = walk(child)
			}
			return t
		case string:
			folded, did := foldHarnessPathsInString(t, harnessRoot, harnessRel, tildePrefix)
			if did {
				changed = true
			}
			return folded
		default:
			return v
		}
	}
	root = walk(root)
	if !changed {
		return settingsJSON
	}
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return settingsJSON
	}
	return append(out, '\n')
}

func foldHarnessPathsInString(s, harnessRoot, harnessRel, tildePrefix string) (string, bool) {
	if s == "" {
		return s, false
	}
	orig := s
	replacements := []struct{ from, to string }{
		{filepath.ToSlash(harnessRoot), tildePrefix},
		{harnessRoot, tildePrefix},
		{"$HOME/" + harnessRel, tildePrefix},
		{"${HOME}/" + harnessRel, tildePrefix},
	}
	for _, r := range replacements {
		if r.from != "" && strings.Contains(s, r.from) {
			s = strings.ReplaceAll(s, r.from, r.to)
		}
	}
	if s == orig {
		return s, false
	}
	return s, true
}

// PrimaryRelHome returns the first CapturePolicy root RelHome, or "".
func PrimaryRelHome(a adapter.Adapter) string {
	roots := a.CapturePolicy().Roots
	if len(roots) == 0 {
		return ""
	}
	return roots[0].RelHome
}
