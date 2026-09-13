package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Scope is global, project, or all.
type Scope string

const (
	ScopeGlobal  Scope = "global"
	ScopeProject Scope = "project"
	ScopeAll     Scope = "all"
)

// Options configures a skill sync operation.
type Options struct {
	MagicHome     string // required for global / include-global
	UserHome      string // global target root (usually $HOME)
	Scope         Scope
	ProjectRoot   string // resolved; empty → ResolveProjectRoot
	ProjectArg    string // raw --project
	Cwd           string
	Agents        []string // empty → DefaultAgents
	Skills        []string // optional filter
	IncludeGlobal bool
	DryRun        bool
	Force         bool
	NoPrune       bool // sync prunes stale managed links unless set
	Orphaned      bool
	Verbose       bool
	ImportCopy    bool
	ImportName    string
	AdoptFrom     string
	SyncAfter     bool
}

// Stats accumulates mutation counters.
type Stats struct {
	Created   int
	Updated   int
	Removed   int
	Unchanged int
	Conflicts int
	Issues    int
	Imported  int
}

// Result holds logs and stats from an operation.
type Result struct {
	Lines []string
	Stats Stats
}

func (r *Result) line(format string, args ...any) {
	r.Lines = append(r.Lines, fmt.Sprintf(format, args...))
}

func (r *Result) warn(format string, args ...any) {
	r.Stats.Issues++
	r.line("warn: "+format, args...)
}

func (r *Result) errf(format string, args ...any) {
	r.Stats.Issues++
	r.line("error: "+format, args...)
}

// scopeCtx is the resolved layout for one scope invocation.
type scopeCtx struct {
	Scope           Scope
	ScopeRoot       string
	PrimarySource   string
	SecondarySource string
	ManagedSources  []string
	Agents          []string
	Effective       []string // skill names
	SkillSource     map[string]string
	Inherited       map[string]bool
}

func (o Options) withDefaults() Options {
	out := o
	if out.Scope == "" {
		out.Scope = ScopeGlobal
	}
	if out.UserHome == "" {
		if h, err := os.UserHomeDir(); err == nil {
			out.UserHome = h
		}
	}
	return out
}

func setupScope(o Options, scope Scope) (*scopeCtx, error) {
	agents, err := ParseAgents(o.Agents)
	if err != nil {
		return nil, err
	}
	ctx := &scopeCtx{
		Scope:       scope,
		Agents:      agents,
		SkillSource: map[string]string{},
		Inherited:   map[string]bool{},
	}
	switch scope {
	case ScopeGlobal:
		if o.MagicHome == "" {
			return nil, fmt.Errorf("MAGIC_HOME required for global skills scope")
		}
		ctx.ScopeRoot = o.UserHome
		ctx.PrimarySource = CentralDir(o.MagicHome)
		ctx.ManagedSources = []string{ctx.PrimarySource}
	case ScopeProject:
		root := o.ProjectRoot
		if root == "" {
			root, err = ResolveProjectRoot(o.ProjectArg, o.Cwd)
			if err != nil {
				return nil, err
			}
		}
		ctx.ScopeRoot = root
		ctx.PrimarySource = ProjectSource(root)
		ctx.ManagedSources = []string{ctx.PrimarySource}
		if o.IncludeGlobal {
			if o.MagicHome == "" {
				return nil, fmt.Errorf("MAGIC_HOME required for --include-global")
			}
			ctx.SecondarySource = CentralDir(o.MagicHome)
			if ctx.SecondarySource != ctx.PrimarySource {
				ctx.ManagedSources = append(ctx.ManagedSources, ctx.SecondarySource)
			}
		}
	default:
		return nil, fmt.Errorf("unknown scope %q", scope)
	}
	return ctx, nil
}

func (ctx *scopeCtx) targetDir(agent string) (string, error) {
	if ctx.Scope == ScopeGlobal {
		return GlobalTargetDir(agent, ctx.ScopeRoot)
	}
	return ProjectTargetDir(agent, ctx.ScopeRoot)
}

