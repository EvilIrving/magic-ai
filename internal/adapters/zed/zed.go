// Package zed is the Magic adapter for the Zed editor.
// MCP: ~/.config/zed/settings.json under context_servers (stdio only; Zed has
// no native HTTP transport). Skills: ~/.agents/skills.

// Source: internal | docs/supported-harnesses.md §调研来源 (Zed) | checked 2026-09-13 | stale?
//   zed.dev MCP doc URLs currently 404. Zed names MCP `context_servers` and has
//   no native HTTP transport (stdio only).
package zed

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "zed" }
func (a *Adapter) Name() string        { return "Zed" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, MCP: true}
}
func (a *Adapter) root(home string) string {
	return filepath.Join(home, ".config", "zed")
}
func (a *Adapter) Detect(home string) bool {
	if st, err := os.Stat(a.root(home)); err == nil && st.IsDir() {
		return true
	}
	_, err := os.Stat(filepath.Join(a.root(home), "settings.json"))
	return err == nil
}
func (a *Adapter) SkillsDir(home string) string  { return filepath.Join(home, ".agents", "skills") }
func (a *Adapter) AgentsDir(home string) string  { return "" }
func (a *Adapter) MirrorRoot(home string) string { return a.root(home) }
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "zed")
}
