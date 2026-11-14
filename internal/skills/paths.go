package skills

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"magic/internal/appdir"
	"magic/internal/registry"
)

// DefaultAgents are the agent skill sync targets used when none are given.
// These are the harnesses that also have a Magic product adapter (plus cursor,
// which is skills-only). The full vercel-labs/skills long tail is reachable
// with `--target all` (see AllAgents).
//
// Source: internal | docs/supported-harnesses.md → 用量驱动的 v1 覆盖范围 (D20/D21)
//
//	checked 2026-09-13 | verified
//
// Order follows the OpenRouter app usage ranking, not alphabetical and not import
// order — `magic skills status` output reads as a priority list.
var DefaultAgents = []string{
	"codex", "claude", "cursor", "pi", "grok", "antigravity-cli", "opencode",
	"kimi-code", "copilot", "workbuddy", "trae", "qoder", "qoder-cn", "cline",
	"omp", "hermes", "mimocode", "openclaw",
}

// globalSkills maps an agent id to its global skills dir, relative to $HOME.
//
// Source: upstream | github.com/vercel-labs/skills → README "Supported Agents"
//
//	(https://raw.githubusercontent.com/vercel-labs/skills/main/README.md)
//	snapshot 2026-09-11 | checked 2026-09-13 | verified
//
// Adopted verbatim as fixed config (decision D11): this is someone else's path
// table, so do not re-derive it from "what looks conventional".
//   - `windsurf` / `roo` are omitted on purpose (D13, both discontinued) —
//     Never: re-add them because the upstream list still has them.
//   - `eve` / `promptscript` are project-only upstream, hence absent here.
//   - Several ids share one directory (`.agents/skills`, `.config/agents/skills`);
//     that is the upstream layout, not a copy-paste bug.
//
// When updating: re-read the snapshot above, then bump the snapshot date. Magic
// adapters whose path was measured locally (see internal/adapters/*) must agree
// with this table; if they disagree, the adapter comment carries the evidence.
var globalSkills = map[string]string{
	"aider-desk":      ".aider-desk/skills",
	"amp":             ".config/agents/skills",
	"replit":          ".config/agents/skills",
	"universal":       ".config/agents/skills",
	"antigravity":     ".gemini/antigravity/skills",
	"antigravity-cli": ".gemini/antigravity-cli/skills",
	"astrbot":         ".astrbot/data/skills",
	"autohand-code":   ".autohand/skills",
	"augment":         ".augment/skills",
	"bob":             ".bob/skills",
	"claude":          ".claude/skills",
	"claude-code":     ".claude/skills",
	"openclaw":        ".openclaw/skills",
	"omp":             ".omp/agent/skills",
	"oh-my-pi":        ".omp/agent/skills",
	"cline":           ".agents/skills",
	"dexto":           ".agents/skills",
	"kimi-code":       ".agents/skills",
	"kimi-code-cli":   ".agents/skills",
	"loaf":            ".agents/skills",
	"sarvam-code":     ".agents/skills",
	"warp":            ".agents/skills",
	"zed":             ".agents/skills",
	"codearts-agent":  ".codeartsdoer/skills",
	"codebuddy":       ".codebuddy/skills",
	"workbuddy":       ".codebuddy/skills",
	"codemaker":       ".codemaker/skills",
	"codestudio":      ".codestudio/skills",
	"codex":           ".codex/skills",
	"command-code":    ".commandcode/skills",
	"continue":        ".continue/skills",
	"cortex":          ".snowflake/cortex/skills",
	"crush":           ".config/crush/skills",
	"cursor":          ".cursor/skills",
	"deepagents":      ".deepagents/agent/skills",
	"devin":           ".config/devin/skills",
	"droid":           ".factory/skills",
	"firebender":      ".firebender/skills",
	"forgecode":       ".forge/skills",
	"fx":              ".fx/skills",
	"gemini-cli":      ".gemini/skills",
	"github-copilot":  ".copilot/skills",
	"copilot":         ".copilot/skills",
	"goose":           ".config/goose/skills",
	"grok":            ".grok/skills",
	"hermes-agent":    ".hermes/skills",
	"inference-sh":    ".inferencesh/skills",
	"jazz":            ".jazz/skills",
	"junie":           ".junie/skills",
	"iflow-cli":       ".iflow/skills",
	"kilo":            ".kilo/skills",
	"kimchi":          ".config/kimchi/harness/skills",
	"kiro":            ".kiro/skills",
	"kiro-cli":        ".kiro/skills",
	"kode":            ".kode/skills",
	"lingma":          ".lingma/skills",
	"mcpjam":          ".mcpjam/skills",
	"minimax-code":    ".minimax/skills",
	"mistral-vibe":    ".vibe/skills",
	"moxby":           ".moxby/skills",
	"mux":             ".mux/skills",
	"opencode":        ".config/opencode/skills",
	"openhands":       ".openhands/skills",
	"ona":             ".ona/skills",
	"pi":              ".pi/agent/skills",
	"posit-assistant": ".posit/assistant/skills",
	"qoder":           ".qoder/skills",
	"qoder-cn":        ".qoder-cn/skills",
	"qwen-code":       ".qwen/skills",
	"reasonix":        ".reasonix/skills",
	"rovodev":         ".rovodev/skills",
	"tabnine-cli":     ".tabnine/agent/skills",
	"terramind":       ".terramind/skills",
	"tinycloud":       ".tinycloud/skills",
	"trae":            ".trae/skills",
	"trae-cn":         ".trae-cn/skills",
	"zcode":           ".zcode/skills",
	"zencoder":        ".zencoder/skills",
	"zenflow":         ".zencoder/skills",
	"neovate":         ".neovate/skills",
	"pochi":           ".pochi/skills",
	"adal":            ".adal/skills",
}