// agentTarget is one resolved agent → skills dir.
type agentTarget struct {
	agent string
	dir   string
}

// uniqueTargets resolves agents to dirs, dropping duplicate dirs: several
// long-tail agents share the same global path (e.g. ~/.agents/skills).
func (ctx *scopeCtx) uniqueTargets() (out []agentTarget, errs []error) {
	seen := map[string]bool{}
	for _, agent := range ctx.Agents {
		dir, err := ctx.targetDir(agent)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		abs, _ := filepath.Abs(dir)
		if seen[abs] {
			continue
		}
		seen[abs] = true
		out = append(out, agentTarget{agent: agent, dir: dir})
	}
	return out, errs
}

func (ctx *scopeCtx) sourceAvailable() bool {
	if st, err := os.Stat(ctx.PrimarySource); err == nil && st.IsDir() {
		return true
	}
	if ctx.SecondarySource != "" {
		if st, err := os.Stat(ctx.SecondarySource); err == nil && st.IsDir() {
			return true
		}
	}
	return false
}

func (ctx *scopeCtx) buildEffective(filter []string) error {
	primary, err := ListSkillDirs(ctx.PrimarySource)
	if err != nil {
		return err
	}
	var names []string
	for _, s := range primary {
		names = append(names, s)
		ctx.SkillSource[s] = filepath.Join(ctx.PrimarySource, s)
	}
	if ctx.SecondarySource != "" {
		secondary, err := ListSkillDirs(ctx.SecondarySource)
		if err != nil {
			return err
		}
		for _, s := range secondary {
			if _, ok := ctx.SkillSource[s]; ok {
				continue
			}
			names = append(names, s)
			ctx.SkillSource[s] = filepath.Join(ctx.SecondarySource, s)
			ctx.Inherited[s] = true
		}
	}
	if len(filter) > 0 {
		var selected []string
		seen := map[string]bool{}
		for _, s := range filter {
			if !ValidSkillName(s) {
				return fmt.Errorf("invalid skill name: %s", s)
			}
			src, ok := ctx.SkillSource[s]
			if !ok {
				// allow filter miss to be reported by caller via empty check
				return fmt.Errorf("skill not in source: %s", s)
			}
			_ = src
			if !seen[s] {
				seen[s] = true
				selected = append(selected, s)
			}
		}
		names = selected
	}
	sort.Strings(names)
	ctx.Effective = names
	return nil
}

func linkMatchesPath(target, expected string) bool {
	fi, err := os.Lstat(target)
	if err != nil {
		return false
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		dest, err := LinkDestination(target)
		if err != nil {
			return false
		}
		exp, _ := LexicalAbs(expected, "")
		if dest == exp {
			return true
		}
	}
	// When source is itself a symlink, matching resolved dirs also counts.
	stT, errT := os.Stat(target)
	stE, errE := os.Stat(expected)
	if errT != nil || errE != nil || !stT.IsDir() || !stE.IsDir() {
		return false
	}
	realT, err1 := filepath.EvalSymlinks(target)
	realE, err2 := filepath.EvalSymlinks(expected)
	if err1 != nil || err2 != nil {
		return false
	}
	return filepath.Clean(realT) == filepath.Clean(realE)
}

func linkIsManaged(link string, managed []string) bool {
	fi, err := os.Lstat(link)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		return false
	}
	skill := filepath.Base(link)
	dest, err := LinkDestination(link)
	if err != nil {
		return false
	}
	for _, src := range managed {
		want, _ := LexicalAbs(filepath.Join(src, skill), "")
		if dest == want {
			return true
		}
	}
	return false
}

func linkIsStaleManaged(link string, managed []string) bool {
	if !linkIsManaged(link, managed) {
		return false
	}
	dest, err := LinkDestination(link)
	if err != nil {
		return true
	}
	st, err := os.Stat(dest)
	return err != nil || !st.IsDir()
}

