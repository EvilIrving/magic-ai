package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"magic/internal/adapter"
	"magic/internal/adapters/claude"
	"magic/internal/adapters/codex"
	"magic/internal/adapters/pi"
	"magic/internal/adapters/stub"
	"magic/internal/adapters/trae"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func findChange(t *testing.T, p Plan, id adapter.ID, rel string) Change {
	t.Helper()
	for _, ap := range p.Adapters {
		if ap.Adapter != id {
			continue
		}
		for _, c := range ap.Changes {
			if c.Rel == rel {
				return c
			}
		}
	}
	t.Fatalf("no change for %s/%s in plan", id, rel)
	return Change{}
}

func TestPlanCreateOnMissingTarget(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "pi", "AGENTS.md"), "# Pi rules\n")
	// The harness is installed (Detect finds ~/.pi/agent) but the target file is
	// missing: that is the create the plan reports.
	if err := os.MkdirAll(filepath.Join(userHome, ".pi", "agent"), 0o755); err != nil {
		t.Fatal(err)
	}

	p, err := BuildPlan([]adapter.Adapter{pi.New()}, ApplyOptions{MagicHome: magicHome, UserHome: userHome})
	if err != nil {
		t.Fatal(err)
	}
	c := findChange(t, p, "pi", "AGENTS.md")
	if c.Action != ActionCreate {
		t.Fatalf("action = %s, want create", c.Action)
	}
	if c.Before != "" {
		t.Fatalf("before must be empty, got %s", c.Before)
	}
	if want := filepath.Join(userHome, ".pi", "agent", "AGENTS.md"); c.Path != want {
		t.Fatalf("path = %s, want %s", c.Path, want)
	}
	if c.Mode != "0644" {
		t.Fatalf("mode = %s, want 0644", c.Mode)
	}
	if c.Backup {
		t.Fatal("a new file has nothing to back up")
	}
	if p.Summary.Create != 1 || p.Summary.Files != 1 {
		t.Fatalf("summary = %+v", p.Summary)
	}
}

func TestPlanJSONMergeKeepsTargetOnlyKeys(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "claude", "settings.json"),
		`{"env":{"ANTHROPIC_AUTH_TOKEN":"sk-live","ANTHROPIC_BASE_URL":"https://relay"},"hooks":{"Stop":[]}}`)
	writeFile(t, filepath.Join(userHome, ".claude", "settings.json"),
		`{"env":{"FOO":"bar"},"theme":"dark"}`)

	p, err := BuildPlan([]adapter.Adapter{claude.New()}, ApplyOptions{MagicHome: magicHome, UserHome: userHome})
	if err != nil {
		t.Fatal(err)
	}
	c := findChange(t, p, "claude", "settings.json")
	if c.Action != ActionUpdate {
		t.Fatalf("action = %s, want update", c.Action)
	}
	if !c.Backup {
		t.Fatal("an existing target must be backed up before apply")
	}
	got := map[string]KeyOp{}
	for _, kc := range c.Keys {
		got[kc.Key] = kc.Op
	}
	want := map[string]KeyOp{
		"env.ANTHROPIC_AUTH_TOKEN": KeyAdd,
		"env.ANTHROPIC_BASE_URL":   KeyAdd,
		"hooks.Stop":               KeyAdd,
		"theme":                    KeyKeep,
	}
	for k, op := range want {
		if got[k] != op {
			t.Errorf("key %s = %q, want %q (all: %v)", k, got[k], op, got)
		}
	}

	// The plan must predict the bytes apply writes, otherwise it lies.
	if _, err := ApplyAdapter(claude.New(), ApplyOptions{MagicHome: magicHome, UserHome: userHome}, ""); err != nil {
		t.Fatal(err)
	}
	written := readFile(t, filepath.Join(userHome, ".claude", "settings.json"))
	if FingerprintOf([]byte(written)) != c.After {
		t.Fatalf("plan after=%s but apply wrote %s", c.After, FingerprintOf([]byte(written)))
	}
	if !strings.Contains(written, `"theme": "dark"`) {
		t.Fatalf("target-only key was dropped: %s", written)
	}
}

func TestPlanUnchangedWhenBytesMatch(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	body := `{"model":"kimi"}`
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "pi", "models.json"), body)
	writeFile(t, filepath.Join(userHome, ".pi", "agent", "models.json"), body)

	p, err := BuildPlan([]adapter.Adapter{pi.New()}, ApplyOptions{MagicHome: magicHome, UserHome: userHome})
	if err != nil {
		t.Fatal(err)
	}
	c := findChange(t, p, "pi", "models.json")
	if c.Action != ActionUnchanged {
		t.Fatalf("action = %s, want unchanged", c.Action)
	}
	if c.Before != c.After {
		t.Fatalf("before %s != after %s", c.Before, c.After)
	}
}

func TestPlanTOMLReportsLineChanges(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "codex", "config.toml"),
		"model = \"kimi\"\n")
	writeFile(t, filepath.Join(userHome, ".codex", "config.toml"),
		"model = \"gpt\"\napproval_policy = \"never\"\n")

	p, err := BuildPlan([]adapter.Adapter{codex.New()}, ApplyOptions{MagicHome: magicHome, UserHome: userHome})
	if err != nil {
		t.Fatal(err)
	}
	c := findChange(t, p, "codex", "config.toml")
	if c.Action != ActionUpdate {
		t.Fatalf("action = %s, want update", c.Action)
	}
	if c.Lines == nil {
		t.Fatal("config.toml must report line changes")
	}
	if c.Lines.Added != 1 || c.Lines.Removed != 1 {
		t.Fatalf("lines = %+v, want added=1 removed=1 (model line replaced)", *c.Lines)
	}
}

