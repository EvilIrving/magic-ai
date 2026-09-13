// Package omp is the Magic adapter for Oh My Pi (can1357/oh-my-pi), a fork of
// Pi with the IDE wired in. Config root is ~/.omp (PI_CONFIG_DIR); the agent
// directory holds config.yml, mcp.json, skills/ and extensions/.
// Source: packages/utils/src/dirs.ts, docs/settings.md, docs/mcp-config.md.

// Source: upstream | omp packages/utils/src/dirs.ts, docs/settings.md, docs/mcp-config.md
//   | checked 2026-09-13 | verified
//   Oh My Pi reuses the ~/.<name>/agent layout, which is why root resolution
//   probes a sibling name instead of hardcoding one directory.
package omp

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "omp" }
func (a *Adapter) Name() string        { return "Oh My Pi" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, Agents: true, MCP: true, Hooks: true, Models: true}
}

// configRoot is ~/.omp, honoring PI_CONFIG_DIR.
func (a *Adapter) configRoot(home string) string {
	name := os.Getenv("PI_CONFIG_DIR")
	if name == "" {
		name = ".omp"
	}
	if filepath.IsAbs(name) {
		return name
	}
	return filepath.Join(home, name)
}

// agentDir is <config>/.omp/agent, honoring PI_CODING_AGENT_DIR.
func (a *Adapter) agentDir(home string) string {
	if v := os.Getenv("PI_CODING_AGENT_DIR"); v != "" {
		return filepath.Clean(v)
	}
	return filepath.Join(a.configRoot(home), "agent")
}

func (a *Adapter) Detect(home string) bool {
	if st, err := os.Stat(a.configRoot(home)); err == nil && st.IsDir() {
		return true
	}
	_, err := os.Stat(filepath.Join(a.agentDir(home), "config.yml"))
	return err == nil
}

func (a *Adapter) SkillsDir(home string) string {
	return filepath.Join(a.agentDir(home), "skills")
}
func (a *Adapter) AgentsDir(home string) string { return filepath.Join(a.agentDir(home), "agents") }
func (a *Adapter) MirrorRoot(home string) string {
	return a.agentDir(home)
}
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "omp")
}