// KnownAgent reports whether an agent id has a skills path mapping.
func KnownAgent(agent string) bool {
	_, ok := globalSkills[agent]
	return ok
}

// AllAgents returns every known agent id, sorted (DefaultAgents first).
func AllAgents() []string {
	out := append([]string{}, DefaultAgents...)
	seen := map[string]bool{}
	for _, a := range out {
		seen[a] = true
	}
	var rest []string
	for a := range globalSkills {
		if !seen[a] {
			rest = append(rest, a)
		}
	}
	sort.Strings(rest)
	return append(out, rest...)
}

// CentralDir is Magic's global skills source: $MAGIC_HOME/packs/skills.
//
// Source: internal | docs/library-layout.md (layout is frozen) | verified
// Never: treat ~/.agents/skills as Magic's central store — it is some other tool's
// directory; Magic only projects *into* it (AGENTS.md → 真相源与数据方向).
func CentralDir(magicHome string) string {
	return filepath.Join(magicHome, "packs", "skills")
}

// ProjectSource is the repo-level skills library: <project>/.agents/skills.
//
// Source: upstream | vercel-labs/skills project-scope convention (.agents/skills) |
//
//	checked 2026-09-13 | verified
func ProjectSource(projectRoot string) string {
	return filepath.Join(projectRoot, ".agents", "skills")
}

// AgentDotDir returns the agent config directory name under a scope root
// (e.g. ".claude"). Project layouts are only defined for the product-adapter
// harnesses; the long tail is global-only.
//
// Source: upstream | vercel-labs/skills README project-scope paths (same snapshot
//
//	as globalSkills) + internal/adapters/* for Magic-owned harnesses |
//	checked 2026-09-13 | verified
//
// `antigravity-cli` and the `.agents/skills` crowd resolve to ".agents": those tools
// read the industry-standard workspace directory rather than their own dot-dir
// (per the upstream table).
func AgentDotDir(agent string) (string, error) {
	switch agent {
	case "claude", "claude-code":
		return ".claude", nil
	case "codex":
		return ".codex", nil
	case "cursor":
		return ".cursor", nil
	case "pi":
		return ".pi", nil
	case "grok":
		return ".grok", nil
	case "opencode":
		return ".opencode", nil
	case "copilot", "github-copilot":
		return ".copilot", nil
	case "trae":
		return ".trae", nil
	case "trae-cn":
		return ".trae-cn", nil
	case "codebuddy", "workbuddy":
		return ".codebuddy", nil
	case "qoder":
		return ".qoder", nil
	case "qoder-cn":
		return ".qoder-cn", nil
	case "qwen-code":
		return ".qwen", nil
	case "iflow-cli":
		return ".iflow", nil
	case "zcode":
		return ".zcode", nil
	case "kiro":
		return ".kiro", nil
	case "kilo":
		return ".kilo", nil
	case "continue":
		return ".continue", nil
	case "droid":
		return ".factory", nil
	case "antigravity-cli":
		return ".agents", nil
	case "cline", "dexto", "kimi-code", "kimi-code-cli", "loaf", "sarvam-code", "warp", "zed":
		return ".agents", nil
	default:
		return "", fmt.Errorf("unknown project agent %q (project scope supports: claude,codex,cursor,pi,grok,opencode,copilot,trae,trae-cn,codebuddy,qoder,qoder-cn,qwen-code,continue,droid,antigravity-cli,cline,dexto,kimi-code,kimi-code-cli,loaf,sarvam-code,warp,zed,kiro,kilo,iflow-cli,zcode)", agent)
	}
}

// GlobalTargetDir returns the harness skills directory for global scope. The
// table above stays verbatim (D11); its .config/ entries are expanded through
// $XDG_CONFIG_HOME here, at the single place the table is read.
func GlobalTargetDir(agent, userHome string) (string, error) {
	rel, ok := globalSkills[agent]
	if !ok {
		return "", fmt.Errorf("unknown agent %q (see magic skills sync --target all)", agent)
	}
	return globalPath(userHome, rel), nil
}

