package skills

import (
	"fmt"
	"path/filepath"
	"sort"

	"magic/internal/registry"
)

// GlobalDir describes how one agent's global skills directory is shared.
//
// Display only: this decides nothing about what `magic apply` or `magic skills
// sync` writes. Path comes from globalSkills (D11, never re-derived);
// CompatReads records what a harness is *documented* to read and is not a path
// Magic writes to.
type GlobalDir struct {
	Path string
	// SharedWith are the other *tools* whose globalSkills entry is the same
	// directory, alphabetically and with aliases collapsed to one id. A tool
	// reading a directory shared with six others sees their skills too, which is
	// why `magic skills status` shows it; an alias of itself is not a second tool.
	SharedWith []string
	// Primary is true when Path is the harness's own skills path, i.e. the one
	// its adapter declares (adapter.SkillsDir). False means the table points at a
	// directory owned by another tool or by convention (a compatibility path),
	// or the harness has no adapter of its own to confirm it.
	Primary bool
	// CompatReads are extra directories the harness reads on top of its primary
	// path, as stated by its own documentation, relative to the scope root. They
	// are evidence, not a code-verified write target: grok's .claude/.cursor
	// entries are documented as lowest priority and configurable, codex reads
	// .agents/skills at the workspace tier only.
	CompatReads []string
}

// GlobalDirInfo describes agent's global skills dir: where it points, who else
// reads it, whether the agent owns it, and what else the agent is documented to
// read.
func GlobalDirInfo(agent, userHome string) (GlobalDir, error) {
	rel, ok := globalSkills[agent]
	if !ok {
		return GlobalDir{}, fmt.Errorf("unknown agent %q (see magic skills sync --target all)", agent)
	}
	path := globalPath(userHome, rel)
	return GlobalDir{
		Path:        path,
		SharedWith:  sharedWith(agent, rel),
		Primary:     isPrimaryPath(agent, userHome, path),
		CompatReads: append([]string(nil), compatReadsDoc[agent]...),
	}, nil
}

// agentAliases groups ids that name the same harness, first id canonical. They
// share one global skills dir because they are the same product under two names,
// not because two tools read one directory, so they must not show up as each
// other's "shared with".
//
// Source: internal | docs/decisions.md D15 (`workbuddy` = CodeBuddy) +
//
//	internal/skills/paths.go: AgentDotDir branches `claude` with `claude-code`,
//	`kimi-code` with `kimi-code-cli` and `codebuddy` with `workbuddy`, and the
//	adapter ids are `claude` / `kimi-code` / `workbuddy` (registry.List) |
//	checked 2026-11-14 | verified
//
// Never: add a group for ids that only share a path in the upstream table
// (.agents/skills) — those are different tools, which is exactly what
// SharedWith is for. Keep the canonical id first: it is the one Magic's own
// adapter and DefaultAgents use, and the only one status shows.
var agentAliases = [][]string{
	{"claude", "claude-code"},
	{"kimi-code", "kimi-code-cli"},
	{"workbuddy", "codebuddy"},
}

// canonicalAgent maps an id to the canonical id of its alias group, or returns
// it unchanged when it has no alias.
func canonicalAgent(agent string) string {
	for _, group := range agentAliases {
		for _, id := range group {
			if id == agent {
				return group[0]
			}
		}
	}
	return agent
}

// sharedWith lists the other tools the table points at the same directory,
// one entry per tool. Grouping goes by the table's relative entry, so two
// .config/ agents stay grouped even when $XDG_CONFIG_HOME moves them; ids are
// collapsed through agentAliases first, so `kimi-code-cli` is reported as
// `kimi-code` and never twice.
func sharedWith(agent, rel string) []string {
	seen := map[string]bool{canonicalAgent(agent): true}
	var out []string
	for id, other := range globalSkills {
		if other != rel {
			continue
		}
		canonical := canonicalAgent(id)
		if seen[canonical] {
			continue
		}
		seen[canonical] = true
		out = append(out, canonical)
	}
	sort.Strings(out)
	return out
}

// isPrimaryPath reports whether dir is the path the agent's own adapter declares
// as its skills dir. An agent with no adapter of its own, or one that declares no
// skills dir (workbuddy), is not the primary owner of that directory.
func isPrimaryPath(agent, userHome, dir string) bool {
	for _, a := range registry.List() {
		if string(a.ID()) != agent {
			continue
		}
		own := a.SkillsDir(userHome)
		if own == "" {
			return false
		}
		ownAbs, ownErr := LexicalAbs(own, "")
		dirAbs, dirErr := LexicalAbs(dir, "")
		if ownErr != nil || dirErr != nil {
			return filepath.Clean(own) == filepath.Clean(dir)
		}
		return ownAbs == dirAbs
	}
	return false
}

// compatReadsDoc records the extra directories a harness reads besides its own
// primary path, exactly as the harness documents them (relative to the scope
// root). Magic writes nothing here because of this table: it exists so
// `magic skills status` can say "the shared directory is read by this tool too".
//
// Source: vendor-doc | pi CHANGELOG "global skills from ~/.agents/skills, in
// addition to existing .pi skill paths" | verified upstream, date not recorded
//
// Source: vendor-doc | grok crates/codegen/xai-grok-pager/docs/user-guide/
// 08-skills.md: also scans .agents/skills at each tier; ~/.claude/skills and
// ~/.cursor/skills are "Lowest" priority and configurable | verified upstream,
// date not recorded
//
// Source: vendor-doc | codex official docs "scans .agents/skills in every
// directory" — workspace tier, NOT under $HOME | verified upstream, date not
// recorded
//
// Source: internal | docs/harness-profiles.md: cursor "也认 .agents/skills/";
// trae "也支持标准 .agents/skills/"(设置里开关，重名时 .trae/skills 优先);
// antigravity-cli ".agents/skills/（工作区）"; lingma "支持 .agents/skills/" |
//
//	checked 2026-09-13 | verified
//
// Never: add an entry here to make a write decision, and never add a harness
// whose own docs are still marked ❓ (qoder's user-level skills dir is one).
var compatReadsDoc = map[string][]string{
	"pi":              {".agents/skills"},
	"grok":            {".agents/skills", ".claude/skills", ".cursor/skills"},
	"codex":           {".agents/skills"},
	"cursor":          {".agents/skills"},
	"trae":            {".agents/skills"},
	"antigravity-cli": {".agents/skills"},
	"lingma":          {".agents/skills"},
}
