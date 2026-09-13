package main

import (
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
}

func (f *skillsFlags) bindCommon(c *cobra.Command) {
	c.Flags().StringVar(&f.scope, "scope", "global", "global|project|all")
	c.Flags().StringVar(&f.project, "project", "", "project root (git root or cwd if omitted)")
	c.Flags().StringArrayVar(&f.targets, "target", nil, "agent: claude,codex,cursor,pi,grok,opencode,copilot,trae (repeatable or comma list)")
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
  global opencode → ~/.config/opencode/skills
  global copilot → ~/.copilot/skills
  global trae → ~/.trae/skills
  project   → <project>/.claude|codex|cursor|pi|grok|opencode|copilot|trae/skills`,
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
			r, err := skills.Sync(o)
			printResult(r)
			if r != nil {
				fmt.Println(skills.SummaryLine("sync", r.Stats))
			}
			return err
		},
	})
	syncCmd.Flags().BoolVar(&f.noPrune, "no-prune", false, "keep stale managed links")

	add(&cobra.Command{
		Use:   "status [skill...]",
		Short: "Per-agent: linked / missing / conflicts / stale / external (read-only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := f.opts(args)
			if err != nil {
				return err
			}
			r, err := skills.Status(o)
			printResult(r)
			return err
		},
	})

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
		fmt.Println("  magic skills sync --scope project")
		fmt.Println()
		fmt.Println(strings.TrimSpace(cmd.Long))
		return nil
	}

	return root
}

// silence unused import if os only needed elsewhere — keep for future