// globalAgentDir resolves the global skills dir an agent is synced into.
//
// Order is fixed: the upstream table first, then the agent's own adapter.
//   - Table first because D11 adopts vercel-labs/skills verbatim as fixed
//     config; a table entry is never overridden by what an adapter says.
//   - Adapter second because DefaultAgents names harnesses (hermes, mimocode)
//     the upstream snapshot does not list, yet Magic does declare their skills
//     dir — ~/.hermes/skills (internal/adapters/hermes/hermes.go, source:
//     hermes_cli/config.py) and $XDG_CONFIG_HOME/mimocode/skills
//     (internal/adapters/mimocode/mimocode.go, source: MiMo-Code
//     packages/shared/src/global.ts). An adapter SkillsDir beats no answer.
//
// An id in neither is still an error, with GlobalTargetDir's wording.
func globalAgentDir(agent, userHome string) (string, error) {
	dir, err := GlobalTargetDir(agent, userHome)
	if err == nil {
		return dir, nil
	}
	if a, ok := registry.ByID(agent); ok {
		if own := a.SkillsDir(userHome); own != "" {
			return own, nil
		}
	}
	return "", err
}

// globalPath expands one globalSkills-style $HOME-relative entry to an absolute
// directory. Only .config/ entries move; every other entry is joined onto home.
func globalPath(userHome, rel string) string {
	// ExpandConfig leaves non-.config/ entries untouched, so an unchanged result
	// means the entry is a plain $HOME-relative path.
	if expanded := appdir.ExpandConfig(userHome, rel); expanded != rel {
		return expanded
	}
	return filepath.Join(userHome, filepath.FromSlash(rel))
}

// ProjectTargetDir returns <project>/.{agent}/skills (project layout).
func ProjectTargetDir(agent, projectRoot string) (string, error) {
	dot, err := AgentDotDir(agent)
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, dot, "skills"), nil
}

// ResolveProjectRoot: --project DIR, else git root, else cwd.
func ResolveProjectRoot(projectArg, cwd string) (string, error) {
	if projectArg != "" {
		abs, err := filepath.Abs(projectArg)
		if err != nil {
			return "", err
		}
		st, err := os.Stat(abs)
		if err != nil || !st.IsDir() {
			return "", fmt.Errorf("project directory does not exist: %s", projectArg)
		}
		return filepath.Clean(abs), nil
	}
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	cmd := exec.Command("git", "-C", cwd, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err == nil {
		root := strings.TrimSpace(string(out))
		if root != "" {
			return filepath.Clean(root), nil
		}
	}
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

// ParseAgents expands comma lists / repeats into unique valid agent ids.
func ParseAgents(values []string) ([]string, error) {
	if len(values) == 0 {
		return append([]string{}, DefaultAgents...), nil
	}
	seen := map[string]bool{}
	var out []string
	for _, v := range values {
		for _, part := range strings.Split(v, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if part == "all" {
				return AllAgents(), nil
			}
			if !KnownAgent(part) {
				return nil, fmt.Errorf("unknown agent %q (see magic skills sync --target all)", part)
			}
			if !seen[part] {
				seen[part] = true
				out = append(out, part)
			}
		}
	}
	if len(out) == 0 {
		return append([]string{}, DefaultAgents...), nil
	}
	return out, nil
}

// ValidSkillName rejects empty / path-like names.
func ValidSkillName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.Contains(name, "/") && !strings.Contains(name, string(filepath.Separator))
}

// LexicalAbs normalizes a path without requiring the target to exist
// (so broken symlinks still have a comparable destination).
func LexicalAbs(path, base string) (string, error) {
	if !filepath.IsAbs(path) {
		if base == "" {
			return "", fmt.Errorf("relative path %q needs a base", path)
		}
		path = filepath.Join(base, path)
	}
	cleaned := filepath.Clean(path)
	if !filepath.IsAbs(cleaned) {
		abs, err := filepath.Abs(cleaned)
		if err != nil {
			return "", err
		}
		cleaned = abs
	}
	return cleaned, nil
}

// LinkDestination returns the absolute lexical destination of a symlink.
func LinkDestination(link string) (string, error) {
	raw, err := os.Readlink(link)
	if err != nil {
		return "", err
	}
	parent, err := filepath.Abs(filepath.Dir(link))
	if err != nil {
		return "", err
	}
	// Prefer real parent dir if it exists
	if st, err := os.Stat(parent); err == nil && st.IsDir() {
		if real, err := filepath.EvalSymlinks(parent); err == nil {
			parent = real
		}
	}
	return LexicalAbs(raw, parent)
}

// IsOrphanedSkillLink reports broken links that look like skill-library pointers
// (.agents/skills or packs/skills).
func IsOrphanedSkillLink(link string) bool {
	fi, err := os.Lstat(link)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		return false
	}
	dest, err := LinkDestination(link)
	if err != nil {
		return false
	}
	if st, err := os.Stat(dest); err == nil && st.IsDir() {
		return false
	}
	slash := filepath.ToSlash(dest)
	return strings.Contains(slash, "/.agents/skills/") || strings.Contains(slash, "/packs/skills/")
}
