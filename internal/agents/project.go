package agents

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Convention: packs/agents/<name>.md — one markdown file per sub-agent.
// On apply, for adapters with Caps.Agents, copy to <AgentsDir>/<name>.md.
// Directories packs/agents/<name>/ are also accepted (copy tree or AGENT.md).

const Doc = `Sub-agent convention (packs/agents/)
=====================================
Each sub-agent is a markdown file:

  packs/agents/<name>.md

Optional: packs/agents/<name>/AGENT.md (or any *.md inside) — the directory
is mirrored as <name>/ under the harness agents path.

On apply, Magic copies into each Caps.Agents harness AgentsDir:
  ~/.claude/agents/   ~/.codex/agents/   ~/.grok/agents/   ~/.pi/agent/agents/

"agents" means sub-agent definitions — not an adapter named Central Agents.
`

// ListLibrary lists sub-agent entry names under packs/agents.
func ListLibrary(agentsRoot string) ([]string, error) {
	ents, err := os.ReadDir(agentsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range ents {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if strings.EqualFold(name, "README.md") || strings.EqualFold(name, "README") {
			continue
		}
		if e.IsDir() {
			names = append(names, name)
			continue
		}
		if strings.EqualFold(filepath.Ext(name), ".md") || strings.EqualFold(filepath.Ext(name), ".yaml") || strings.EqualFold(filepath.Ext(name), ".yml") {
			names = append(names, name)
		}
	}
	return names, nil
}

// ApplyProjects copies library sub-agents into each harness agents dir.
func ApplyProjects(libAgents string, harnessAgentsDirs []string, dryRun bool) ([]string, error) {
	entries, err := ListLibrary(libAgents)
	if err != nil {
		return nil, err
	}
	var log []string
	for _, dir := range harnessAgentsDirs {
		if dir == "" {
			continue
		}
		if !dryRun {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return log, err
			}
		}
		for _, name := range entries {
			src := filepath.Join(libAgents, name)
			dstName := name
			// normalize: foo.md stays; directory stays
			dst := filepath.Join(dir, dstName)
			if dryRun {
				log = append(log, fmt.Sprintf("would project %s → %s", src, dst))
				continue
			}
			info, err := os.Stat(src)
			if err != nil {
				log = append(log, fmt.Sprintf("skip %s: %v", src, err))
				continue
			}
			if info.IsDir() {
				if err := copyDir(src, dst); err != nil {
					return log, err
				}
			} else {
				if err := copyFile(src, dst); err != nil {
					return log, err
				}
			}
			log = append(log, fmt.Sprintf("project %s → %s", src, dst))
		}
	}
	return log, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, dst)
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	ents, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range ents {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		info, err := e.Info()
		if err != nil {
			return err
		}
		if info.IsDir() {
			if err := copyDir(s, d); err != nil {
				return err
			}
		} else {
			if err := copyFile(s, d); err != nil {
				return err
			}
		}
	}
	return nil
}
