// Package qoder is the Magic adapter for Qoder CLI (and Qoder CN / 通义灵码).
// MCP and hooks both live in ~/.qoder/settings.json; skills in ~/.qoder/skills.

// Source: vendor-doc | docs.qoder.com/cli/settings-reference · /cli/mcp-servers | checked 2026-09-13 | verified
//   Hooks are Claude-shaped, so the Claude hook writer is reusable here.
package qoder

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

type Adapter struct {
	id   string
	name string
	dir  string
}

// New is the international Qoder adapter (~/.qoder).
func New() *Adapter {
	return &Adapter{id: "qoder", name: "Qoder", dir: ".qoder"}
}

// NewCN is the Qoder CN / 通义灵码 adapter (~/.qoder-cn).
func NewCN() *Adapter {
	return &Adapter{id: "qoder-cn", name: "Qoder CN", dir: ".qoder-cn"}
}

func (a *Adapter) ID() adapter.ID      { return adapter.ID(a.id) }
func (a *Adapter) Name() string        { return a.name }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, MCP: true, Hooks: true}
}
func (a *Adapter) root(home string) string { return filepath.Join(home, a.dir) }
func (a *Adapter) Detect(home string) bool {
	root := a.root(home)
	if st, err := os.Stat(root); err == nil && st.IsDir() {
		return true
	}
	_, err := os.Stat(filepath.Join(root, "settings.json"))
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
	return filepath.Join("packs", "harness", a.id)
}
