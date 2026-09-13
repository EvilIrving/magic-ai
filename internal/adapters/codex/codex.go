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
	return adapter.Caps{Skills: true, Agents: true, Themes: true}
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
