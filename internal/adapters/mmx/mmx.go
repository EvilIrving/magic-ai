// Package mmx is the Magic adapter for MiniMax Code (MiniMax-AI/minimax-code,
// the `mcode` CLI and the desktop app, which share one runtime).
//
// Source: upstream | packages/config/src/data-dir.ts (NEW_DATA_DIR_BASENAME ".minimax", LEGACY_DATA_DIR_BASENAME ".mavis", profile suffix "-<profile>") | checked 2026-11-14 | verified
// Source: upstream | packages/config/src/config.ts (MINIMAX_DATA_DIR / MAVIS_DATA_DIR; getConfigPath = <dataDir>/config.yaml; skillsDir = <dataDir>/skills) | checked 2026-11-14 | verified
// Source: upstream | packages/local-runtime/src/skills/roots.ts (user-global skill root = <dataDir>/skills) | checked 2026-11-14 | verified
//
// Magic deviation: the harness id is `mmx`, but the config root is ~/.minimax.
// MiniMax-AI/cli (npm `mmx-cli`, binary `mmx`) is a different product that
// keeps ~/.mmx/config.json and has no skills/MCP surface — do not repoint this
// adapter at ~/.mmx (harness-rollout-plan.md §2.2 recorded that row before the
// minimax-code repository was available).
//
// Never: touch <dataDir>/sessions, <dataDir>/.credentials*, or rewrite
// config.yaml — MiniMax Code rewrites it itself.
package mmx

import (
	"os"
	"path/filepath"
	"strings"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "mmx" }
func (a *Adapter) Name() string        { return "MiniMax Code" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }

// Caps: skills only. Agent personas live at <dataDir>/agents/<name>/agent.md
// (journaled folders), not flat markdown files, and MCP/model configuration is
// in config.yaml with no Magic projection.
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true}
}

// root is $MINIMAX_DATA_DIR, else $MAVIS_DATA_DIR, else ~/.minimax.
func (a *Adapter) root(home string) string {
	for _, env := range []string{"MINIMAX_DATA_DIR", "MAVIS_DATA_DIR"} {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			return filepath.Clean(v)
		}
	}
	return filepath.Join(home, ".minimax")
}

func (a *Adapter) Detect(home string) bool {
	st, err := os.Stat(a.root(home))
	if err == nil && st.IsDir() {
		return true
	}
	_, err = os.Stat(filepath.Join(a.root(home), "config.yaml"))
	return err == nil
}

func (a *Adapter) SkillsDir(home string) string  { return filepath.Join(a.root(home), "skills") }
func (a *Adapter) AgentsDir(home string) string  { return "" }
func (a *Adapter) MirrorRoot(home string) string { return a.root(home) }
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "mmx")
}
