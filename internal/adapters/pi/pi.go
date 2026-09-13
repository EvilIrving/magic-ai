// Package pi is the Magic adapter for Pi Agent (config root ~/.pi/agent).
//
// Source: local-measure | ~/.pi/agent/{skills,agents,mcp.json,models.json}
//   on this Mac + pi-coding-agent docs | measured 2026-09-13 | verified
//
// Pi is Magic's primary harness. themes/ and AGENTS.md under this root are
// per-harness user data (AGENTS.md -> 配置库布局): they travel in
// packs/harness/pi/ so another machine restores the same Pi, and are NOT
// translated into another harness's theme format. Never: capture files back
// out of ~/.pi - the library is the only truth source (capture was deleted).
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
	return adapter.Caps{Skills: true, Agents: true, Themes: true, Models: true, MCP: true, Hooks: true}
}
func (a *Adapter) Detect(home string) bool {
	st, err := os.Stat(filepath.Join(home, ".pi", "agent"))
	return err == nil && st.IsDir()
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
