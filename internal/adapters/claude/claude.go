// Package claude is the Magic adapter for Claude Code.
//
// Source: local-measure | ~/.claude/{skills,agents} + ~/.claude.json | measured 2026-09-13 | verified
// Source: vendor-doc | code.claude.com/docs/en/settings | checked 2026-09-13 | verified
//
// Two roots by design: ~/.claude holds per-item dirs (skills, agents) while
// ~/.claude.json holds the user-level MCP map. Magic merges
// packs/harness/claude/claude.json.partial.json into the latter by key
// (field-level merge, docs/apply-design.md).
package claude

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "claude" }
func (a *Adapter) Name() string        { return "Claude Code" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, Agents: true, Themes: true, MCP: true, Hooks: true}
}
func (a *Adapter) Detect(home string) bool {
	if st, err := os.Stat(filepath.Join(home, ".claude")); err == nil && st.IsDir() {
		return true
	}
	_, err := os.Stat(filepath.Join(home, ".claude.json"))
	return err == nil
}
func (a *Adapter) SkillsDir(home string) string {
	return filepath.Join(home, ".claude", "skills")
}
func (a *Adapter) AgentsDir(home string) string {
	return filepath.Join(home, ".claude", "agents")
}
func (a *Adapter) MirrorRoot(home string) string {
	return filepath.Join(home, ".claude")
}
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "claude")
}
