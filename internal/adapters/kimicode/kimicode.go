// Package kimicode is the Magic adapter for Kimi Code CLI.
// Config root ~/.kimi-code (or $KIMI_CODE_HOME); MCP in ~/.kimi-code/mcp.json;
// skills are read from ~/.agents/skills (shared long-tail path).

// Source: vendor-doc | kimi.com/code/docs/en/kimi-code-cli/customization/mcp.html
//   | checked 2026-09-13 | verified
//   Agent files are Claude/OpenCode-compatible; skills go to the shared
//   ~/.agents/skills dir while agents use ~/.kimi-code/agents.
package kimicode

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "kimi-code" }
func (a *Adapter) Name() string        { return "Kimi Code CLI" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, Agents: true, MCP: true, Hooks: true}
}
func (a *Adapter) root(home string) string {
	if v := os.Getenv("KIMI_CODE_HOME"); v != "" {
		return filepath.Clean(v)
	}
	return filepath.Join(home, ".kimi-code")
}
func (a *Adapter) Detect(home string) bool {
	root := a.root(home)
	if st, err := os.Stat(root); err == nil && st.IsDir() {
		return true
	}
	_, err := os.Stat(filepath.Join(root, "mcp.json"))
	return err == nil
}
func (a *Adapter) SkillsDir(home string) string {
	return filepath.Join(home, ".agents", "skills")
}
func (a *Adapter) AgentsDir(home string) string {
	return filepath.Join(a.root(home), "agents")
}
func (a *Adapter) MirrorRoot(home string) string {
	return a.root(home)
}
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "kimi-code")
}
