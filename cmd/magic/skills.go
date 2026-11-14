package main

import (
	"encoding/json"
	"fmt"

	"strings"

	"github.com/spf13/cobra"

	"magic/internal/core"
	"magic/internal/skills"
)

type skillsFlags struct {
	scope         string
	project       string
	targets       []string
	includeGlobal bool
	dryRun        bool
	force         bool
	noPrune       bool
	orphaned      bool
	verbose       bool
	copy          bool
	name          string
	from          string
	syncAfter     bool
	home          string
	shared        bool
	asJSON        bool
}

func (f *skillsFlags) bindCommon(c *cobra.Command) {
	c.Flags().StringVar(&f.scope, "scope", "global", "global|project|all")
	c.Flags().StringVar(&f.project, "project", "", "project root (git root or cwd if omitted)")
	c.Flags().StringArrayVar(&f.targets, "target", nil, "agent id or 'all' (repeatable/comma list); 'all' covers the full skills path table")
	c.Flags().BoolVar(&f.includeGlobal, "include-global", false, "project scope also inherits $MAGIC_HOME/packs/skills")
	c.Flags().BoolVarP(&f.dryRun, "dry-run", "n", false, "print actions without modifying files")
	c.Flags().BoolVarP(&f.force, "force", "f", false, "replace conflicting symlinks (never real dirs)")
	c.Flags().BoolVarP(&f.verbose, "verbose", "v", false, "show already-correct entries")
	c.Flags().StringVar(&f.home, "home", "", "user home for global targets (fixture)")
}

func (f *skillsFlags) opts(skillArgs []string) (skills.Options, error) {
	mh, err := resolveMagicHome()
	if err != nil {
		return skills.Options{}, err
	}
	uh, err := core.UserHome(f.home)
	if err != nil {
		return skills.Options{}, err
	}
	scope := skills.Scope(f.scope)
	switch scope {
	case skills.ScopeGlobal, skills.ScopeProject, skills.ScopeAll:
	default:
		return skills.Options{}, fmt.Errorf("--scope must be global, project, or all")
	}
	if f.shared && scope != skills.ScopeGlobal {
		return skills.Options{}, fmt.Errorf("--shared writes the global shared layer; cannot combine with --scope %s", f.scope)
	}
	var agents []string
	for _, t := range f.targets {
		agents = append(agents, t)
	}
	return skills.Options{
		MagicHome:     mh,
		UserHome:      uh,
		Scope:         scope,
		ProjectArg:    f.project,
		Agents:        agents,
		Skills:        skillArgs,
		IncludeGlobal: f.includeGlobal,
		DryRun:        f.dryRun,
		Force:         f.force,
		NoPrune:       f.noPrune,
		Orphaned:      f.orphaned,
		Verbose:       f.verbose,
		ImportCopy:    f.copy,
		ImportName:    f.name,
		AdoptFrom:     f.from,
		SyncAfter:     f.syncAfter,
	}, nil
}

func printResult(r *skills.Result) {
	if r == nil {
		return
	}
	for _, line := range r.Lines {
		fmt.Println(line)
	}
}

// skillsStatusRow is the --json shape of one `magic skills status` row. The
// sharing fields are pointers so a project-scope row omits them instead of
// claiming primary=false, and a global row always carries shared_with ([] when
// the directory is not shared).
type skillsStatusRow struct {
	Scope       string    `json:"scope"`
	Agent       string    `json:"agent"`
	Dir         string    `json:"dir"`
	Linked      int       `json:"linked"`
	Missing     int       `json:"missing"`
	Conflicts   int       `json:"conflicts"`
	Stale       int       `json:"stale"`
	External    int       `json:"external"`
	SharedWith  *[]string `json:"shared_with,omitempty"`
	Primary     *bool     `json:"primary,omitempty"`
	CompatReads []string  `json:"compat_reads,omitempty"`
	// NoUpstreamEntry: the upstream skills path table has no row for this agent
	// (hermes, mimocode), so shared_with / primary describe the adapter's own dir.
	NoUpstreamEntry bool `json:"no_upstream_entry,omitempty"`
}

