// Package kilo is the Magic adapter for Kilo Code CLI.
// MCP: <config>/kilo.json (key mcp, OpenCode-style). Skills: ~/.kilo/skills.
// Config root is $XDG_CONFIG_HOME/kilo, ~/.config/kilo when XDG_CONFIG_HOME is
// unset.
//
// Source: vendor-doc | kilo.ai/docs | checked 2026-09-13 | stale?
//   Docs confirm the OpenCode `mcp` dialect; the config filename has varied by
//   version, so kiloPath probes several candidates instead of guessing one.
//
// Source: vendor-blog | blog.kilo.ai — Kilo Code CLI relocated its config and data dirs to XDG_CONFIG_HOME and XDG_DATA_HOME | checked 2026-11-14 | verified
//
// Never: apply that XDG move to SkillsDir. Kilo keeps agent skills in
// ~/.kilo/skills, not under the config root (docs/supported-harnesses.md).
//
// Never: declare hooks. Kilo has no native hook events — the request is still
// open upstream (docs/hooks-design.md), so Caps.Hooks stays false and Magic
// writes no hooks for it. Claiming otherwise would make `magic adapters`
// advertise a projection that lands nowhere.
package kilo

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
	"magic/internal/appdir"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "kilo" }
func (a *Adapter) Name() string        { return "Kilo Code" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	// No Hooks: Kilo has no native hook events (docs/hooks-design.md).
	return adapter.Caps{Skills: true, MCP: true}
}
func (a *Adapter) root(home string) string {
	return filepath.Join(appdir.ConfigHome(home), "kilo")
}
func (a *Adapter) Detect(home string) bool {
	if st, err := os.Stat(a.root(home)); err == nil && st.IsDir() {
		return true
	}
	for _, name := range []string{"kilo.json", "kilo.jsonc", "config.json"} {
		if _, err := os.Stat(filepath.Join(a.root(home), name)); err == nil {
			return true
		}
	}
	return false
}
func (a *Adapter) SkillsDir(home string) string  { return filepath.Join(home, ".kilo", "skills") }
func (a *Adapter) AgentsDir(home string) string  { return "" }
func (a *Adapter) MirrorRoot(home string) string { return a.root(home) }
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "kilo")
}
