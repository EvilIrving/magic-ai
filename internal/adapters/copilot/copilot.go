package copilot

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

// Adapter mirrors ~/.copilot user-editable config (GitHub Copilot CLI).
// config.json holds auth/runtime state and is denied.
type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "copilot" }
func (a *Adapter) Name() string        { return "GitHub Copilot CLI" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, Agents: true, MCP: true}
}
func (a *Adapter) Detect(home string) bool {
	root := filepath.Join(home, ".copilot")
	if st, err := os.Stat(root); err == nil && st.IsDir() {
		return true
	}
	_, err := os.Stat(filepath.Join(root, "settings.json"))
	return err == nil
}
func (a *Adapter) CapturePolicy() adapter.CapturePolicy {
	return adapter.CapturePolicy{
		Roots: []adapter.RootSpec{{ID: "main", RelHome: ".copilot"}},
		Allow: []string{
			"settings.json",
			"copilot-instructions.md",
			"instructions/**",
			"skills/**",
			"agents/**",
			"hooks/**",
			"extensions/**",
			"mcp-config.json",
			"lsp-config.json",
		},
		Deny: []string{
			"config.json", // auth + managed state
			"logs/**",
			"session-state/**",
			"command-history-state/**",
			"session-store.db",
			"session-store.db-*",
			"mcp-oauth-config/**",
			"mcp-secrets/**",
			"ide/**",
			"installed-plugins/**",
			"plugin-data/**",
		},
	}
}
func (a *Adapter) SkillsDir(home string) string {
	return filepath.Join(home, ".copilot", "skills")
}
func (a *Adapter) AgentsDir(home string) string {
	return filepath.Join(home, ".copilot", "agents")
}
func (a *Adapter) MirrorRoot(home string) string {
	return filepath.Join(home, ".copilot")
}
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "copilot")
}
