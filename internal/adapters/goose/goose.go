// Package goose is the Magic adapter for goose (aaif-goose/goose, formerly Block).
//
// Source: upstream | crates/goose/src/config/paths.rs (etcetera choose_app_strategy: XDG config dir on Linux and macOS, Windows strategy on Windows; GOOSE_PATH_ROOT=<abs>) | checked 2026-11-14 | verified
// Source: vendor-doc | documentation/docs/guides/config-files.md (~/.config/goose/config.yaml; %APPDATA%\Block\goose\config\config.yaml) | checked 2026-11-14 | verified
// Source: upstream | crates/goose/src/skills/mod.rs all_skill_dirs_with_config (<config dir>/skills is a writable global skill root) | checked 2026-11-14 | verified
// Source: vendor-doc | documentation/docs/guides/context-engineering/custom-agents.md (~/.agents/agents/<name>.md) | checked 2026-11-14 | verified
//
// Never: write config.yaml's provider block or secrets.yaml. goose keeps API
// keys in secrets.yaml / the keyring; Magic has no goose SwitchWriter, so the
// mirror only carries files a user put in packs/harness/goose.
package goose

import (
	"os"
	"path/filepath"
	"runtime"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "goose" }
func (a *Adapter) Name() string        { return "goose" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }

// Caps: skills and sub-agents. MCP in goose is the `extensions` block of
// config.yaml, and models are its `providers` block; neither has a Magic
// projection today.
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Skills: true, Agents: true}
}

// root is the config dir: $GOOSE_PATH_ROOT/config when that root is absolute,
// else the platform default (%APPDATA%\Block\goose\config on Windows,
// $XDG_CONFIG_HOME/goose on Linux and macOS, ~/.config/goose when XDG is unset).
func (a *Adapter) root(home string) string {
	if v := os.Getenv("GOOSE_PATH_ROOT"); v != "" && filepath.IsAbs(v) {
		return filepath.Join(v, "config")
	}
	if runtime.GOOS == "windows" {
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "Block", "goose", "config")
		}
		return filepath.Join(home, "AppData", "Roaming", "Block", "goose", "config")
	}
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, "goose")
	}
	return filepath.Join(home, ".config", "goose")
}

func (a *Adapter) Detect(home string) bool {
	st, err := os.Stat(a.root(home))
	if err == nil && st.IsDir() {
		return true
	}
	_, err = os.Stat(filepath.Join(a.root(home), "config.yaml"))
	return err == nil
}

func (a *Adapter) SkillsDir(home string) string {
	return filepath.Join(a.root(home), "skills")
}

// AgentsDir is the shared global agent dir, not a goose-config child.
func (a *Adapter) AgentsDir(home string) string {
	return filepath.Join(home, ".agents", "agents")
}

func (a *Adapter) MirrorRoot(home string) string { return a.root(home) }
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "goose")
}