func statusRows(r *skills.Result) []skillsStatusRow {
	rows := []skillsStatusRow{}
	if r == nil {
		return rows
	}
	for _, t := range r.Targets {
		row := skillsStatusRow{
			Scope:     string(t.Scope),
			Agent:     t.Agent,
			Dir:       t.Dir,
			Linked:    t.Linked,
			Missing:   t.Missing,
			Conflicts: t.Conflicts,
			Stale:     t.Stale,
			External:  t.External,
		}
		if t.Scope == skills.ScopeGlobal {
			shared := append([]string{}, t.SharedWith...)
			primary := t.Primary
			row.SharedWith = &shared
			row.Primary = &primary
			row.CompatReads = t.CompatReads
			row.NoUpstreamEntry = t.NoUpstreamEntry
		}
		rows = append(rows, row)
	}
	return rows
}

func cmdSkills() *cobra.Command {
	root := &cobra.Command{
		Use:          "skills",
		Short:        "Manage skills (sync into each agent's skill dirs)",
		SilenceUsage: true,
		Long: `Skills source of truth:
  global:  $MAGIC_HOME/packs/skills  (default MAGIC_HOME=~/.magic-ai)
  project: <project>/.agents/skills

Repo-level sync: magic skills sync --scope project
(Not magic apply — apply only projects the global packs/skills library.)

Targets: claude, codex, cursor, pi, grok, opencode, copilot, trae
  global pi → ~/.pi/agent/skills
  global opencode → $XDG_CONFIG_HOME/opencode/skills (default ~/.config/opencode/skills)
  global copilot → ~/.copilot/skills
  global trae → ~/.trae/skills
  project   → <project>/.claude|codex|cursor|pi|grok|opencode|copilot|trae/skills

Shared layer: magic skills sync --shared
  Writes ONLY the standard ~/.agents/skills directory — no harness's own path
  (that is plain 'magic skills sync'), so --target has nothing to select and is
  refused. Eight ids in the skills path table use ~/.agents/skills as their
  PRIMARY path (cline, dexto, kimi-code, kimi-code-cli, loaf, sarvam-code, warp,
  zed), but Magic has an adapter for only cline, kimi-code and zed — one copy
  there serves the other five as well.
  For a harness that reads it while owning its own path (pi, grok, codex, cursor,
  trae) the shared copy is only a fallback: their own directory wins, so the
  user's main path overrides it. grok reads ~/.claude/skills at "Lowest" priority
  and that scan is configurable; values in ~/.grok/skills win.
  Entries in the shared directory that Magic did not create are kept and
  reported; it is never pruned, and --force does not apply to it.`,
	}

	var f skillsFlags

	add := func(c *cobra.Command) *cobra.Command {
		c.SilenceUsage = true
		f.bindCommon(c)
		root.AddCommand(c)
		return c
	}

	syncCmd := add(&cobra.Command{
		Use:   "sync [skill...]",
		Short: "Sync source → agent skill dirs; repair managed links; prune stale by default",
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := f.opts(args)
			if err != nil {
				return err
			}
			var r *skills.Result
			if f.shared {
				r, err = skills.SyncShared(o)
			} else {
				r, err = skills.Sync(o)
			}
			printResult(r)
			if r != nil {
				fmt.Println(skills.SummaryLine("sync", r.Stats))
			}
			return err
		},
	})
	syncCmd.Flags().BoolVar(&f.noPrune, "no-prune", false, "keep stale managed links")
	syncCmd.Flags().BoolVar(&f.shared, "shared", false, "write only the shared layer ~/.agents/skills (no per-agent primary dirs; cannot combine with --target)")

	statusCmd := add(&cobra.Command{
		Use:   "status [skill...]",
		Short: "Per-agent: shared / primary / linked / missing / conflicts / stale / external (read-only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := f.opts(args)
			if err != nil {
				return err
			}
			r, err := skills.Status(o)
			if f.asJSON {
				b, merr := json.MarshalIndent(statusRows(r), "", "  ")
				if merr != nil {
					return merr
				}
				fmt.Println(string(b))
				return err
			}
			printResult(r)
			return err
		},
	})
	statusCmd.Flags().BoolVar(&f.asJSON, "json", false, "print as JSON (adds shared_with / primary per global target)")

	add(&cobra.Command{
		Use:   "list",
		Short: "List skills in scope; --include-global marks inherited",
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := f.opts(nil)
			if err != nil {
				return err
			}
			r, err := skills.List(o)
			printResult(r)
			return err
		},
	})

	add(&cobra.Command{
		Use:   "doctor [skill...]",
		Short: "Check SKILL.md, bad links, conflicts",
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := f.opts(args)
			if err != nil {
				return err
			}
			r, err := skills.Doctor(o)
			printResult(r)
			return err
		},
	})

	cleanCmd := add(&cobra.Command{
		Use:   "clean [skill...]",
		Short: "Remove managed links whose source is gone",
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := f.opts(args)
			if err != nil {
				return err
			}
			r, err := skills.Clean(o)
			printResult(r)
			if r != nil {
				fmt.Println(skills.SummaryLine("clean", r.Stats))
			}
			return err
		},
	})
	cleanCmd.Flags().BoolVar(&f.orphaned, "orphaned", false, "also remove broken .agents/skills and packs/skills links")

	add(&cobra.Command{
		Use:   "init",
		Short: "Create source + target dirs for scope",
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := f.opts(nil)
			if err != nil {
				return err
			}
			r, err := skills.Init(o)
			printResult(r)
			return err
		},
	})

	imp := add(&cobra.Command{
		Use:   "import PATH",
		Short: "Link (default) or --copy a local skill dir into source (requires SKILL.md)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := f.opts(nil)
			if err != nil {
				return err
			}
			r, err := skills.ImportPath(o, args[0])
			printResult(r)
			if r != nil {
				fmt.Println(skills.SummaryLine("import", r.Stats))
			}
			return err
		},
	})
	imp.Flags().BoolVar(&f.copy, "copy", false, "copy into source instead of symlink")
	imp.Flags().BoolVar(&f.syncAfter, "sync", false, "sync imported skill to targets after import")
	imp.Flags().StringVar(&f.name, "name", "", "name in source library")
	imp.Flags().Bool("link", true, "symlink into source (default)")

	adopt := add(&cobra.Command{
		Use:   "adopt",
		Short: "Adopt unmanaged skills from an agent dir into source",
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := f.opts(args)
			if err != nil {
				return err
			}
			r, err := skills.Adopt(o)
			printResult(r)
			if r != nil {
				fmt.Println(skills.SummaryLine("adopt", r.Stats))
			}
			return err
		},
	})
	adopt.Flags().StringVar(&f.from, "from", "", "source agent (required)")
	adopt.Flags().BoolVar(&f.copy, "copy", false, "copy into source instead of symlink")
	adopt.Flags().BoolVar(&f.syncAfter, "sync", false, "sync adopted skills after adopt")
	_ = adopt.MarkFlagRequired("from")

	add(&cobra.Command{
		Use:   "enable [skill...]",
		Short: "Alias for sync (requires skill names)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := f.opts(args)
			if err != nil {
				return err
			}
			r, err := skills.Sync(o)
			printResult(r)
			if r != nil {
				fmt.Println(skills.SummaryLine("sync", r.Stats))
			}
			return err
		},
	})

	add(&cobra.Command{
		Use:   "disable [skill...]",
		Short: "Alias for remove — drop managed target links, keep source",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := f.opts(args)
			if err != nil {
				return err
			}
			r, err := skills.Remove(o)
			printResult(r)
			if r != nil {
				fmt.Println(skills.SummaryLine("disable", r.Stats))
			}
			return err
		},
	})

	add(&cobra.Command{
		Use:   "remove [skill...]",
		Short: "Remove managed target links for skills (source kept)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := f.opts(args)
			if err != nil {
				return err
			}
			r, err := skills.Remove(o)
			printResult(r)
			if r != nil {
				fmt.Println(skills.SummaryLine("remove", r.Stats))
			}
			return err
		},
	})

	// Default bare `magic skills` → status overview + path hint
	root.RunE = func(cmd *cobra.Command, args []string) error {
		mh, err := resolveMagicHome()
		if err != nil {
			return err
		}
		fmt.Println("Skills")
		fmt.Printf("  global source:  %s\n", skills.CentralDir(mh))
		fmt.Println("  project source: <project>/.agents/skills")
		fmt.Println("  targets:        claude, codex, cursor, pi, grok")
		fmt.Println()
		fmt.Println("Try:")
		fmt.Println("  magic adapters")
		fmt.Println("  magic skills list")
		fmt.Println("  magic skills status")
		fmt.Println("  magic skills status --json")
		fmt.Println("  magic skills sync --scope project")
		fmt.Println("  magic skills sync --shared")
		fmt.Println()
		fmt.Println(strings.TrimSpace(cmd.Long))
		return nil
	}

	return root
}

// silence unused import if os only needed elsewhere — keep for future
