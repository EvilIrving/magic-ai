// Package opencode is the Magic adapter for OpenCode. Its config root is
// $XDG_CONFIG_HOME/opencode, or ~/.config/opencode when XDG_CONFIG_HOME is unset.
//
// Source: local-measure | ~/.config/opencode/opencode.json | measured 2026-09-13 | verified
// Source: vendor-doc | opencode.ai/docs/mcp-servers · opencode.ai/docs/providers ·
//   opencode.ai/config.json ($schema) | checked 2026-09-13 | verified
// Source: upstream | opencode source reads XDG_CONFIG_HOME for its global config root (5 call sites) | checked 2026-11-14 | verified
//
// Never: touch ~/.local/share/opencode/auth.json - OpenCode's own credential
// store, outside Magic scope (docs/harness-matrix.md).
package opencode

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
	"magic/internal/appdir"
)

// Adapter mirrors the OpenCode config root (global config + packs), which is
// $XDG_CONFIG_HOME/opencode, else ~/.config/opencode. Credentials live in
// ~/.local/share/opencode/auth.json and are intentionally outside this root (not
// managed by default).
type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "opencode" }
func (a *Adapter) Name() string        { return "OpenCode" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, Agents: true, Themes: true, Models: true, MCP: true, Hooks: true}
}
func (a *Adapter) root(home string) string {
	return filepath.Join(appdir.ConfigHome(home), "opencode")
}
func (a *Adapter) Detect(home string) bool {
	root := a.root(home)
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
	return filepath.Join(a.root(home), "skills")
}
func (a *Adapter) AgentsDir(home string) string {
	return filepath.Join(a.root(home), "agents")
}
func (a *Adapter) MirrorRoot(home string) string {
	return a.root(home)
}
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "opencode")
}
