// Package droid is the Magic adapter for Factory Droid CLI.
// MCP: ~/.factory/mcp.json (key mcpServers). Skills: ~/.factory/skills.
// Agents (custom droids): ~/.factory/droids. Hooks: ~/.factory/hooks.

// Source: internal | docs/supported-harnesses.md §调研来源 (Factory Droid) | checked 2026-09-13 | stale?
//   factory.ai/docs does not publish these paths - confirm on a real install.
//   Custom agents live in `droids/`, and the root is ~/.factory, not ~/.droid.
package droid

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "droid" }
func (a *Adapter) Name() string        { return "Droid (Factory)" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, Agents: true, MCP: true, Hooks: true, Models: true}
}
func (a *Adapter) root(home string) string { return filepath.Join(home, ".factory") }
func (a *Adapter) Detect(home string) bool {
	if st, err := os.Stat(a.root(home)); err == nil && st.IsDir() {
		return true
	}
	_, err := os.Stat(filepath.Join(a.root(home), "settings.json"))
	return err == nil
}
func (a *Adapter) SkillsDir(home string) string  { return filepath.Join(a.root(home), "skills") }
func (a *Adapter) AgentsDir(home string) string  { return filepath.Join(a.root(home), "droids") }
func (a *Adapter) MirrorRoot(home string) string { return a.root(home) }
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "droid")
}
