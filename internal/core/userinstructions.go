package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"magic/internal/adapter"
)

// SharedInstructionsRel is the optional shared instruction source. One file the
// user writes once is projected into every harness that has a verified
// user-level instruction file, each harness's own path under the user home.
const SharedInstructionsRel = "packs/AGENTS.md"

// userInstructionFiles names the file each harness reads as its user-level
// instructions, relative to the harness config root (Adapter.MirrorRoot) so an
// adapter that resolves its root from $XDG_CONFIG_HOME keeps working.
//
// Only verified paths appear here. The user-level instruction file is not one
// convention: every harness picked its own name, and a guessed path would write
// a file the harness never reads (agentsmd/agents.md issue #91 is the open
// upstream discussion). The date is when the path was checked against the source
// below; re-check before changing a value.
//
// Source: vendor-doc | code.claude.com/docs/en/memory (user instructions: ~/.claude/CLAUDE.md) | checked 2026-11-14 | verified
// Source: vendor-doc | developers.openai.com/codex/agent-configuration/agents-md (~/.codex/AGENTS.md) | checked 2026-11-14 | verified
// Source: vendor-doc | docs.factory.com/harness/agents-md (personal directories: ~/.factory/) | checked 2026-11-14 | verified
// Source: vendor-doc | grok 1.0.77 docs/user-guide/01-getting-started.md ($GROK_HOME/AGENTS.md) | checked 2026-11-14 | verified
// Source: vendor-doc | opencode docs "Instructions" + source config/plugin/instruction.ts (join(global.config, "AGENTS.md")) | checked 2026-11-14 | verified
// Source: vendor-doc | github.com/badlogic/pi-mono packages/coding-agent/docs/configuration.md (<agent-dir>/AGENTS.md) | checked 2026-11-14 | verified
//
// Not listed, because no primary source gives a user-level file: amp (no adapter
// in this registry, so nothing to project into), cursor (User Rules live in the
// app's settings, not in a file), copilot, gemini, qwen-code, kimi-code, trae,
// cline, kiro, zcode, iflow, kilo, zed.
var userInstructionFiles = map[adapter.ID]string{
	"claude":   "CLAUDE.md",
	"codex":    "AGENTS.md",
	"droid":    "AGENTS.md",
	"grok":     "AGENTS.md",
	"opencode": "AGENTS.md",
	"pi":       "AGENTS.md",
}

// sharedInstructionsPlaceholder is what `magic init` writes to
// SharedInstructionsRel. It carries no instructions and no instruction text, so
// hasInstructionText reports false and apply projects nothing until the user adds
// their own line.
const sharedInstructionsPlaceholder = `<!-- packs/AGENTS.md — 共享用户级指令（可选，本文件目前是占位）

写在这里的内容会原样写进每一家已安装 harness 的用户级指令文件：

  ~/.codex/AGENTS.md              Codex
  ~/.claude/CLAUDE.md             Claude Code
  ~/.factory/AGENTS.md            Droid
  ~/.pi/agent/AGENTS.md           Pi
  ~/.grok/AGENTS.md               Grok
  <config>/opencode/AGENTS.md     OpenCode

某一家要不一样，就放 packs/harness/<id>/AGENTS.md（Claude 是 CLAUDE.md），
它优先于本文件，apply 只写那一家。

本文件只有注释和空行时，apply 不写任何目标文件。
-->
`

// userInstructionTarget resolves the user-level instruction file a reads: the
// absolute path and the file name inside the harness config root. ok is false for
// a harness with no verified path, which is the gate that keeps apply from
// inventing one.
func userInstructionTarget(a adapter.Adapter, userHome string) (path, name string, ok bool) {
	name, ok = userInstructionFiles[a.ID()]
	if !ok {
		return "", "", false
	}
	root := a.MirrorRoot(userHome)
	if root == "" {
		return "", "", false
	}
	return filepath.Join(root, name), name, true
}

// harnessInstructionSource is the per-harness library file that wins over the
// shared one, packs/harness/<id>/<name>.
func harnessInstructionSource(a adapter.Adapter, magicHome, name string) string {
	return filepath.Join(magicHome, a.LibraryMirrorRel(), name)
}

// LoadSharedInstructions returns the bytes of the shared source and whether it
// holds any instruction.
//
// A source with no content after comments and blank lines is the init
// placeholder: the user has not written the shared layer yet. Writing it would
// either blank a harness's own file or paste Magic's comments into it, so this
// layer writes nothing. Only the shared source has that rule — a per-harness file
// in the library is the user's own file for that harness and is always written,
// comment-only or not.
func LoadSharedInstructions(magicHome string) ([]byte, bool, error) {
	data, err := os.ReadFile(filepath.Join(magicHome, filepath.FromSlash(SharedInstructionsRel)))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return data, hasInstructionText(data), nil
}

