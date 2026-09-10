package pi

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "pi" }
func (a *Adapter) Name() string        { return "Pi Agent" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, Agents: true, Themes: true, Models: true}
}
func (a *Adapter) Detect(home string) bool {
	st, err := os.Stat(filepath.Join(home, ".pi", "agent"))
	return err == nil && st.IsDir()
}
func (a *Adapter) CapturePolicy() adapter.CapturePolicy {
	return adapter.CapturePolicy{
		Roots: []adapter.RootSpec{{ID: "main", RelHome: ".pi/agent"}},
		Allow: []string{
			"settings.json", "auth.json", "models.json", "keybindings.json", "trust.json",
			"SYSTEM.md", "APPEND_SYSTEM.md", "AGENTS.md",
			"extensions/**", "skills/**", "prompts/**", "themes/**", "agents/**",
		},
		Deny: []string{
			"sessions/**", "bin/**", "npm/**", "git/**", "tmp/**",
			"models-store.json", "pi-debug.log",
		},
	}
}
func (a *Adapter) SkillsDir(home string) string {
	return filepath.Join(home, ".pi", "agent", "skills")
}
func (a *Adapter) AgentsDir(home string) string {
	return filepath.Join(home, ".pi", "agent", "agents")
}
func (a *Adapter) MirrorRoot(home string) string {
	return filepath.Join(home, ".pi", "agent")
}
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "pi")
}
