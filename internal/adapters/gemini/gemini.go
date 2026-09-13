// Package gemini is the Magic adapter for Gemini CLI (config root ~/.gemini).
//
// Source: internal | docs/harness-profiles.md §Gemini/Antigravity | checked 2026-09-13 | verified
//
// Superseded: Gemini CLI was replaced by Antigravity CLI (decision D14). This
// package stays only as a transition target for users who still have ~/.gemini;
// new work goes to internal/adapters/antigravity.
// Never: manage .env - Gemini auth frequently lives there, out of scope.
package gemini

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

// Adapter mirrors ~/.gemini user settings (Gemini CLI).
// Auth often lives in .env / OAuth — Magic does not manage .env.
type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "gemini" }
func (a *Adapter) Name() string        { return "Gemini CLI" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{MCP: true, Themes: true, Hooks: true}
}
func (a *Adapter) Detect(home string) bool {
	root := filepath.Join(home, ".gemini")
	if st, err := os.Stat(root); err == nil && st.IsDir() {
		return true
	}
	_, err := os.Stat(filepath.Join(root, "settings.json"))
	return err == nil
}
func (a *Adapter) SkillsDir(home string) string { return "" }
func (a *Adapter) AgentsDir(home string) string { return "" }
func (a *Adapter) MirrorRoot(home string) string {
	return filepath.Join(home, ".gemini")
}
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "gemini")
}
