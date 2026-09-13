// Package openclaw is the Magic adapter for OpenClaw (openclaw/openclaw), a
// local personal AI assistant with agent harnesses. Config lives in
// ~/.openclaw/openclaw.json (JSON5-ish); skills in ~/.openclaw/skills
// (OPENCLAW_STATE_DIR/skills takes precedence). Sources: docs/start/setup.md,
// docs/nodes/mcp-and-skills.md.

// Source: internal | docs/supported-harnesses.md §第三轮调研 (OpenClaw) | checked 2026-09-13 | stale?
//   openclaw.json is JSON5-ish: see LenientJSON in internal/mcp/target.go.
//   Never: rewrite that file with a strict json.Marshal - it would drop user
//   comments and break OpenClaw's own parse.
package openclaw

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "openclaw" }
func (a *Adapter) Name() string        { return "OpenClaw" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, MCP: true}
}

// root is ~/.openclaw, honoring OPENCLAW_HOME.
func (a *Adapter) root(home string) string {
	if v := os.Getenv("OPENCLAW_HOME"); v != "" {
		return filepath.Clean(v)
	}
	return filepath.Join(home, ".openclaw")
}

// stateDir is where skills live: OPENCLAW_STATE_DIR wins, else the config root.
func (a *Adapter) stateDir(home string) string {
	if v := os.Getenv("OPENCLAW_STATE_DIR"); v != "" {
		return filepath.Clean(v)
	}
	return a.root(home)
}

func (a *Adapter) Detect(home string) bool {
	if st, err := os.Stat(a.root(home)); err == nil && st.IsDir() {
		return true
	}
	_, err := os.Stat(filepath.Join(a.root(home), "openclaw.json"))
	return err == nil
}

func (a *Adapter) SkillsDir(home string) string  { return filepath.Join(a.stateDir(home), "skills") }
func (a *Adapter) AgentsDir(home string) string  { return "" }
func (a *Adapter) MirrorRoot(home string) string { return a.root(home) }
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "openclaw")
}
