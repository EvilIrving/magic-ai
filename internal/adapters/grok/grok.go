package grok

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "grok" }
func (a *Adapter) Name() string        { return "Grok" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, Agents: true}
}
func (a *Adapter) Detect(home string) bool {
	st, err := os.Stat(filepath.Join(home, ".grok"))
	return err == nil && st.IsDir()
}
func (a *Adapter) SkillsDir(home string) string {
	return filepath.Join(home, ".grok", "skills")
}
func (a *Adapter) AgentsDir(home string) string {
	return filepath.Join(home, ".grok", "agents")
}
func (a *Adapter) MirrorRoot(home string) string {
	return filepath.Join(home, ".grok")
}
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "grok")
}