func ensureDir(path string, dryRun bool, r *Result) error {
	if st, err := os.Stat(path); err == nil {
		if st.IsDir() {
			return nil
		}
		r.errf("path is not a directory: %s", path)
		return fmt.Errorf("not a directory: %s", path)
	}
	if dryRun {
		r.line("  [dry-run] mkdir %s", path)
		return nil
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		r.errf("cannot create directory: %s", path)
		return err
	}
	r.line("  created dir: %s", path)
	return nil
}

func createLink(source, target string, dryRun bool, r *Result) error {
	if dryRun {
		r.line("  [dry-run] link: %s -> %s", filepath.Base(target), source)
		r.Stats.Created++
		return nil
	}
	if err := os.Symlink(source, target); err != nil {
		r.errf("cannot create symlink: %s", target)
		return err
	}
	r.line("  linked: %s", filepath.Base(target))
	r.Stats.Created++
	return nil
}

func replaceLink(source, target string, dryRun bool, r *Result) error {
	fi, err := os.Lstat(target)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		r.errf("refusing to replace non-symlink: %s", target)
		return fmt.Errorf("not a symlink: %s", target)
	}
	if dryRun {
		r.line("  [dry-run] update link: %s -> %s", filepath.Base(target), source)
		r.Stats.Updated++
		return nil
	}
	tmp := target + ".magic-skills-tmp"
	_ = os.Remove(tmp)
	if err := os.Symlink(source, tmp); err != nil {
		r.errf("cannot create temp symlink: %s", tmp)
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		r.errf("cannot update symlink: %s", target)
		return err
	}
	r.line("  updated link: %s", filepath.Base(target))
	r.Stats.Updated++
	return nil
}

func removeLink(target, reason string, dryRun bool, r *Result) error {
	fi, err := os.Lstat(target)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		r.errf("refusing to delete non-symlink: %s", target)
		return fmt.Errorf("not a symlink: %s", target)
	}
	if dryRun {
		r.line("  [dry-run] remove link: %s (%s)", filepath.Base(target), reason)
		r.Stats.Removed++
		return nil
	}
	if err := os.Remove(target); err != nil {
		r.errf("cannot remove symlink: %s", target)
		return err
	}
	r.line("  removed link: %s (%s)", filepath.Base(target), reason)
	r.Stats.Removed++
	return nil
}

func cleanupTarget(target string, ctx *scopeCtx, o Options, r *Result) {
	ents, err := os.ReadDir(target)
	if err != nil {
		return
	}
	filter := map[string]bool{}
	for _, s := range o.Skills {
		filter[s] = true
	}
	for _, e := range ents {
		entry := filepath.Join(target, e.Name())
		fi, err := os.Lstat(entry)
		if err != nil || fi.Mode()&os.ModeSymlink == 0 {
			continue
		}
		skill := e.Name()
		if len(filter) > 0 && !filter[skill] {
			continue
		}
		if linkIsStaleManaged(entry, ctx.ManagedSources) {
			_ = removeLink(entry, "source skill gone", o.DryRun, r)
		} else if o.Orphaned && IsOrphanedSkillLink(entry) {
			_ = removeLink(entry, "orphaned skill link", o.DryRun, r)
		}
	}
}

