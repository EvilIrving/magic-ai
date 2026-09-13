// Package qwencode is the Magic adapter for Qwen Code CLI.
// MCP: ~/.qwen/settings.json (key mcpServers). Skills: ~/.qwen/skills.

// Source: internal | docs/supported-harnesses.md §调研来源 (Qwen Code) | checked 2026-09-13 | stale?
//   Gemini CLI lineage (17 hook events, settings.json mcpServers); confirm the
//   directory against QwenLM/qwen-code docs before editing.
package qwencode

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "qwen-code" }
func (a *Adapter) Name() string        { return "Qwen Code" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, Agents: true, MCP: true, Hooks: true}
}
func (a *Adapter) root(home string) string {
	if v := os.Getenv("QWEN_CODE_HOME"); v != "" {
		return filepath.Clean(v)
	}
	return filepath.Join(home, ".qwen")
}
func (a *Adapter) Detect(home string) bool {
	if st, err := os.Stat(a.root(home)); err == nil && st.IsDir() {
		return true
	}
	_, err := os.Stat(filepath.Join(a.root(home), "settings.json"))
	return err == nil
}
func (a *Adapter) SkillsDir(home string) string  { return filepath.Join(a.root(home), "skills") }
func (a *Adapter) AgentsDir(home string) string  { return filepath.Join(a.root(home), "agents") }
func (a *Adapter) MirrorRoot(home string) string { return a.root(home) }
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "qwen-code")
}
