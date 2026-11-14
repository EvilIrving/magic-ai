package core

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"magic/internal/adapter"
	"magic/internal/agents"
	"magic/internal/backup"
	"magic/internal/skills"
)

// ApplyOptions controls apply.
type ApplyOptions struct {
	MagicHome string
	UserHome  string
	DryRun    bool
	Only      string
	// BackupDir is the backup set this run records into. Empty on a dry run and
	// for a plan, which writes nothing.
	BackupDir string
}

// ApplyAdapter restores the mirrored custom tree for one adapter, then projects
// the shared instruction source (packs/AGENTS.md) to the harness's user-level
// instruction file.
// Structured config files are field-level merged (decisions.md D5); the local
// library copy is the final content (no secret placeholder reinjection).
//
// A harness the user home does not show (Detect) is left alone unless --only
// names it: apply must not invent ~/.factory/ on a machine that never had
// Droid, and provider / MCP / shared instructions use the same gate. --only is
// the explicit way to pre-install a harness's config before the tool is opened.
func ApplyAdapter(a adapter.Adapter, opts ApplyOptions, backupDir string) ([]string, error) {
	var log []string
	if !a.Implemented() {
		return []string{fmt.Sprintf("[%s] skipped (not implemented)", a.ID())}, nil
	}
	if !a.Detect(opts.UserHome) && opts.Only != string(a.ID()) {
		return []string{fmt.Sprintf("[%s] skipped (not detected)", a.ID())}, nil
	}
	srcRoot := filepath.Join(opts.MagicHome, a.LibraryMirrorRel())
	dstRoot := a.MirrorRoot(opts.UserHome)
	if dstRoot == "" {
		return log, nil
	}
	st, err := os.Stat(srcRoot)
	mirrorMissing := err != nil || !st.IsDir()
	if mirrorMissing {
		log = append(log, fmt.Sprintf("[%s] no library mirror at %s", a.ID(), srcRoot))
	} else if err := walkMirror(a, opts, backupDir, srcRoot, dstRoot, &log); err != nil {
		return log, err
	}

	// The shared instruction source is not a mirrored file, so it is projected
	// whether or not the library holds a mirror tree for this adapter.
	sharedLogs, err := applySharedInstructions(a, opts, backupDir)
	if err != nil {
		return log, err
	}
	log = append(log, sharedLogs...)
	if mirrorMissing {
		return log, nil
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

// walkMirror restores the library mirror tree of one adapter, file by file.
func walkMirror(a adapter.Adapter, opts ApplyOptions, backupDir, srcRoot, dstRoot string, log *[]string) error {
	return filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
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
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			mode := os.FileMode(0o644)
			if secretBearingName(rel) {
				mode = 0o600
			}
			if err := mergeLibraryFile(backupDir, a.ID(), dst, rel, data, mode); err != nil {
				return err
			}
		}
		*log = append(*log, fmt.Sprintf("[%s] restore %s", a.ID(), rel))
		return nil
	})
}

// recordWrite adds one target to the backup set's manifest. An existing target is
// copied into the set first (I4: back up before overwriting); a missing one is
// recorded as create, so rollback deletes it. out is the content apply is about
// to write: its fingerprint lets rollback tell apply's own output from a later
// user edit.
func recordWrite(backupDir string, id adapter.ID, rel, target string, mode os.FileMode, out []byte) error {
	if backupDir == "" {
		return nil
	}
	action, backupRel := backup.ActionCreate, ""
	if _, err := os.Stat(target); err == nil {
		action, backupRel = backup.ActionUpdate, filepath.Join(string(id), rel)
	}
	return backup.Record(backupDir, backup.Entry{
		Adapter: string(id),
		Rel:     rel,
		Target:  target,
		Backup:  backupRel,
		Action:  action,
		Mode:    backup.ModeString(mode),
		After:   backup.FingerprintOf(out),
	})
}

// ApplySkills projects $MAGIC_HOME/packs/skills to all Caps.Skills adapters.
// Only an installed harness (Detect) is projected, unless --only names it.
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
		if !a.Detect(opts.UserHome) && opts.Only != string(a.ID()) {
			continue
		}
		if d := a.SkillsDir(opts.UserHome); d != "" {
			dirs = append(dirs, d)
		}
	}
	return skills.ApplyProjects(central, dirs, opts.DryRun)
}

// ApplyAgents projects packs/agents sub-agent defs to Caps.Agents harnesses.
// Only an installed harness (Detect) is projected, unless --only names it.
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
		if !a.Detect(opts.UserHome) && opts.Only != string(a.ID()) {
			continue
		}
		if d := a.AgentsDir(opts.UserHome); d != "" {
			dirs = append(dirs, d)
		}
	}
	return agents.ApplyProjects(lib, dirs, opts.DryRun)
}
