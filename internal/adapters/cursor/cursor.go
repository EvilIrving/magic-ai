// Package cursor is the Magic adapter for Cursor (IDE + CLI).
// Product decision D20: Cursor was promoted from skills-only to a full adapter
// (usage rank #4). Cursor model access is account/UI driven, so there is no
// model SwitchWriter here.
//
// Source: local-measure | ~/.cursor/{mcp.json,skills} | measured 2026-09-13 | verified
// Source: vendor-doc | cursor.com/en/docs/mcp | checked 2026-09-13 | verified
package cursor

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "cursor" }
func (a *Adapter) Name() string        { return "Cursor" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, MCP: true, Hooks: true}
}
func (a *Adapter) Detect(home string) bool {
	root := filepath.Join(home, ".cursor")
	if st, err := os.Stat(root); err == nil && st.IsDir() {
		return true
	}
	_, err := os.Stat(filepath.Join(root, "mcp.json"))
	return err == nil
}
func (a *Adapter) SkillsDir(home string) string {
	return filepath.Join(home, ".cursor", "skills")
}
func (a *Adapter) AgentsDir(home string) string { return "" }
func (a *Adapter) MirrorRoot(home string) string {
	return filepath.Join(home, ".cursor")
}
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "cursor")
}
