package claude

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "claude" }
func (a *Adapter) Name() string        { return "Claude Code" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, Agents: true, Themes: true}
}
func (a *Adapter) Detect(home string) bool {
	if st, err := os.Stat(filepath.Join(home, ".claude")); err == nil && st.IsDir() {
		return true
	}
	_, err := os.Stat(filepath.Join(home, ".claude.json"))
	return err == nil
}
func (a *Adapter) SkillsDir(home string) string {
	return filepath.Join(home, ".claude", "skills")
}
func (a *Adapter) AgentsDir(home string) string {
	return filepath.Join(home, ".claude", "agents")
}
func (a *Adapter) MirrorRoot(home string) string {
	return filepath.Join(home, ".claude")
}
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "claude")
}
