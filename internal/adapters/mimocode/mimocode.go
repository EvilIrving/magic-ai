// Package mimocode is the Magic adapter for MiMo Code (XiaomiMiMo/MiMo-Code),
// a fork of OpenCode. Config root is ~/.config/mimocode (MIMOCODE_HOME changes
// it to $MIMOCODE_HOME/config); the config file is mimocode.json[c] and uses the
// OpenCode dialect. Source: packages/shared/src/global.ts,
// packages/opencode/src/config/config.ts.

// Source: upstream | mimocode packages/shared/src/global.ts | checked 2026-09-13 | verified
//   OpenCode dialect; honours an env-var config root before falling back.
package mimocode

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "mimocode" }
func (a *Adapter) Name() string        { return "MiMo Code" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, MCP: true, Models: true}
}

// root is $MIMOCODE_HOME/config when MIMOCODE_HOME is set, else
// $XDG_CONFIG_HOME/mimocode (default ~/.config/mimocode).
func (a *Adapter) root(home string) string {
	if v := os.Getenv("MIMOCODE_HOME"); v != "" {
		return filepath.Join(filepath.Clean(v), "config")
	}
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(filepath.Clean(v), "mimocode")
	}
	return filepath.Join(home, ".config", "mimocode")
}

func (a *Adapter) Detect(home string) bool {
	st, err := os.Stat(a.root(home))
	if err == nil && st.IsDir() {
		return true
	}
	for _, f := range []string{"mimocode.json", "mimocode.jsonc"} {
		if _, err := os.Stat(filepath.Join(a.root(home), f)); err == nil {
			return true
		}
	}
	return false
}

func (a *Adapter) SkillsDir(home string) string  { return filepath.Join(a.root(home), "skills") }
func (a *Adapter) AgentsDir(home string) string  { return filepath.Join(a.root(home), "agents") }
func (a *Adapter) MirrorRoot(home string) string { return a.root(home) }
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "mimocode")
}
