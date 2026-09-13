// Package zcode is the Magic adapter for ZCode (Z.ai agentic IDE/CLI).
// MCP: ~/.zcode/cli/config.json under mcp.servers (the .agents/mcp.json
// compatibility file is left to the custom mirror). Skills: ~/.zcode/skills.

// Source: internal | docs/supported-harnesses.md §调研来源 (ZCode / Z.ai) | checked 2026-09-13 | stale?
//   Nested `mcp.servers` key, unlike every flat `mcpServers` neighbour.
package zcode

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "zcode" }
func (a *Adapter) Name() string        { return "ZCode" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, MCP: true, Hooks: true}
}
func (a *Adapter) root(home string) string { return filepath.Join(home, ".zcode") }
func (a *Adapter) Detect(home string) bool {
	if st, err := os.Stat(a.root(home)); err == nil && st.IsDir() {
		return true
	}
	_, err := os.Stat(filepath.Join(a.root(home), "cli", "config.json"))
	return err == nil
}
func (a *Adapter) SkillsDir(home string) string  { return filepath.Join(a.root(home), "skills") }
func (a *Adapter) AgentsDir(home string) string  { return "" }
func (a *Adapter) MirrorRoot(home string) string { return a.root(home) }
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "zcode")
}
