package trae

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

// Adapter mirrors ~/.trae user assets (Trae IDE).
// Does not mirror Application Support/Trae/User (UI/login state).
type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "trae" }
func (a *Adapter) Name() string        { return "Trae" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, Agents: true, MCP: true}
}
func (a *Adapter) Detect(home string) bool {
	root := filepath.Join(home, ".trae")
	if st, err := os.Stat(root); err == nil && st.IsDir() {
		return true
	}
	// macOS app support presence alone is not enough for Magic scope
	return false
}
func (a *Adapter) SkillsDir(home string) string {
	return filepath.Join(home, ".trae", "skills")
}
func (a *Adapter) AgentsDir(home string) string {
	return filepath.Join(home, ".trae", "agents")
}
func (a *Adapter) MirrorRoot(home string) string {
	return filepath.Join(home, ".trae")
}
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "trae")
}
