// Package vibe is the Magic adapter for Mistral Vibe (mistralai/mistral-vibe,
// the `vibe` CLI).
//
// Source: upstream | vibe/utils/vibe_home.py ($VIBE_HOME, else ~/.vibe) | checked 2026-11-14 | verified
// Source: vendor-doc | docs/reference/configuration.md (~/.vibe/config.toml is the user TOML layer and the default write target) | checked 2026-11-14 | verified
// Source: upstream | vibe/core/config/harness_files/_paths.py (GLOBAL_SKILLS_DIR = $VIBE_HOME/skills; GLOBAL_AGENTS_DIR = $VIBE_HOME/agents) | checked 2026-11-14 | verified
// Source: local-measure | ~/.vibe/skills | measured 2026-11-14 | verified
//
// Never: write $VIBE_HOME/.env or $VIBE_HOME/vibehistory — credential and
// prompt-history files, not configuration.
package vibe

import (
	"os"
	"path/filepath"
	"strings"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "vibe" }
func (a *Adapter) Name() string        { return "Mistral Vibe" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }

// Caps: skills only. Agent profiles in $VIBE_HOME/agents are TOML files, not
// the markdown packs/agents convention, and models/MCP live in config.toml with
// no Magic projection.
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true}
}

// root is $VIBE_HOME (a leading ~ is expanded, as upstream does), else ~/.vibe.
func (a *Adapter) root(home string) string {
	v := strings.TrimSpace(os.Getenv("VIBE_HOME"))
	if v == "" {
		return filepath.Join(home, ".vibe")
	}
	if rest, ok := strings.CutPrefix(v, "~"); ok && (rest == "" || rest[0] == '/' || rest[0] == '\\') {
		return filepath.Join(home, strings.TrimLeft(rest, `/\`))
	}
	return filepath.Clean(v)
}

func (a *Adapter) Detect(home string) bool {
	st, err := os.Stat(a.root(home))
	if err == nil && st.IsDir() {
		return true
	}
	_, err = os.Stat(filepath.Join(a.root(home), "config.toml"))
	return err == nil
}

func (a *Adapter) SkillsDir(home string) string  { return filepath.Join(a.root(home), "skills") }
func (a *Adapter) AgentsDir(home string) string  { return "" }
func (a *Adapter) MirrorRoot(home string) string { return a.root(home) }
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "vibe")
}
