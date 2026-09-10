package skills

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DefaultAgents are skillmesh-compatible skill sync targets.
// Includes cursor (skills-only) and opencode (also a Magic adapter).
var DefaultAgents = []string{"claude", "codex", "cursor", "pi", "grok", "opencode", "copilot", "trae"}

// CentralDir is Magic's global skills source: $MAGIC_HOME/packs/skills.
func CentralDir(magicHome string) string {
	return filepath.Join(magicHome, "packs", "skills")
}

// ProjectSource is the repo-level skills library: <project>/.agents/skills.
func ProjectSource(projectRoot string) string {
	return filepath.Join(projectRoot, ".agents", "skills")
}

// AgentDotDir returns the agent config directory name under a scope root
// (e.g. ".claude"). For global pi, Magic uses ".pi/agent" via GlobalTargetDir.
func AgentDotDir(agent string) (string, error) {
	switch agent {
	case "claude":
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
	case "copilot":
		return ".copilot", nil
	case "trae":
		return ".trae", nil
	default:
		return "", fmt.Errorf("unknown agent %q (want: claude,codex,cursor,pi,grok,opencode,copilot,trae)", agent)
	}
}

// GlobalTargetDir returns the harness skills directory for global scope.
// Maps to Magic adapter SkillsDir where available; cursor uses ~/.cursor/skills.
func GlobalTargetDir(agent, userHome string) (string, error) {
	switch agent {
	case "claude":
		return filepath.Join(userHome, ".claude", "skills"), nil
	case "codex":
		return filepath.Join(userHome, ".codex", "skills"), nil
	case "cursor":
		return filepath.Join(userHome, ".cursor", "skills"), nil
	case "pi":
		// Magic adapter: ~/.pi/agent/skills (not skillmesh's ~/.pi/skills)
		return filepath.Join(userHome, ".pi", "agent", "skills"), nil
	case "grok":
		return filepath.Join(userHome, ".grok", "skills"), nil
	case "opencode":
		return filepath.Join(userHome, ".config", "opencode", "skills"), nil
	case "copilot":
		return filepath.Join(userHome, ".copilot", "skills"), nil
	case "trae":
		return filepath.Join(userHome, ".trae", "skills"), nil
	default:
		return "", fmt.Errorf("unknown agent %q", agent)
	}
}

// ProjectTargetDir returns <project>/.{agent}/skills (skillmesh project layout).
func ProjectTargetDir(agent, projectRoot string) (string, error) {
	dot, err := AgentDotDir(agent)
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, dot, "skills"), nil
}

// ResolveProjectRoot mirrors skillmesh: --project DIR, else git root, else cwd.
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
				return append([]string{}, DefaultAgents...), nil
			}
			if _, err := AgentDotDir(part); err != nil {
				return nil, err
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
