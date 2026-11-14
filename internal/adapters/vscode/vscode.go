// Package vscode is the Magic adapter for the Visual Studio Code user profile.
//
// Source: vendor-doc | code.visualstudio.com/docs/configure/settings (user settings.json: %APPDATA%\Code\User, $HOME/Library/Application Support/Code/User, $HOME/.config/Code/User) | checked 2026-11-14 | verified
// Source: local-measure | ~/Library/Application Support/Code/User (settings.json, chatLanguageModels.json, keybindings.json, profiles/) | measured 2026-11-14 | verified
// Source: vendor-doc | code.visualstudio.com/docs/copilot/customization/agent-skills (personal skill roots are ~/.copilot/skills, ~/.claude/skills, ~/.agents/skills; VS Code owns no skills directory) | checked 2026-11-14 | verified
// Source: vendor-doc | code.visualstudio.com/docs/agents/reference/mcp-configuration (.vscode/mcp.json and the profile mcp.json are the deprecated Local-harness editor format with top-level `servers`; the Copilot harness reads ~/.copilot/mcp-config.json and workspace .mcp.json) | checked 2026-11-14 | verified
//
// Caps: none. Every customization surface VS Code reads belongs to another
// adapter (skill roots of copilot/claude/codex, MCP in ~/.copilot), so this
// adapter only mirrors packs/harness/vscode into the user profile directory —
// settings.json is key-merged, never replaced.
//
// Never: write chatLanguageModels.json. It exists (measured) but its shape is
// undocumented; it must not be generated or overwritten.
// Never: mirror ~/.vscode/extensions — installed extensions are artifacts, not
// configuration; the user-level config root is Code/User.
package vscode

import (
	"os"
	"path/filepath"
	"runtime"

	"magic/internal/adapter"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "vscode" }
func (a *Adapter) Name() string        { return "Visual Studio Code" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }

func (a *Adapter) Caps() adapter.Caps { return adapter.Caps{} }

// userDir is the platform's user profile dir (Code/User), which holds
// settings.json and the other per-user files.
func (a *Adapter) userDir(home string) string {
	switch runtime.GOOS {
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "Code", "User")
		}
		return filepath.Join(home, "AppData", "Roaming", "Code", "User")
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Code", "User")
	default:
		if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
			return filepath.Join(v, "Code", "User")
		}
		return filepath.Join(home, ".config", "Code", "User")
	}
}

func (a *Adapter) Detect(home string) bool {
	st, err := os.Stat(a.userDir(home))
	if err == nil && st.IsDir() {
		return true
	}
	_, err = os.Stat(filepath.Join(a.userDir(home), "settings.json"))
	return err == nil
}

func (a *Adapter) SkillsDir(home string) string  { return "" }
func (a *Adapter) AgentsDir(home string) string  { return "" }
func (a *Adapter) MirrorRoot(home string) string { return a.userDir(home) }
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "vscode")
}
