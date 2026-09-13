// Package codex is the Magic adapter for Codex CLI (config root ~/.codex).
//
// Source: local-measure | ~/.codex/{config.toml,skills} | measured 2026-09-13 | verified
// Source: vendor-doc | developers.openai.com/codex/config-reference | checked 2026-09-13 | verified
//
// Codex keeps MCP and provider config in TOML, unlike the JSON crowd.
// Third-party models go through Codex's own model_catalog_json + env_key
// mechanism (docs/harness-matrix.md). Never: shell out to a curl|bash
// installer - the DeepSeek official-integration pattern is already in place
// (AGENTS.md -> 继续 registry 工作时 #5).
package codex

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "codex" }
func (a *Adapter) Name() string        { return "Codex" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, Agents: true, Themes: true, MCP: true, Hooks: true}
}
func (a *Adapter) Detect(home string) bool {
	st, err := os.Stat(filepath.Join(home, ".codex"))
	return err == nil && st.IsDir()
}
func (a *Adapter) SkillsDir(home string) string {
	return filepath.Join(home, ".codex", "skills")
}
func (a *Adapter) AgentsDir(home string) string {
	return filepath.Join(home, ".codex", "agents")
}
func (a *Adapter) MirrorRoot(home string) string {
	return filepath.Join(home, ".codex")
}
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "codex")
}
