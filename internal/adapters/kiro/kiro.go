// Package kiro is the Magic adapter for Kiro (AWS).
// MCP: ~/.kiro/settings/mcp.json (key mcpServers). Skills: ~/.kiro/skills.
// Agents: ~/.kiro/agents. Hooks: ~/.kiro/hooks (via the custom mirror).

// Source: vendor-doc | kiro.dev/docs/mcp (documents mcpServers; the page gives no
//   file path) | checked 2026-09-13 | path stale?
package kiro

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "kiro" }
func (a *Adapter) Name() string        { return "Kiro" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, Agents: true, MCP: true, Hooks: true}
}
func (a *Adapter) root(home string) string { return filepath.Join(home, ".kiro") }
func (a *Adapter) Detect(home string) bool {
	if st, err := os.Stat(a.root(home)); err == nil && st.IsDir() {
		return true
	}
	_, err := os.Stat(filepath.Join(a.root(home), "settings", "mcp.json"))
	return err == nil
}
func (a *Adapter) SkillsDir(home string) string  { return filepath.Join(a.root(home), "skills") }
func (a *Adapter) AgentsDir(home string) string  { return filepath.Join(a.root(home), "agents") }
func (a *Adapter) MirrorRoot(home string) string { return a.root(home) }
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "kiro")
}