// hasInstructionText reports whether src holds anything other than blank lines
// and HTML comments. Markdown headings count as content: "# 项目约定" is the
// first line of a normal AGENTS.md, not a comment.
//
// A comment starts at the beginning of a line (after indentation), which is how
// the init placeholder is written. An HTML comment in the middle of a line does
// not hide the text before it, so a sentence that happens to mention "<!--" is
// still an instruction.
func hasInstructionText(src []byte) bool {
	inComment := false
	for _, line := range strings.Split(string(src), "\n") {
		text := strings.TrimSpace(line)
		for {
			if inComment {
				end := strings.Index(text, "-->")
				if end < 0 {
					text = ""
					break
				}
				text = strings.TrimSpace(text[end+3:])
				inComment = false
				continue
			}
			if !strings.HasPrefix(text, "<!--") {
				break
			}
			if end := strings.Index(text[4:], "-->"); end >= 0 {
				text = strings.TrimSpace(text[4+end+3:])
				continue
			}
			text = ""
			inComment = true
			break
		}
		if text != "" {
			return true
		}
	}
	return false
}

// applySharedInstructions writes packs/AGENTS.md to a's verified user-level
// instruction file.
//
// Three gates, the same ones the plan applies so apply and plan cannot disagree:
// a per-harness file in the library wins (the mirror walk already wrote it), the
// shared source has to hold an instruction, and the harness has to be installed
// (Detect) or be named by --only. The Detect gate is why apply does not invent
// ~/.factory/ on a machine that never had Droid: provider and MCP projections are
// gated the same way.
func applySharedInstructions(a adapter.Adapter, opts ApplyOptions, backupDir string) ([]string, error) {
	target, name, ok := userInstructionTarget(a, opts.UserHome)
	if !ok {
		return nil, nil
	}
	if st, err := os.Stat(harnessInstructionSource(a, opts.MagicHome, name)); err == nil && !st.IsDir() {
		return nil, nil
	}
	if !a.Detect(opts.UserHome) && opts.Only != string(a.ID()) {
		return nil, nil
	}
	data, hasContent, err := LoadSharedInstructions(opts.MagicHome)
	if err != nil || !hasContent {
		return nil, err
	}
	log := fmt.Sprintf("[%s] shared instructions → %s", a.ID(), target)
	if opts.DryRun {
		return []string{log}, nil
	}
	// The backup rel is the path under the user home, not the mirror rel: this
	// file comes from packs/AGENTS.md, so ".codex/AGENTS.md" tells it apart from a
	// mirrored packs/harness/codex/AGENTS.md, whose rel is "AGENTS.md".
	rel := homeRel(opts.UserHome, target)
	if err := mergeLibraryFile(backupDir, a.ID(), target, rel, data, 0o644); err != nil {
		return nil, err
	}
	return []string{log}, nil
}

// homeRel names a target by its path under the user home, the form plan and the
// backup manifest use for a file the library mirror root does not own.
func homeRel(userHome, path string) string {
	if rel, err := filepath.Rel(userHome, path); err == nil && !escapesDir(rel) {
		return rel
	}
	return filepath.Base(path)
}

// sharedInstructionLayer stages the shared instructions for one adapter: the
// same gates applySharedInstructions uses, merged onto the plan's simulated file
// so a target an earlier layer already wrote keeps that layer's bytes.
func sharedInstructionLayer(a adapter.Adapter, opts ApplyOptions, files *planFiles) error {
	target, name, ok := userInstructionTarget(a, opts.UserHome)
	if !ok {
		return nil
	}
	if st, err := os.Stat(harnessInstructionSource(a, opts.MagicHome, name)); err == nil && !st.IsDir() {
		return nil
	}
	if !a.Detect(opts.UserHome) && opts.Only != string(a.ID()) {
		return nil
	}
	data, hasContent, err := LoadSharedInstructions(opts.MagicHome)
	if err != nil || !hasContent {
		return err
	}
	f, err := files.touch(target, 0o644)
	if err != nil {
		return err
	}
	// Name the file by its path under the user home. It sits inside the harness
	// mirror root, but it is not a mirrored file: the plan and the backup manifest
	// spell it ".codex/AGENTS.md".
	if len(f.layers) == 0 {
		f.rel = homeRel(opts.UserHome, target)
	}
	f.cur, f.mode = data, 0o644
	f.layers = append(f.layers, SharedInstructionsLayerLabel)
	return nil
}

// SharedInstructionsLayerLabel is the plan reason of a file written from the
// shared instruction source. Change.Reason joins layer labels with " → ".
const SharedInstructionsLayerLabel = "共享 AGENTS.md 投影"
