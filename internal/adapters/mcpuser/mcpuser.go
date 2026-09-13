// Package mcpuser is the MCP-config mirror (~/.config/mcp/mcp.json), not a harness.
//
// Source: internal | docs/mcp-design.md (generic MCP consumer) | verified
//
// mcpuser is deliberately NOT a model provider: no SwitchWriter, and it must
// not grow a stub pretending to be one (AGENTS.md -> 继续 registry 工作时 #6).
package mcpuser

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "mcpuser" }
func (a *Adapter) Name() string        { return "MCP User Config" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{MCP: true}
}
func (a *Adapter) Detect(home string) bool {
	_, err := os.Stat(filepath.Join(home, ".config", "mcp", "mcp.json"))
	return err == nil
}
func (a *Adapter) SkillsDir(home string) string { return "" }
func (a *Adapter) AgentsDir(home string) string { return "" }
func (a *Adapter) MirrorRoot(home string) string {
	return filepath.Join(home, ".config", "mcp")
}
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "mcpuser")
}
