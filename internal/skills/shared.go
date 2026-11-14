package skills

import (
	"fmt"
	"path/filepath"
)

// sharedTargetRel is the industry-standard shared global skills directory. The
// group in globalSkills (`cline`, `dexto`, `kimi-code`, `kimi-code-cli`, `loaf`,
// `sarvam-code`, `warp`, `zed`) resolves to it.
//
// Source: internal | globalSkills (.agents/skills group) + vercel-labs/skills
//
//	project-scope convention | checked 2026-09-13 | verified
const sharedTargetRel = ".agents/skills"

// SharedTargetDir is the shared global skills layer, ~/.agents/skills, expanded
// the same way the globals table is.
func SharedTargetDir(userHome string) string {
	return globalPath(userHome, sharedTargetRel)
}

// SyncShared writes the shared layer and nothing else: one projection of the
// global library into ~/.agents/skills, no agent's own primary directory. The
// per-agent paths are what plain `magic skills sync` is for, and --target names
// the same thing, so combining either with --shared is a usage error rather than
// a silent no-op.
//
// What the single copy buys: of the eight ids whose primary path is .agents/skills,
// Magic has an adapter for only cline, kimi-code and zed. One copy there serves
// dexto, kimi-code-cli, loaf, sarvam-code and warp as well, without an adapter
// each. For a harness that reads it on top of its own path (pi, grok, codex,
// cursor, trae) the shared copy is a fallback: its own directory wins.
func SyncShared(o Options) (*Result, error) {
	if o.Scope != "" && o.Scope != ScopeGlobal {
		return nil, fmt.Errorf("--shared writes the global shared layer; cannot combine with --scope %s", o.Scope)
	}
	if len(o.Agents) > 0 {
		return nil, fmt.Errorf("--shared writes only the shared layer (%s); cannot combine with --target, which names a harness's own primary dir", sharedTargetRel)
	}
	o.Scope = ScopeGlobal
	o = o.withDefaults()
	r := &Result{}
	r.line("sync %s scope: %s", ScopeGlobal, CentralDir(o.MagicHome))
	if err := projectSharedLayer(o, r); err != nil {
		return r, err
	}
	if r.Stats.Issues > 0 {
		return r, fmt.Errorf("skills completed with %d issue(s)", r.Stats.Issues)
	}
	return r, nil
}

// projectSharedLayer links (or copies) the global library into the shared layer
// with the same link/copy rules `magic apply` uses.
//
// The directory belongs to the other tools too, so an entry Magic did not create
// is reported and kept: no prune, no replacement, and --force does not reach
// here. Setup problems are returned without printing, because Sync already
// reported them.
func projectSharedLayer(o Options, r *Result) error {
	ctx, err := setupScope(o, ScopeGlobal)
	if err != nil {
		return err
	}
	if !ctx.sourceAvailable() {
		return fmt.Errorf("source missing: %s", ctx.PrimarySource)
	}
	if err := ctx.buildEffective(o.Skills); err != nil {
		return err
	}
	target := SharedTargetDir(o.UserHome)
	r.line("shared layer: %s", target)
	if len(ctx.Effective) == 0 {
		r.warn("no skills to project into %s", target)
		return nil
	}
	for _, skill := range ctx.Effective {
		source := ctx.SkillSource[skill]
		link := filepath.Join(target, skill)
		action, aerr := EnsureLink(link, source, o.DryRun)
		switch {
		case aerr != nil && action != "skip-real":
			r.errf("shared: %v", aerr)
		case action == "ok":
			r.Stats.Unchanged++
			if o.Verbose {
				r.line("  shared ok: %s", link)
			}
		case action == "skip-unmanaged":
			dest, _ := LinkDestination(link)
			r.Stats.Conflicts++
			r.warn("shared: keep existing link %s -> %s (not created by Magic)", link, dest)
		case action == "skip-real":
			// EnsureLink refuses to replace a real path; here that is the normal
			// outcome for a user's own entry, so it is a report, not an error.
			r.Stats.Conflicts++
			r.warn("shared: keep real file/dir %s", link)
		case o.DryRun:
			r.Stats.Created++
			r.line("  [dry-run] would link: %s -> %s", link, source)
		default:
			r.Stats.Created++
			r.line("  %s: %s -> %s", action, link, source)
		}
	}
	return nil
}
