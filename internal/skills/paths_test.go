package skills

import "testing"

func TestKnownAgentLongTail(t *testing.T) {
	for _, id := range []string{"qoder", "kimi-code-cli", "antigravity-cli", "qwen-code", "iflow-cli", "zed"} {
		if !KnownAgent(id) {
			t.Errorf("%s should be known", id)
		}
	}
	for _, id := range []string{"windsurf", "roo", "", "nope"} {
		if KnownAgent(id) {
			t.Errorf("%s should not be known", id)
		}
	}
}

func TestGlobalTargetDirLongTail(t *testing.T) {
	cases := map[string]string{
		"qoder":           ".qoder/skills",
		"kimi-code-cli":   ".agents/skills",
		"antigravity-cli": ".gemini/antigravity-cli/skills",
		"pi":              ".pi/agent/skills",
		"opencode":        ".config/opencode/skills",
	}
	for id, want := range cases {
		got, err := GlobalTargetDir(id, "/home/u")
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		if got != "/home/u/"+want {
			t.Errorf("%s: got %q want %q", id, got, "/home/u/"+want)
		}
	}
	if _, err := GlobalTargetDir("windsurf", "/home/u"); err == nil {
		t.Error("windsurf must not resolve (D13)")
	}
}

func TestParseAgentsAll(t *testing.T) {
	got, err := ParseAgents([]string{"all"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) <= len(DefaultAgents) {
		t.Fatalf("all should expand past defaults: %d", len(got))
	}
	if !KnownAgent(got[len(got)-1]) {
		t.Fatalf("unexpected agent %q", got[len(got)-1])
	}
	// No duplicates.
	seen := map[string]bool{}
	for _, a := range got {
		if seen[a] {
			t.Fatalf("duplicate %q", a)
		}
		seen[a] = true
	}
}

func TestParseAgentsExplicit(t *testing.T) {
	got, err := ParseAgents([]string{"qoder,pi"})
	if err != nil || len(got) != 2 || got[0] != "qoder" || got[1] != "pi" {
		t.Fatalf("%v %v", got, err)
	}
	if _, err := ParseAgents([]string{"windsurf"}); err == nil {
		t.Fatal("windsurf should be rejected")
	}
}
