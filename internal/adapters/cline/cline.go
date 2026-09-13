// Package cline is the Magic adapter for Cline (VS Code extension + CLI).
// MCP is in ~/.cline/mcp.json; skills are read from ~/.agents/skills.
// Cline hooks are code plugins (SDK), handled by the per-harness custom mirror.

// Source: vendor-doc | docs.cline.bot/mcp/configuring-mcp-servers
//   (states ~/.cline/mcp.json) | checked 2026-09-13 | verified
//   Skills use the shared ~/.agents/skills dir, not a Cline-owned directory.
package cline

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "cline" }
func (a *Adapter) Name() string        { return "Cline" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, MCP: true, Hooks: true}
}
func (a *Adapter) Detect(home string) bool {
	root := filepath.Join(home, ".cline")
	if st, err := os.Stat(root); err == nil && st.IsDir() {
		return true
	}
	_, err := os.Stat(filepath.Join(root, "mcp.json"))
	return err == nil
}
func (a *Adapter) SkillsDir(home string) string {
	return filepath.Join(home, ".agents", "skills")
}
func (a *Adapter) AgentsDir(home string) string { return "" }
func (a *Adapter) MirrorRoot(home string) string {
	return filepath.Join(home, ".cline")
}
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "cline")
}
