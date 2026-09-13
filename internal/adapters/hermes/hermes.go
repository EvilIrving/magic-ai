// Package hermes is the Magic adapter for Hermes Agent (NousResearch/hermes-agent).
// Home is ~/.hermes (HERMES_HOME); mcp_servers live in config.yaml and skills
// are under ~/.hermes/skills. Source: hermes_cli/config.py,
// website/docs/reference/mcp-config-reference.md.

// Source: upstream | hermes_cli/config.py + website/docs/reference/mcp-config-reference.md
//   | checked 2026-09-13 | verified
//   YAML target (FormatYAML, snake_case mcp_servers).
package hermes

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "hermes" }
func (a *Adapter) Name() string        { return "Hermes Agent" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	// Sub-agents are spawned at runtime (no declarable agent directory).
	return adapter.Caps{Skills: true, MCP: true, Models: true}
}

func (a *Adapter) root(home string) string {
	if v := os.Getenv("HERMES_HOME"); v != "" {
		return filepath.Clean(v)
	}
	return filepath.Join(home, ".hermes")
}

func (a *Adapter) Detect(home string) bool {
	st, err := os.Stat(a.root(home))
	if err == nil && st.IsDir() {
		return true
	}
	_, err = os.Stat(filepath.Join(a.root(home), "config.yaml"))
	return err == nil
}

func (a *Adapter) SkillsDir(home string) string  { return filepath.Join(a.root(home), "skills") }
func (a *Adapter) AgentsDir(home string) string  { return "" }
func (a *Adapter) MirrorRoot(home string) string { return a.root(home) }
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "hermes")
}
