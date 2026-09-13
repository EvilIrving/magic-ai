// Package dimagent is the Magic adapter for DimAgent (npm `dimagent`).
// State home is ~/.dim/v2 (DIMCODE_HOME); MCP servers live in
// <home>/mcp.json under mcpServers. Source: dimagent.com/en/docs/config.

// Source: vendor-doc | dimagent.com/en/docs/config | checked 2026-09-13 | stale?
//   The skills directory is unconfirmed, so SkillsDir returns empty rather
//   than inventing a path.
package dimagent

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "dimagent" }
func (a *Adapter) Name() string        { return "DimAgent" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	// Skills/hooks/plugins are configured through the TUI; no documented
	// on-disk skills directory yet, so only MCP is projected.
	return adapter.Caps{MCP: true}
}

func (a *Adapter) root(home string) string {
	if v := os.Getenv("DIMCODE_HOME"); v != "" {
		return filepath.Clean(v)
	}
	return filepath.Join(home, ".dim", "v2")
}

func (a *Adapter) Detect(home string) bool {
	if st, err := os.Stat(a.root(home)); err == nil && st.IsDir() {
		return true
	}
	_, err := os.Stat(filepath.Join(home, ".dim"))
	return err == nil
}

func (a *Adapter) SkillsDir(home string) string  { return "" }
func (a *Adapter) AgentsDir(home string) string  { return "" }
func (a *Adapter) MirrorRoot(home string) string { return a.root(home) }
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "dimagent")
}