func TestPlanNeverCarriesSecretText(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	const secret = "sk-live-abcdef123456"
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "claude", "settings.json"),
		`{"env":{"ANTHROPIC_AUTH_TOKEN":"`+secret+`"}}`)
	if err := os.MkdirAll(filepath.Join(userHome, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}

	p, err := BuildPlan([]adapter.Adapter{claude.New()}, ApplyOptions{MagicHome: magicHome, UserHome: userHome})
	if err != nil {
		t.Fatal(err)
	}
	out, err := p.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), secret) {
		t.Fatalf("plan leaked a secret:\n%s", out)
	}
	if strings.Contains(string(out), "sk-live") {
		t.Fatalf("plan leaked a secret prefix:\n%s", out)
	}
}

func TestPlanClaudePartialMatchesApply(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "claude", "claude.json.partial.json"),
		`{"mcpServers":{"docs":{"command":"npx"}}}`)
	writeFile(t, filepath.Join(userHome, ".claude.json"),
		`{"mcpServers":{"mine":{"command":"x"}},"projects":{}}`)
	opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome}
	p, err := BuildPlan([]adapter.Adapter{claude.New()}, opts)
	if err != nil {
		t.Fatal(err)
	}
	c := findChange(t, p, "claude", "_claude.json")
	if c.Action != ActionUpdate || c.Mode != "0600" {
		t.Fatalf("change = %+v", c)
	}

	if _, err := applyClaudePartial(opts, ""); err != nil {
		t.Fatal(err)
	}
	written := readFile(t, filepath.Join(userHome, ".claude.json"))
	if FingerprintOf([]byte(written)) != c.After {
		t.Fatalf("plan after=%s but applyClaudePartial wrote %s", c.After, FingerprintOf([]byte(written)))
	}
	// Deep merge: the plan's keys and the bytes must keep the user's server.
	if !strings.Contains(written, `"mine"`) || !strings.Contains(written, `"docs"`) {
		t.Fatalf("partial merge lost a side:\n%s", written)
	}
}

// A file named like a structured JSON config can still be unparsable. plan must
// report the fallback projectBytes took, not the merge its name suggests.
func TestPlanReasonFollowsActualStrategy(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "claude", "settings.json"),
		`{"env":{"A":"1"}}`)
	writeFile(t, filepath.Join(userHome, ".claude", "settings.json"), `{not json`)

	p, err := BuildPlan([]adapter.Adapter{claude.New()}, ApplyOptions{MagicHome: magicHome, UserHome: userHome})
	if err != nil {
		t.Fatal(err)
	}
	c := findChange(t, p, "claude", "settings.json")
	if c.Action != ActionUpdate {
		t.Fatalf("action = %s, want update", c.Action)
	}
	if strings.Contains(c.Reason, "深合并") {
		t.Fatalf("reason claims a merge that did not happen: %q", c.Reason)
	}
	if !strings.Contains(c.Reason, "整文件覆盖") {
		t.Fatalf("reason = %q, want the whole-file fallback", c.Reason)
	}
	if len(c.Keys) != 0 {
		t.Fatalf("keys = %+v, want none for a fallback", c.Keys)
	}
	// apply must write the same bytes the plan predicted.
	if _, err := ApplyAdapter(claude.New(), ApplyOptions{MagicHome: magicHome, UserHome: userHome}, ""); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(userHome, ".claude", "settings.json")); FingerprintOf([]byte(got)) != c.After {
		t.Fatalf("plan after=%s but apply wrote %s", c.After, FingerprintOf([]byte(got)))
	}
}

// D26: plan uses the same Detect gate as apply. An uninstalled harness is
// skipped with the reason the user can act on, and its library files are not
// reported as missing targets, so they do not count as drift. --only names the
// harness explicitly, the way apply pre-installs one.
func TestPlanSkipsUninstalledHarness(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "trae", "foo.txt"), "library\n")

	p, err := BuildPlan([]adapter.Adapter{trae.New()}, ApplyOptions{MagicHome: magicHome, UserHome: userHome})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Adapters) != 1 || p.Adapters[0].Skipped != "未安装" {
		t.Fatalf("adapters = %+v, want trae skipped as 未安装", p.Adapters)
	}
	if p.Summary.Files != 0 {
		t.Fatalf("summary = %+v, want no drift for an uninstalled harness", p.Summary)
	}

	p, err = BuildPlan([]adapter.Adapter{trae.New()},
		ApplyOptions{MagicHome: magicHome, UserHome: userHome, Only: "trae"})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Adapters) != 1 || p.Adapters[0].Skipped != "" {
		t.Fatalf("adapters = %+v, want --only to bypass the gate", p.Adapters)
	}
	c := findChange(t, p, "trae", "foo.txt")
	if c.Action != ActionCreate {
		t.Fatalf("action = %s, want create", c.Action)
	}
}

func TestPlanSkipsUnimplementedAdapter(t *testing.T) {
	p, err := BuildPlan([]adapter.Adapter{stub.New("nope", "Nope")}, ApplyOptions{MagicHome: t.TempDir(), UserHome: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Adapters) != 1 || p.Adapters[0].Skipped == "" {
		t.Fatalf("adapters = %+v", p.Adapters)
	}
	if p.Summary.Files != 0 {
		t.Fatalf("summary = %+v", p.Summary)
	}
}
