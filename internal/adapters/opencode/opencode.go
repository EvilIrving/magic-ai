// Package opencode is the Magic adapter for OpenCode (root ~/.config/opencode).
//
// Source: local-measure | ~/.config/opencode/opencode.json | measured 2026-09-13 | verified
// Source: vendor-doc | opencode.ai/docs/mcp-servers · opencode.ai/docs/providers ·
//   opencode.ai/config.json ($schema) | checked 2026-09-13 | verified
//
// Never: touch ~/.local/share/opencode/auth.json - OpenCode's own credential
// store, outside Magic scope (docs/harness-matrix.md).
package opencode

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

// Adapter mirrors ~/.config/opencode (global OpenCode config + packs).
// Credentials live in ~/.local/share/opencode/auth.json and are intentionally
// outside this root (not managed by default).
type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "opencode" }
func (a *Adapter) Name() string        { return "OpenCode" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, Agents: true, Themes: true, Models: true, MCP: true, Hooks: true}
}
func (a *Adapter) Detect(home string) bool {
	root := filepath.Join(home, ".config", "opencode")
	if st, err := os.Stat(root); err == nil && st.IsDir() {
		return true
	}
	for _, name := range []string{"opencode.json", "opencode.jsonc"} {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			return true
		}
	}
	return false
}
func (a *Adapter) SkillsDir(home string) string {
	return filepath.Join(home, ".config", "opencode", "skills")
}
func (a *Adapter) AgentsDir(home string) string {
	return filepath.Join(home, ".config", "opencode", "agents")
}
func (a *Adapter) MirrorRoot(home string) string {
	return filepath.Join(home, ".config", "opencode")
}
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "opencode")
}
