// Package craft is the Magic adapter for Craft Agents
// (craft-ai-agents/craft-agents-oss).
//
// Source: upstream | packages/shared/src/config/paths.ts (CONFIG_DIR = $CRAFT_CONFIG_DIR, else ~/.craft-agent) | checked 2026-11-14 | verified
// Source: upstream | README §Configuration + packages/shared/src/workspaces/storage.ts (skills are per workspace: <workspaceRoot>/skills) | checked 2026-11-14 | verified
// Source: upstream | packages/server-core/src/handlers/rpc/skills.ts (loadAllSkills(workspaceRoot, workingDirectory)) | checked 2026-11-14 | verified
//
// Caps: none. Every syncable surface is workspace-scoped — skills and sources
// (MCP/API) hang off a workspace root, and config.json holds the workspace
// registry plus LLM connections. There is no single user-level skills,
// sub-agent or mcpServers file to project, so this adapter only mirrors
// packs/harness/craft custom files into ~/.craft-agent.
//
// Never: touch credentials.enc — Craft's AES-256-GCM credential store.
package craft

import (
	"os"
	"path/filepath"
	"strings"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "craft" }
func (a *Adapter) Name() string        { return "Craft Agents" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }

func (a *Adapter) Caps() adapter.Caps { return adapter.Caps{} }

// root is $CRAFT_CONFIG_DIR, else ~/.craft-agent.
func (a *Adapter) root(home string) string {
	if v := strings.TrimSpace(os.Getenv("CRAFT_CONFIG_DIR")); v != "" {
		return filepath.Clean(v)
	}
	return filepath.Join(home, ".craft-agent")
}

func (a *Adapter) Detect(home string) bool {
	st, err := os.Stat(a.root(home))
	if err == nil && st.IsDir() {
		return true
	}
	_, err = os.Stat(filepath.Join(a.root(home), "config.json"))
	return err == nil
}

// SkillsDir and AgentsDir are empty: Craft has no user-level directory for
// either. Returning a guessed path would write files Craft never reads.
func (a *Adapter) SkillsDir(home string) string  { return "" }
func (a *Adapter) AgentsDir(home string) string  { return "" }
func (a *Adapter) MirrorRoot(home string) string { return a.root(home) }
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "craft")
}