func syncTarget(target string, ctx *scopeCtx, o Options, r *Result) {
	if err := ensureDir(target, o.DryRun, r); err != nil {
		return
	}
	for _, skill := range ctx.Effective {
		source := ctx.SkillSource[skill]
		targetSkill := filepath.Join(target, skill)
		if linkMatchesPath(targetSkill, source) {
			r.Stats.Unchanged++
			if o.Verbose {
				r.line("  synced: %s", skill)
			}
			continue
		}
		fi, err := os.Lstat(targetSkill)
		if err == nil && fi.Mode()&os.ModeSymlink != 0 {
			if linkIsManaged(targetSkill, ctx.ManagedSources) || o.Force {
				_ = replaceLink(source, targetSkill, o.DryRun, r)
			} else {
				dest, _ := LinkDestination(targetSkill)
				r.warn("conflict, keep external link: %s -> %s (use --force)", targetSkill, dest)
				r.Stats.Conflicts++
			}
			continue
		}
		if err == nil {
			r.warn("conflict, keep real file/dir: %s", targetSkill)
			r.Stats.Conflicts++
			continue
		}
		_ = createLink(source, targetSkill, o.DryRun, r)
	}
	if !o.NoPrune {
		cleanupTarget(target, ctx, o, r)
	}
}

