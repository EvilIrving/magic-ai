// Package dsh is the Magic adapter for DeepSeek Harness
// (deepseek-ai/deepseek-harness, npm @deepseek-ai/dsh).
//
// Source: upstream | @deepseek-ai/dsh-home-paths lib/types/index.d.ts (DSH_HOME_DIR_NAME=".dsh", DSH_HOME_ENV="DSH_HOME"; one root for all user data) | checked 2026-11-14 | verified
// Source: upstream | @deepseek-ai/dsh-skill-filesystem README "Roots and priority" (rank 400 user-dsh = <dshHome>/skills) | checked 2026-11-14 | verified
// Source: upstream | @deepseek-ai/dsh-settings README (a home settings.yaml is imported once, then renamed settings.yaml.imported) | checked 2026-11-14 | verified
// Source: local-measure | ~/.dsh (settings.yaml.imported, profiles/web/cordis.patch.yml, storages/, .credentials.yaml) with dsh 0.2.0-rc.2 | measured 2026-11-14 | verified
//
// Never: write ~/.dsh/settings.yaml as configuration. dsh 0.2 imports that
// legacy file once and renames it; live configuration is the profile's
// cordis.patch.yml, including MCP plugin entries (Magic does not project those,
// docs/decisions.md D24).
// Never: touch ~/.dsh/.credentials.yaml or ~/.dsh/sessions.
package dsh

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "dsh" }
func (a *Adapter) Name() string        { return "DeepSeek Harness" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }

// Caps: skills only. Agents are declared as cordis presets inside a profile
// (dsh-agent-preset), and models/MCP/hooks live in cordis.patch.yml — none of
// them is a flat file Magic can project.
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true}
}

// root is $DSH_HOME, else ~/.dsh.
func (a *Adapter) root(home string) string {
	if v := os.Getenv("DSH_HOME"); v != "" {
		return filepath.Clean(v)
	}
	return filepath.Join(home, ".dsh")
}

func (a *Adapter) Detect(home string) bool {
	st, err := os.Stat(a.root(home))
	return err == nil && st.IsDir()
}

func (a *Adapter) SkillsDir(home string) string  { return filepath.Join(a.root(home), "skills") }
func (a *Adapter) AgentsDir(home string) string  { return "" }
func (a *Adapter) MirrorRoot(home string) string { return a.root(home) }
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "dsh")
}
