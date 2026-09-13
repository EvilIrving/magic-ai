// Package kilo is the Magic adapter for Kilo Code CLI.
// MCP: ~/.config/kilo/kilo.json (key mcp, OpenCode-style). Skills: ~/.kilo/skills.

// Source: vendor-doc | kilo.ai/docs | checked 2026-09-13 | stale?
//   Docs confirm the OpenCode `mcp` dialect; the config filename has varied by
//   version, so kiloPath probes several candidates instead of guessing one.
package kilo

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "kilo" }
func (a *Adapter) Name() string        { return "Kilo Code" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, MCP: true, Hooks: true}
}
func (a *Adapter) root(home string) string {
	return filepath.Join(home, ".config", "kilo")
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