// runScopes executes fn for each requested scope.
func runScopes(o Options, fn func(Scope, *scopeCtx, *Result) error) (*Result, error) {
	o = o.withDefaults()
	r := &Result{}
	scopes := []Scope{o.Scope}
	if o.Scope == ScopeAll {
		scopes = []Scope{ScopeGlobal, ScopeProject}
	}
	var firstErr error
	for _, sc := range scopes {
		ctx, err := setupScope(o, sc)
		if err != nil {
			r.errf("%v", err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if err := fn(sc, ctx, r); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	if r.Stats.Issues > 0 && firstErr == nil {
		firstErr = fmt.Errorf("skills completed with %d issue(s)", r.Stats.Issues)
	}
	return r, firstErr
}

// Sync projects skills from the source dir into agent skill dirs.
func Sync(o Options) (*Result, error) {
	return runScopes(o, func(sc Scope, ctx *scopeCtx, r *Result) error {
		if !ctx.sourceAvailable() {
			r.errf("source skills dir missing: %s", ctx.PrimarySource)
			return fmt.Errorf("source missing: %s", ctx.PrimarySource)
		}
		if err := ctx.buildEffective(o.Skills); err != nil {
			r.errf("%v", err)
			return err
		}
		if len(ctx.Effective) == 0 {
			r.warn("no skills to sync in %s", ctx.PrimarySource)
		}
		r.line("sync %s scope: %s", sc, ctx.PrimarySource)
		if ctx.SecondarySource != "" {
			r.line("  inherit global: %s", ctx.SecondarySource)
		}
		targets, errs := ctx.uniqueTargets()
		for _, e := range errs {
			r.errf("%v", e)
		}
		for _, t := range targets {
			r.line("- %s: %s", t.agent, t.dir)
			syncTarget(t.dir, ctx, o, r)
		}
		return nil
	})
}

// Status reports per-agent link state (read-only).
func Status(o Options) (*Result, error) {
	return runScopes(o, func(sc Scope, ctx *scopeCtx, r *Result) error {
		if err := ctx.buildEffective(o.Skills); err != nil {
			r.errf("%v", err)
			// continue with empty/partial effective set
		}
		r.line("status %s scope: %s", sc, ctx.PrimarySource)
		if ctx.SecondarySource != "" {
			r.line("  inherit global: %s", ctx.SecondarySource)
		}
		if !ctx.sourceAvailable() {
			r.line("  source missing; still scanning managed stale links")
		}
		targets, errs := ctx.uniqueTargets()
		for _, e := range errs {
			r.errf("%v", e)
		}
		for _, t := range targets {
			statusTarget(t.dir, t.agent, ctx, o, r)
		}
		return nil
	})
}

func statusTarget(target, agent string, ctx *scopeCtx, o Options, r *Result) {
	if st, err := os.Stat(target); err != nil || !st.IsDir() {
		r.line("- %s: target missing (%s)", agent, target)
		return
	}
	linked, missing, conflicts, stale, external := 0, 0, 0, 0, 0
	inEffective := map[string]bool{}
	for _, skill := range ctx.Effective {
		inEffective[skill] = true
		source := ctx.SkillSource[skill]
		targetSkill := filepath.Join(target, skill)
		if linkMatchesPath(targetSkill, source) {
			linked++
			if o.Verbose {
				r.line("  linked: %s", skill)
			}
		} else if fi, err := os.Lstat(targetSkill); err == nil && fi.Mode()&os.ModeSymlink != 0 {
			dest, _ := LinkDestination(targetSkill)
			r.line("  conflict: %s -> %s", skill, dest)
			conflicts++
			r.Stats.Conflicts++
		} else if err == nil {
			r.line("  conflict: %s occupied by real file/dir", skill)
			conflicts++
			r.Stats.Conflicts++
		} else {
			r.line("  missing: %s", skill)
			missing++
		}
	}
	ents, _ := os.ReadDir(target)
	for _, e := range ents {
		skill := e.Name()
		if inEffective[skill] {
			continue
		}
		entry := filepath.Join(target, skill)
		fi, err := os.Lstat(entry)
		if err != nil {
			continue
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			if linkIsStaleManaged(entry, ctx.ManagedSources) {
				r.line("  stale managed: %s", skill)
				stale++
			} else if _, err := os.Stat(entry); err != nil {
				r.line("  broken external: %s", skill)
				external++
			} else {
				if o.Verbose {
					r.line("  external link: %s", skill)
				}
				external++
			}
		} else {
			if o.Verbose {
				r.line("  external file/dir: %s", skill)
			}
			external++
		}
	}
	r.line("- %s: linked %d, missing %d, conflicts %d, stale %d, external %d",
		agent, linked, missing, conflicts, stale, external)
}

// Clean removes stale managed links (and optionally orphaned).
func Clean(o Options) (*Result, error) {
	return runScopes(o, func(sc Scope, ctx *scopeCtx, r *Result) error {
		r.line("clean %s scope: %s", sc, ctx.PrimarySource)
		if o.Orphaned {
			r.line("  also remove orphaned .agents/skills and packs/skills broken links")
		}
		targets, errs := ctx.uniqueTargets()
		for _, e := range errs {
			r.errf("%v", e)
		}
		for _, t := range targets {
			if st, err := os.Stat(t.dir); err != nil || !st.IsDir() {
				if o.Verbose {
					r.line("- %s: target missing, skip", t.agent)
				}
				continue
			}
			r.line("- %s: %s", t.agent, t.dir)
			cleanupTarget(t.dir, ctx, o, r)
		}
		return nil
	})
}

// List lists effective skills in scope.
func List(o Options) (*Result, error) {
	return runScopes(o, func(sc Scope, ctx *scopeCtx, r *Result) error {
		if err := ctx.buildEffective(nil); err != nil {
			return err
		}
		r.line("skills (%s):", sc)
		if len(ctx.Effective) == 0 {
			r.line("  (none; source: %s)", ctx.PrimarySource)
			return nil
		}
		for _, skill := range ctx.Effective {
			if ctx.Inherited[skill] {
				r.line("  %s (inherited global)", skill)
			} else {
				r.line("  %s", skill)
			}
		}
		return nil
	})
}

// Init creates source + target dirs for scope.
func Init(o Options) (*Result, error) {
	return runScopes(o, func(sc Scope, ctx *scopeCtx, r *Result) error {
		r.line("init %s scope: %s", sc, ctx.ScopeRoot)
		if err := ensureDir(ctx.PrimarySource, o.DryRun, r); err != nil {
			return err
		}
		targets, errs := ctx.uniqueTargets()
		for _, e := range errs {
			r.errf("%v", e)
		}
		for _, t := range targets {
			_ = ensureDir(t.dir, o.DryRun, r)
		}
		return nil
	})
}

// Doctor checks SKILL.md, bad links, conflicts.
func Doctor(o Options) (*Result, error) {
	return runScopes(o, func(sc Scope, ctx *scopeCtx, r *Result) error {
		r.line("doctor %s scope: %s", sc, ctx.PrimarySource)
		checkSourceSkills(ctx.PrimarySource, "primary", o, r)
		if ctx.SecondarySource != "" {
			checkSourceSkills(ctx.SecondarySource, "global-inherited", o, r)
			primary, _ := ListSkillDirs(ctx.PrimarySource)
			for _, skill := range primary {
				if st, err := os.Stat(filepath.Join(ctx.SecondarySource, skill)); err == nil && st.IsDir() {
					r.warn("project skill overrides global: %s", skill)
				}
			}
		}
		if err := ctx.buildEffective(o.Skills); err != nil {
			r.errf("%v", err)
			return err
		}
		targets, errs := ctx.uniqueTargets()
		for _, e := range errs {
			r.errf("%v", e)
		}
		for _, t := range targets {
			doctorTarget(t.dir, t.agent, ctx, o, r)
		}
		return nil
	})
}

func checkSourceSkills(source, label string, o Options, r *Result) {
	st, err := os.Stat(source)
	if err != nil || !st.IsDir() {
		r.errf("%s source missing: %s", label, source)
		return
	}
	skills, _ := ListSkillDirs(source)
	if len(skills) == 0 {
		r.warn("%s source empty: %s", label, source)
		return
	}
	for _, skill := range skills {
		md := filepath.Join(source, skill, "SKILL.md")
		if _, err := os.Stat(md); err != nil {
			r.errf("%s skill missing SKILL.md: %s", label, skill)
		} else if o.Verbose {
			r.line("  ok: %s/%s", label, skill)
		}
	}
}

func doctorTarget(target, agent string, ctx *scopeCtx, o Options, r *Result) {
	if st, err := os.Stat(target); err != nil || !st.IsDir() {
		r.warn("%s target missing: %s", agent, target)
		return
	}
	inEffective := map[string]bool{}
	for _, skill := range ctx.Effective {
		inEffective[skill] = true
		source := ctx.SkillSource[skill]
		targetSkill := filepath.Join(target, skill)
		if linkMatchesPath(targetSkill, source) {
			if o.Verbose {
				r.line("  ok: %s/%s", agent, skill)
			}
		} else if fi, err := os.Lstat(targetSkill); err == nil && fi.Mode()&os.ModeSymlink != 0 {
			dest, _ := LinkDestination(targetSkill)
			r.errf("%s/%s points wrong: %s", agent, skill, dest)
		} else if err == nil {
			r.errf("%s/%s occupied by real file/dir", agent, skill)
		} else {
			r.warn("%s/%s not linked yet", agent, skill)
		}
	}
	ents, _ := os.ReadDir(target)
	for _, e := range ents {
		if inEffective[e.Name()] {
			continue
		}
		entry := filepath.Join(target, e.Name())
		fi, err := os.Lstat(entry)
		if err != nil || fi.Mode()&os.ModeSymlink == 0 {
			continue
		}
		if linkIsStaleManaged(entry, ctx.ManagedSources) {
			r.errf("%s/%s stale managed link", agent, e.Name())
		} else if _, err := os.Stat(entry); err != nil {
			r.warn("%s/%s broken external link", agent, e.Name())
		}
	}
}

// Remove disables managed links for named skills (source kept).
func Remove(o Options) (*Result, error) {
	if len(o.Skills) == 0 {
		return nil, fmt.Errorf("remove/disable requires at least one skill name")
	}
	for _, s := range o.Skills {
		if !ValidSkillName(s) {
			return nil, fmt.Errorf("invalid skill name: %s", s)
		}
	}
	return runScopes(o, func(sc Scope, ctx *scopeCtx, r *Result) error {
		r.line("remove %s scope target links (source kept)", sc)
		targets, errs := ctx.uniqueTargets()
		for _, e := range errs {
			r.errf("%v", e)
		}
		for _, at := range targets {
			target := at.dir
			if st, err := os.Stat(target); err != nil || !st.IsDir() {
				r.line("- %s: target missing, skip", at.agent)
				continue
			}
			for _, skill := range o.Skills {
				targetSkill := filepath.Join(target, skill)
				fi, err := os.Lstat(targetSkill)
				if err != nil {
					if o.Verbose {
						r.line("  absent: %s/%s", at.agent, skill)
					}
					continue
				}
				if fi.Mode()&os.ModeSymlink != 0 {
					if linkIsManaged(targetSkill, ctx.ManagedSources) || o.Force {
						_ = removeLink(targetSkill, "manual disable", o.DryRun, r)
					} else {
						dest, _ := LinkDestination(targetSkill)
						r.warn("keep external link: %s -> %s (use --force)", targetSkill, dest)
						r.Stats.Conflicts++
					}
				} else {
					r.warn("keep real file/dir: %s", targetSkill)
					r.Stats.Conflicts++
				}
			}
		}
		return nil
	})
}

// ImportPath links or copies a local skill dir into the scope source.
func ImportPath(o Options, path string) (*Result, error) {
	if path == "" {
		return nil, fmt.Errorf("import requires a local skill directory")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(abs)
	if err != nil || !st.IsDir() {
		return nil, fmt.Errorf("import path is not a directory: %s", path)
	}
	if _, err := os.Stat(filepath.Join(abs, "SKILL.md")); err != nil {
		return nil, fmt.Errorf("import path missing SKILL.md: %s", abs)
	}
	name := o.ImportName
	if name == "" {
		name = filepath.Base(abs)
	}
	if !ValidSkillName(name) {
		return nil, fmt.Errorf("invalid skill name: %s", name)
	}

	o.Skills = nil
	r, err := runScopes(o, func(sc Scope, ctx *scopeCtx, r *Result) error {
		r.line("import into %s source: %s", sc, ctx.PrimarySource)
		if err := importIntoLibrary(abs, name, ctx, o, r); err != nil {
			return err
		}
		if o.SyncAfter {
			so := o
			so.Skills = []string{name}
			sr, serr := Sync(so)
			if sr != nil {
				r.Lines = append(r.Lines, sr.Lines...)
				r.Stats.Created += sr.Stats.Created
				r.Stats.Updated += sr.Stats.Updated
				r.Stats.Removed += sr.Stats.Removed
				r.Stats.Conflicts += sr.Stats.Conflicts
				r.Stats.Issues += sr.Stats.Issues
			}
			return serr
		}
		return nil
	})
	return r, err
}

func importIntoLibrary(external, skill string, ctx *scopeCtx, o Options, r *Result) error {
	dest := filepath.Join(ctx.PrimarySource, skill)
	if _, err := os.Lstat(dest); err == nil {
		r.line("  source already has %s, skip", skill)
		return nil
	}
	// Reject importing from inside the primary source tree.
	primAbs, _ := filepath.Abs(ctx.PrimarySource)
	extAbs, _ := filepath.Abs(external)
	if primAbs != "" && (extAbs == primAbs || strings.HasPrefix(extAbs, primAbs+string(filepath.Separator))) {
		r.errf("refusing to import from inside source: %s", external)
		return fmt.Errorf("refuse import from source tree")
	}
	if err := ensureDir(ctx.PrimarySource, o.DryRun, r); err != nil {
		return err
	}
	mode := "link"
	if o.ImportCopy {
		mode = "copy"
	}
	if o.DryRun {
		r.line("  [dry-run] %s import: %s <- %s", mode, skill, external)
		r.Stats.Imported++
		return nil
	}
	if o.ImportCopy {
		if err := copyPath(external, dest); err != nil {
			r.errf("copy failed: %v", err)
			return err
		}
		r.line("  copied import: %s", skill)
	} else {
		if err := os.Symlink(external, dest); err != nil {
			r.errf("cannot create source symlink: %s", dest)
			return err
		}
		r.line("  linked import: %s", skill)
	}
	r.Stats.Imported++
	return nil
}

// Adopt pulls unmanaged skills from an agent dir into the source.
func Adopt(o Options) (*Result, error) {
	if o.AdoptFrom == "" {
		return nil, fmt.Errorf("adopt requires --from AGENT")
	}
	if _, err := AgentDotDir(o.AdoptFrom); err != nil {
		return nil, err
	}
	return runScopes(o, func(sc Scope, ctx *scopeCtx, r *Result) error {
		sourceTarget, err := ctx.targetDir(o.AdoptFrom)
		if err != nil {
			return err
		}
		if st, err := os.Stat(sourceTarget); err != nil || !st.IsDir() {
			r.errf("adopt source agent dir missing: %s", sourceTarget)
			return fmt.Errorf("missing: %s", sourceTarget)
		}
		if err := ensureDir(ctx.PrimarySource, o.DryRun, r); err != nil {
			return err
		}
		r.line("adopt from %s into %s source", o.AdoptFrom, sc)
		filter := map[string]bool{}
		for _, s := range o.Skills {
			filter[s] = true
		}
		var adopted []string
		ents, _ := os.ReadDir(sourceTarget)
		for _, e := range ents {
			skill := e.Name()
			if len(filter) > 0 && !filter[skill] {
				continue
			}
			entry := filepath.Join(sourceTarget, skill)
			fi, err := os.Lstat(entry)
			if err != nil {
				continue
			}
			if fi.Mode()&os.ModeSymlink != 0 {
				if _, err := os.Stat(entry); err != nil {
					r.warn("skip broken link: %s/%s", o.AdoptFrom, skill)
					continue
				}
			}
			st, err := os.Stat(entry)
			if err != nil || !st.IsDir() {
				continue
			}
			if _, err := os.Stat(filepath.Join(entry, "SKILL.md")); err != nil {
				if o.Verbose {
					r.line("  skip no SKILL.md: %s", skill)
				}
				continue
			}
			if linkIsManaged(entry, ctx.ManagedSources) {
				if o.Verbose {
					r.line("  already managed: %s", skill)
				}
				continue
			}
			existed := false
			if _, err := os.Lstat(filepath.Join(ctx.PrimarySource, skill)); err == nil {
				existed = true
			}
			real, err := filepath.EvalSymlinks(entry)
			if err != nil {
				real = entry
			}
			before := r.Stats.Imported
			if err := importIntoLibrary(real, skill, ctx, o, r); err != nil {
				continue
			}
			if !existed && r.Stats.Imported > before {
				adopted = append(adopted, skill)
			}
		}
		if len(adopted) == 0 {
			r.line("  no new skills to adopt")
			return nil
		}
		if o.SyncAfter {
			so := o
			so.Skills = adopted
			sr, serr := Sync(so)
			if sr != nil {
				r.Lines = append(r.Lines, sr.Lines...)
				r.Stats.Created += sr.Stats.Created
				r.Stats.Updated += sr.Stats.Updated
				r.Stats.Removed += sr.Stats.Removed
				r.Stats.Conflicts += sr.Stats.Conflicts
				r.Stats.Issues += sr.Stats.Issues
			}
			return serr
		}
		return nil
	})
}

// SummaryLine formats completion stats.
func SummaryLine(cmd string, s Stats) string {
	switch cmd {
	case "sync", "clean", "remove", "disable":
		return fmt.Sprintf("done: created %d, updated %d, removed %d, unchanged %d, conflicts %d",
			s.Created, s.Updated, s.Removed, s.Unchanged, s.Conflicts)
	case "import", "adopt":
		return fmt.Sprintf("done: imported %d, created %d, updated %d, conflicts %d",
			s.Imported, s.Created, s.Updated, s.Conflicts)
	default:
		return ""
	}
}
