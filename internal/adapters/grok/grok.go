// Package grok is the Magic adapter for Grok CLI (config root ~/.grok).
//
// Source: local-measure | ~/.grok/config.toml on this Mac | measured 2026-09-13 | verified
//
// No public format doc exists (xAI-internal CLI): every path and key here is
// evidence from a real install, not from memory. Re-measure before editing.
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
	return adapter.Caps{Skills: true, Agents: true, MCP: true, Hooks: true}
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
