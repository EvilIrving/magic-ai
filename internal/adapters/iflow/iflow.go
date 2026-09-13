// Package iflow is the Magic adapter for iFlow CLI.
// MCP: ~/.iflow/settings.json (key mcpServers). Skills: ~/.iflow/skills.

// Source: vendor-doc | github.com/iflow-ai/iflow-cli docs_en/configuration/settings.md
//   (~/.iflow/settings.json, mcpServers, baseUrl) | checked 2026-09-13 | verified
//   iFlow is low volume (npm ~161 weekly downloads, docs idle since 2026-03):
//   kept as a cheap target, not a v1 priority (docs/supported-harnesses.md).
package iflow

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "iflow-cli" }
func (a *Adapter) Name() string        { return "iFlow CLI" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, MCP: true, Hooks: true}
}
func (a *Adapter) root(home string) string { return filepath.Join(home, ".iflow") }
func (a *Adapter) Detect(home string) bool {
	if st, err := os.Stat(a.root(home)); err == nil && st.IsDir() {
		return true
	}
	_, err := os.Stat(filepath.Join(a.root(home), "settings.json"))
	return err == nil
}
func (a *Adapter) SkillsDir(home string) string  { return filepath.Join(a.root(home), "skills") }
func (a *Adapter) AgentsDir(home string) string  { return "" }
func (a *Adapter) MirrorRoot(home string) string { return a.root(home) }
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "iflow-cli")
}
