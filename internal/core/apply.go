package core

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"magic/internal/adapter"
	"magic/internal/agents"
	"magic/internal/skills"
)

// ApplyOptions controls apply.
type ApplyOptions struct {
	MagicHome string
	UserHome  string
	DryRun    bool
	Only      string
}

// ApplyAdapter restores the mirrored custom tree for one adapter.
// Structured config files are field-level merged (decisions.md D5); the local
// library copy is the final content (no secret placeholder reinjection).
func ApplyAdapter(a adapter.Adapter, opts ApplyOptions, backupDir string) ([]string, error) {
	var log []string
	if !a.Implemented() {
		return []string{fmt.Sprintf("[%s] skipped (not implemented)", a.ID())}, nil
	}
	srcRoot := filepath.Join(opts.MagicHome, a.LibraryMirrorRel())
	dstRoot := a.MirrorRoot(opts.UserHome)
	if dstRoot == "" {
		return log, nil
	}
	st, err := os.Stat(srcRoot)
	if err != nil || !st.IsDir() {
		log = append(log, fmt.Sprintf("[%s] no library mirror at %s", a.ID(), srcRoot))
		return log, nil
	}

	err = filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		// skip partial metadata files that are not mirrored into harness root as-is
		if base == "claude.json.partial.json" || base == "_claude.json" {
			return nil
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return nil
		}
		dst := filepath.Join(dstRoot, rel)
		if !opts.DryRun {
			if _, err := os.Stat(dst); err == nil && backupDir != "" {
				_ = BackupPath(dst, backupDir, filepath.Join(string(a.ID()), rel))
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			mode := os.FileMode(0o644)
			if secretBearingName(rel) {
				mode = 0o600
			}
			if err := mergeLibraryFile(dst, rel, data, mode); err != nil {
				return err
			}
		}
		log = append(log, fmt.Sprintf("[%s] restore %s", a.ID(), rel))
		return nil
	})
	if err != nil {
		return log, err
	}

	// Claude: merge partial json config keys into ~/.claude.json (non-destructive merge of extracted keys)
	if a.ID() == "claude" {
		extra, err := applyClaudePartial(opts, backupDir)
		if err != nil {
			return log, err
		}
		log = append(log, extra...)
	}
	return log, nil
}

// ApplySkills projects $MAGIC_HOME/packs/skills to all Caps.Skills adapters.
func ApplySkills(adapters []adapter.Adapter, opts ApplyOptions) ([]string, error) {
	central := skills.CentralDir(opts.MagicHome)
	var dirs []string
	for _, a := range adapters {
		if !a.Implemented() || !a.Caps().Skills {
			continue
		}
		if opts.Only != "" && string(a.ID()) != opts.Only {
			continue
		}
		if d := a.SkillsDir(opts.UserHome); d != "" {
			dirs = append(dirs, d)
		}
	}
	return skills.ApplyProjects(central, dirs, opts.DryRun)
}

// ApplyAgents projects packs/agents sub-agent defs to Caps.Agents harnesses.
func ApplyAgents(adapters []adapter.Adapter, opts ApplyOptions) ([]string, error) {
	lib := filepath.Join(opts.MagicHome, "packs", "agents")
	var dirs []string
	for _, a := range adapters {
		if !a.Implemented() || !a.Caps().Agents {
			continue
		}
		if opts.Only != "" && string(a.ID()) != opts.Only {
			continue
		}
		if d := a.AgentsDir(opts.UserHome); d != "" {
			dirs = append(dirs, d)
		}
	}
	return agents.ApplyProjects(lib, dirs, opts.DryRun)
}
