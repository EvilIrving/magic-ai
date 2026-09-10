package core

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"magic/internal/catalog"
)

const Version = "1.0.0"

// ResolveHome returns MAGIC_HOME, accepting MAGIC_VAULT as alias.
// Default (unset): ~/.magic-ai — Magic working / config directory.
func ResolveHome() (string, error) {
	if v := os.Getenv("MAGIC_HOME"); v != "" {
		return filepath.Clean(v), nil
	}
	if v := os.Getenv("MAGIC_VAULT"); v != "" {
		return filepath.Clean(v), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".magic-ai"), nil
}

// Paths under the config library (配置库).
type Paths struct {
	Root          string
	MagicYAML     string
	ProvidersYAML string
	Overlay       string
	PacksSkills   string
	PacksAgents   string
	PacksHarness  string
	Secrets       string
	Backups       string
	README        string
}

func NewPaths(root string) Paths {
	return Paths{
		Root:          root,
		MagicYAML:     filepath.Join(root, "magic.yaml"),
		ProvidersYAML: filepath.Join(root, "providers.yaml"),
		Overlay:       filepath.Join(root, "capture.overlay.yaml"),
		PacksSkills:   filepath.Join(root, "packs", "skills"),
		PacksAgents:   filepath.Join(root, "packs", "agents"),
		PacksHarness:  filepath.Join(root, "packs", "harness"),
		Secrets:       filepath.Join(root, "secrets"),
		Backups:       filepath.Join(root, "backups"),
		README:        filepath.Join(root, "README.md"),
	}
}

// InitSkeleton creates the 配置库 directory layout.
func InitSkeleton(root string) error {
	p := NewPaths(root)
	dirs := []string{
		p.Root, p.PacksSkills, p.PacksAgents, p.PacksHarness, p.Secrets, p.Backups,
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	if _, err := os.Stat(p.MagicYAML); os.IsNotExist(err) {
		content := `# Magic config library (配置库)
version: "1.0"
adapters:
  enabled:
    - pi
    - grok
    - claude
    - codex
    - mcpuser
    - opencode
    - gemini
    - copilot
    - workbuddy
    - trae
# Note: "agents" means sub-agents under packs/agents/ — not an adapter id.
# Skills truth: packs/skills under this dir.
# Windsurf is not supported (dropped).
`
		if err := AtomicWriteFile(p.MagicYAML, []byte(content), 0o644); err != nil {
			return err
		}
	}
	if _, err := os.Stat(p.Overlay); os.IsNotExist(err) {
		content := `# capture.overlay.yaml — user capture rules (synced with 配置库)
# Precedence: user.exclude > user.include > builtin.deny > builtin.allow > unknown DENY
harnesses: {}
`
		if err := AtomicWriteFile(p.Overlay, []byte(content), 0o644); err != nil {
			return err
		}
	}
	if _, err := os.Stat(p.ProvidersYAML); os.IsNotExist(err) {
		if err := AtomicWriteFile(p.ProvidersYAML, []byte(catalog.ExampleProvidersYAML), 0o644); err != nil {
			return err
		}
	}
	agentsReadme := filepath.Join(p.PacksAgents, "README.md")
	if _, err := os.Stat(agentsReadme); os.IsNotExist(err) {
		content := `# Sub-agents (packs/agents)

Convention: one markdown file per sub-agent:

    packs/agents/<name>.md

Or a directory:

    packs/agents/<name>/AGENT.md

On ` + "`magic apply`" + `, files are copied into each Caps.Agents harness path:
  ~/.claude/agents/  ~/.codex/agents/  ~/.grok/agents/  ~/.pi/agent/agents/

"agents" = sub-agent definitions, not a harness adapter.
`
		if err := AtomicWriteFile(agentsReadme, []byte(content), 0o644); err != nil {
			return err
		}
	}
	if _, err := os.Stat(p.README); os.IsNotExist(err) {
		content := `# Magic 配置库 / Config Library

本目录是 magic CLI 管理的真相源（默认 ~/.magic-ai）。

- packs/skills — 全局 Skill 真相源；magic skills sync / magic apply 投影到各 harness
- packs/agents — sub-agent 定义（*.md）
- packs/harness/<id> — 各 harness 镜像树
- providers.yaml — provider / pick（无密钥明文）
- secrets/ — 捕获 reinject（<adapter>.json.enc）+ provider keys（providers.json.enc；口令加密）
- capture.overlay.yaml — capture 用户 include/exclude
- backups/ — apply 备份

覆盖路径：MAGIC_HOME（别名 MAGIC_VAULT）。
`
		if err := AtomicWriteFile(p.README, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// NewBackupDir creates backups/<utc>/ under MAGIC_HOME.
func NewBackupDir(magicHome string) (string, error) {
	stamp := time.Now().UTC().Format("20060102T150405Z")
	dir := filepath.Join(magicHome, "backups", stamp)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// UserHome resolves --home or $HOME.
func UserHome(flagHome string) (string, error) {
	if flagHome != "" {
		return filepath.Clean(flagHome), nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	return h, nil
}
