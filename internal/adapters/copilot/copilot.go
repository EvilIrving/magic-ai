// Package copilot is the Magic adapter for GitHub Copilot CLI (root ~/.copilot).
//
// Source: local-measure | ~/.copilot/skills on this Mac | measured 2026-09-13 | verified
// Source: vendor-doc | docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-mcp-servers
//   | checked 2026-09-13 | verified
//
// Never: mirror ~/.copilot/config.json - it carries auth/runtime state
// (docs/harness-profiles.md §3).
package copilot

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

// Adapter mirrors ~/.copilot user-editable config (GitHub Copilot CLI).
// config.json holds auth/runtime state and is denied.
type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "copilot" }
func (a *Adapter) Name() string        { return "GitHub Copilot CLI" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, Agents: true, MCP: true, Hooks: true}
}
func (a *Adapter) Detect(home string) bool {
	root := filepath.Join(home, ".copilot")
	if st, err := os.Stat(root); err == nil && st.IsDir() {
		return true
	}
	_, err := os.Stat(filepath.Join(root, "settings.json"))
	return err == nil
}
func (a *Adapter) SkillsDir(home string) string {
	return filepath.Join(home, ".copilot", "skills")
}
func (a *Adapter) AgentsDir(home string) string {
	return filepath.Join(home, ".copilot", "agents")
}
func (a *Adapter) MirrorRoot(home string) string {
	return filepath.Join(home, ".copilot")
}
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "copilot")
}
