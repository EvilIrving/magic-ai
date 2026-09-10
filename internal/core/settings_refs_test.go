package core_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"magic/internal/adapters/claude"
	"magic/internal/adapters/codex"
	"magic/internal/core"
	"magic/internal/policy"
)

func TestExtractLocalRefs_bashStatusline(t *testing.T) {
	home := "/Users/actor"
	root := filepath.Join(home, ".claude")
	settings := []byte(`{
  "statusLine": { "type": "command", "command": "bash ~/.claude/statusline.sh" }
}`)
	got := core.ExtractLocalRefs(settings, home, root)
	want := filepath.Join(home, ".claude", "statusline.sh")
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %#v want [%s]", got, want)
	}
}

func TestExtractLocalRefs_renamedScript(t *testing.T) {
	home := "/home/cain"
	root := filepath.Join(home, ".claude")
	settings := []byte(`{
  "statusLine": { "type": "command", "command": "bash ~/.claude/my-status.sh" }
}`)
	got := core.ExtractLocalRefs(settings, home, root)
	want := filepath.Join(home, ".claude", "my-status.sh")
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %#v want [%s]", got, want)
	}
}

func TestExtractLocalRefs_absoluteAndQuoted(t *testing.T) {
	home := "/Users/x"
	root := filepath.Join(home, ".claude")
	abs := filepath.Join(home, ".claude", "foo.sh")
	settings, err := json.Marshal(map[string]any{
		"a": map[string]any{"command": `bash "` + abs + `"`},
		"b": map[string]any{"script": filepath.Join(home, ".claude", "bar.sh")},
		"c": map[string]any{"path": "$HOME/.claude/baz.sh"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := core.ExtractLocalRefs(settings, home, root)
	want := map[string]bool{
		filepath.Join(home, ".claude", "foo.sh"): true,
		filepath.Join(home, ".claude", "bar.sh"): true,
		filepath.Join(home, ".claude", "baz.sh"): true,
	}
	if len(got) != 3 {
		t.Fatalf("got %#v", got)
	}
	for _, p := range got {
		if !want[p] {
			t.Fatalf("unexpected %s in %#v", p, got)
		}
	}
}

func TestExtractLocalRefs_outsideIgnored(t *testing.T) {
	home := "/Users/x"
	root := filepath.Join(home, ".claude")
	settings, _ := json.Marshal(map[string]any{
		"statusLine": map[string]any{"command": "bash /tmp/evil.sh"},
		"other":      map[string]any{"command": "bash ~/Documents/x.sh"},
		"url":        map[string]any{"path": "https://example.com/x.sh"},
		"bin":        map[string]any{"command": "bash"},
	})
	got := core.ExtractLocalRefs(settings, home, root)
	if len(got) != 0 {
		t.Fatalf("expected no refs, got %#v", got)
	}
}

func TestExtractLocalRefs_anyHarnessRoot(t *testing.T) {
	home := "/Users/x"
	root := filepath.Join(home, ".pi", "agent")
	settings := []byte(`{"statusLine":{"command":"bash ~/.pi/agent/status.sh"}}`)
	got := core.ExtractLocalRefs(settings, home, root)
	want := filepath.Join(home, ".pi", "agent", "status.sh")
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %#v want [%s]", got, want)
	}
}

func TestFoldHomePathsInSettings(t *testing.T) {
	home := "/Users/actor"
	in := []byte(`{
  "statusLine": {
    "type": "command",
    "command": "bash /Users/actor/.claude/my-bar.sh"
  },
  "note": "$HOME/.claude/other.sh"
}`)
	out := core.FoldHomePathsInSettings(in, home, ".claude")
	s := string(out)
	if strings.Contains(s, "/Users/actor/.claude") {
		t.Fatalf("absolute home not folded: %s", s)
	}
	if !strings.Contains(s, `bash ~/.claude/my-bar.sh`) {
		t.Fatalf("expected folded command, got %s", s)
	}
	if !strings.Contains(s, `~/.claude/other.sh`) {
		t.Fatalf("expected $HOME folded, got %s", s)
	}
}

func TestFoldHomePathsInSettings_piAgent(t *testing.T) {
	home := "/home/cain"
	in := []byte(`{"cmd":"bash /home/cain/.pi/agent/x.sh"}`)
	out := core.FoldHomePathsInSettings(in, home, ".pi/agent")
	s := string(out)
	if strings.Contains(s, "/home/cain/.pi/agent") {
		t.Fatalf("not folded: %s", s)
	}
	if !strings.Contains(s, `~/.pi/agent/x.sh`) {
		t.Fatalf("expected tilde fold, got %s", s)
	}
}

func TestExtractConfigLocalRefs_codexToml(t *testing.T) {
	home := "/Users/x"
	root := filepath.Join(home, ".codex")
	toml := []byte(`
model = "gpt"
notify = ["bash", "~/.codex/notify.sh"]
theme_path = "/Users/x/.codex/themes/dark.json"
hook = 'https://example.com/x'
bin = "notify-send"
outside = "~/Documents/x.sh"
`)
	got := core.ExtractConfigLocalRefs(toml, home, root)
	want := map[string]bool{
		filepath.Join(home, ".codex", "notify.sh"):         true,
		filepath.Join(home, ".codex", "themes", "dark.json"): true,
	}
	if len(got) != 2 {
		t.Fatalf("got %#v", got)
	}
	for _, p := range got {
		if !want[p] {
			t.Fatalf("unexpected %s in %#v", p, got)
		}
	}
}

func TestTriage_settingsRefsAllow(t *testing.T) {
	tmp := t.TempDir()
	userHome := filepath.Join(tmp, "home")
	claudeRoot := filepath.Join(userHome, ".claude")
	if err := os.MkdirAll(claudeRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(claudeRoot, "my-bar.sh")
	if err := os.WriteFile(script, []byte("#!/bin/bash\necho ok\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	settings := []byte(`{
  "statusLine": { "type": "command", "command": "bash ~/.claude/my-bar.sh" }
}`)
	if err := os.WriteFile(filepath.Join(claudeRoot, "settings.json"), settings, 0o644); err != nil {
		t.Fatal(err)
	}

	a := claude.New()
	report := core.Triage(userHome, a, nil)
	found := false
	for _, v := range report.Allow {
		if v.Rel == "my-bar.sh" && v.Reason == "settings.ref" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected my-bar.sh settings.ref in allow: %+v", report.Allow)
	}

	// Missing script → no error, not in allow
	missingHome := filepath.Join(tmp, "home2")
	_ = os.MkdirAll(filepath.Join(missingHome, ".claude"), 0o755)
	_ = os.WriteFile(filepath.Join(missingHome, ".claude", "settings.json"), []byte(`{
  "statusLine": { "command": "bash ~/.claude/gone.sh" }
}`), 0o644)
	report2 := core.Triage(missingHome, a, nil)
	for _, v := range report2.Allow {
		if v.Rel == "gone.sh" {
			t.Fatalf("missing script should not appear in allow: %+v", report2.Allow)
		}
	}
}

func TestTriage_settingsLocalJsonRefs(t *testing.T) {
	tmp := t.TempDir()
	userHome := filepath.Join(tmp, "home")
	claudeRoot := filepath.Join(userHome, ".claude")
	_ = os.MkdirAll(claudeRoot, 0o755)
	_ = os.WriteFile(filepath.Join(claudeRoot, "local-hook.sh"), []byte("#!/bin/bash\n"), 0o755)
	_ = os.WriteFile(filepath.Join(claudeRoot, "settings.local.json"), []byte(`{
  "hooks": { "command": "bash ~/.claude/local-hook.sh" }
}`), 0o644)

	report := core.Triage(userHome, claude.New(), nil)
	found := false
	for _, v := range report.Allow {
		if v.Rel == "local-hook.sh" && v.Reason == "settings.ref" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected local-hook.sh settings.ref: %+v", report.Allow)
	}
}

func TestTriage_codexConfigTomlRefs(t *testing.T) {
	tmp := t.TempDir()
	userHome := filepath.Join(tmp, "home")
	codexRoot := filepath.Join(userHome, ".codex")
	_ = os.MkdirAll(filepath.Join(codexRoot, "themes"), 0o755)
	_ = os.WriteFile(filepath.Join(codexRoot, "notify.sh"), []byte("#!/bin/bash\n"), 0o755)
	_ = os.WriteFile(filepath.Join(codexRoot, "themes", "custom.json"), []byte(`{}`), 0o644)
	toml := "notify = \"~/.codex/notify.sh\"\ntheme = \"" + filepath.ToSlash(filepath.Join(userHome, ".codex", "themes", "custom.json")) + "\"\n"
	_ = os.WriteFile(filepath.Join(codexRoot, "config.toml"), []byte(toml), 0o644)
	_ = os.WriteFile(filepath.Join(codexRoot, "hooks.json"), []byte(`{}`), 0o644)

	report := core.Triage(userHome, codex.New(), nil)
	foundNotify, foundTheme, foundHooks := false, false, false
	for _, v := range report.Allow {
		switch {
		case v.Rel == "notify.sh" && v.Reason == "config.ref":
			foundNotify = true
		case v.Rel == "themes/custom.json" && (v.Reason == "config.ref" || strings.HasPrefix(v.Reason, "builtin")):
			// themes/** is also allow-listed; either reason is fine if present
			if v.Reason == "config.ref" {
				foundTheme = true
			} else {
				foundTheme = true // already allow-listed via themes/**
			}
		case v.Rel == "hooks.json":
			foundHooks = true
		}
	}
	if !foundNotify {
		t.Fatalf("expected notify.sh config.ref: %+v", report.Allow)
	}
	if !foundTheme {
		t.Fatalf("expected themes/custom.json allow: %+v", report.Allow)
	}
	if !foundHooks {
		t.Fatalf("expected hooks.json allow: %+v", report.Allow)
	}
}

func TestCapture_foldsSettingsAndCopiesRef(t *testing.T) {
	tmp := t.TempDir()
	userHome := filepath.Join(tmp, "home")
	magicHome := filepath.Join(tmp, "lib")
	claudeRoot := filepath.Join(userHome, ".claude")
	_ = os.MkdirAll(claudeRoot, 0o755)
	absCmd := "bash " + filepath.Join(userHome, ".claude", "my-bar.sh")
	settings, _ := json.MarshalIndent(map[string]any{
		"statusLine": map[string]any{"type": "command", "command": absCmd},
	}, "", "  ")
	_ = os.WriteFile(filepath.Join(claudeRoot, "settings.json"), append(settings, '\n'), 0o644)
	_ = os.WriteFile(filepath.Join(claudeRoot, "my-bar.sh"), []byte("#!/bin/bash\n"), 0o755)

	if err := core.InitSkeleton(magicHome); err != nil {
		t.Fatal(err)
	}
	a := claude.New()
	ov := &policy.Overlay{Harnesses: map[string]policy.OverlayHarness{}}
	report, copied, err := core.CaptureAdapter(a, ov, core.CaptureOptions{
		MagicHome: magicHome, UserHome: userHome, DryRun: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	foundRef := false
	for _, v := range report.Allow {
		if v.Rel == "my-bar.sh" && v.Reason == "settings.ref" {
			foundRef = true
		}
	}
	if !foundRef {
		t.Fatalf("allow missing settings.ref: %+v", report.Allow)
	}
	libSettings := filepath.Join(magicHome, "packs", "harness", "claude", "settings.json")
	raw, err := os.ReadFile(libSettings)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), userHome) {
		t.Fatalf("library settings still has absolute home: %s", raw)
	}
	if !strings.Contains(string(raw), "~/.claude/my-bar.sh") {
		t.Fatalf("expected folded path, got %s", raw)
	}
	libScript := filepath.Join(magicHome, "packs", "harness", "claude", "my-bar.sh")
	if _, err := os.Stat(libScript); err != nil {
		t.Fatalf("expected script copied among %v: %v", copied, err)
	}
}
