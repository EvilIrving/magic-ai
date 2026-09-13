// Package antigravity is the Magic adapter for Antigravity CLI (`agy`), the
// successor to Gemini CLI (decision D14). MCP lives in a separate global file
// (~/.gemini/config/mcp_config.json); skills and custom config live under
// ~/.gemini/antigravity-cli/.

// Source: internal | docs/harness-profiles.md §6 Antigravity | checked 2026-09-13 | verified
//   Remote MCP key is `serverUrl`, not `url`/`httpUrl` - the one dialect that
//   deliberately differs from Gemini's. The global MCP file sits outside the
//   CLI config root, in ~/.gemini/config/mcp_config.json.
package antigravity

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "antigravity-cli" }
func (a *Adapter) Name() string        { return "Antigravity CLI" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, MCP: true, Hooks: true, Themes: true}
}
func (a *Adapter) root(home string) string {
	return filepath.Join(home, ".gemini", "antigravity-cli")
}
func (a *Adapter) Detect(home string) bool {
	if st, err := os.Stat(a.root(home)); err == nil && st.IsDir() {
		return true
	}
	_, err := os.Stat(filepath.Join(home, ".gemini", "config", "mcp_config.json"))
	return err == nil
}
func (a *Adapter) SkillsDir(home string) string {
	return filepath.Join(a.root(home), "skills")
}
func (a *Adapter) AgentsDir(home string) string { return "" }
func (a *Adapter) MirrorRoot(home string) string {
	return a.root(home)
}
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "antigravity-cli")
}
