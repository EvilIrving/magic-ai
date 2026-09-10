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
func (a *Adapter) CapturePolicy() adapter.CapturePolicy {
	return adapter.CapturePolicy{
		Roots: []adapter.RootSpec{{ID: "main", RelHome: ".config/mcp"}},
		Allow: []string{"mcp.json"},
		Deny:  []string{},
	}
}
func (a *Adapter) SkillsDir(home string) string { return "" }
func (a *Adapter) AgentsDir(home string) string { return "" }
func (a *Adapter) MirrorRoot(home string) string {
	return filepath.Join(home, ".config", "mcp")
}
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "mcpuser")
}
