package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"magic/internal/adapter"
	"magic/internal/adapters/claude"
	"magic/internal/adapters/codex"
	"magic/internal/adapters/opencode"
	"magic/internal/adapters/pi"
	"magic/internal/adapters/workbuddy"
	"magic/internal/mcp"
)

// writeProviderLibrary writes the provider layer the way apply reads it: one
// resolvable provider with an anthropic and an openai-chat endpoint, and the key
// in the plaintext secret store.
func writeProviderLibrary(t *testing.T, magicHome, key string) {
	t.Helper()
	writeFile(t, filepath.Join(magicHome, "providers.yaml"), `active: example
providers:
  example:
    key_env: EXAMPLE_API_KEY
    pick: [example-model]
    endpoints:
      - id: anthropic
        base_url: https://api.example.com/anthropic
        protocol: anthropic-messages
      - id: openai-chat
        base_url: https://api.example.com/v1
        protocol: openai-chat-completions
`)
	writeFile(t, filepath.Join(magicHome, "secrets", "providers.json"), `{"EXAMPLE_API_KEY":"`+key+`"}`)
}

// applyLayers runs the layers a plan models, in apply's order, for one adapter.
func applyLayers(t *testing.T, a adapter.Adapter, opts ApplyOptions) {
	t.Helper()
	if _, err := ApplyAdapter(a, opts, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyProviders([]adapter.Adapter{a}, opts); err != nil {
		t.Fatal(err)
	}
	if _, err := mcp.Apply(mcp.Options{
		MagicHome: opts.MagicHome,
		UserHome:  opts.UserHome,
		Only:      opts.Only,
		Adapters:  []adapter.Adapter{a},
	}); err != nil {
		t.Fatal(err)
	}
}

// requirePlanMatchesDisk asserts the plan predicted the bytes apply left, for
// every change it reported.
func requirePlanMatchesDisk(t *testing.T, p Plan) {
	t.Helper()
	for _, ap := range p.Adapters {
		for _, c := range ap.Changes {
			data, err := os.ReadFile(c.Path)
			if err != nil {
				t.Fatalf("%s: read %s: %v", c.Rel, c.Path, err)
			}
			if got := FingerprintOf(data); got != c.After {
				t.Fatalf("%s: plan after=%s but apply wrote %s", c.Rel, c.After, got)
			}
		}
	}
}

// The provider layer is part of what apply writes: after apply the plan must
// report no drift, and it must report a difference once the projected env key is
// gone.
func TestPlanProviderClaudeMatchesApply(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	const key = "sk-live-abcdef123456"
	writeProviderLibrary(t, magicHome, key)
	opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome, Only: "claude"}
	adapters := []adapter.Adapter{claude.New()}

	p, err := BuildPlan(adapters, opts)
	if err != nil {
		t.Fatal(err)
	}
	c := findChange(t, p, "claude", "settings.json")
	if c.Action != ActionCreate || c.Mode != "0600" {
		t.Fatalf("change = %+v, want a 0600 create", c)
	}
	if !strings.Contains(c.Reason, "provider 投影 env") {
		t.Fatalf("reason = %q, want the provider layer", c.Reason)
	}
	planJSON, err := p.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(planJSON), key) {
		t.Fatalf("plan leaked the provider key:\n%s", planJSON)
	}

	applyLayers(t, claude.New(), opts)
	requirePlanMatchesDisk(t, p)

	// apply → verify must be clean: the same plan now finds nothing to write.
	p, err = BuildPlan(adapters, opts)
	if err != nil {
		t.Fatal(err)
	}
	if c := findChange(t, p, "claude", "settings.json"); c.Action != ActionUnchanged {
		t.Fatalf("after apply: action = %s, want unchanged (reason %q)", c.Action, c.Reason)
	}

	// A user who deletes one projected env key has drifted from the library.
	settings := filepath.Join(userHome, ".claude", "settings.json")
	var root map[string]any
	if err := json.Unmarshal([]byte(readFile(t, settings)), &root); err != nil {
		t.Fatal(err)
	}
	delete(root["env"].(map[string]any), "ANTHROPIC_AUTH_TOKEN")
	raw, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, settings, string(raw)+"\n")

	p, err = BuildPlan(adapters, opts)
	if err != nil {
		t.Fatal(err)
	}
	c = findChange(t, p, "claude", "settings.json")
	if c.Action != ActionUpdate {
		t.Fatalf("after deleting a provider env key: action = %s, want update", c.Action)
	}
	if !strings.Contains(c.Reason, "provider 投影 env") {
		t.Fatalf("reason = %q, want the provider layer named", c.Reason)
	}
	if hasKey(c.Keys, "env.ANTHROPIC_AUTH_TOKEN") == nil {
		t.Fatalf("keys = %+v, want the deleted env key reported", c.Keys)
	}
}

// The Pi provider layer writes models.json and the models.magic.json sidecar;
// both are part of the plan, and a dirty sidecar is drift.
func TestPlanProviderPiMatchesApply(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeProviderLibrary(t, magicHome, "sk-live-abcdef123456")
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "pi", "models.json"),
		`{"providers":{"user":{"api":"https://user.example/v1"}}}`)
	writeFile(t, filepath.Join(userHome, ".pi", "agent", "models.json"),
		`{"providers":{"user":{"api":"https://user.example/v1"}}}`)
	opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome, Only: "pi"}
	adapters := []adapter.Adapter{pi.New()}

	p, err := BuildPlan(adapters, opts)
	if err != nil {
		t.Fatal(err)
	}
	models := findChange(t, p, "pi", "models.json")
	if want := "镜像整文件覆盖 → provider 投影 models.json"; models.Reason != want {
		t.Fatalf("models.json reason = %q, want %q", models.Reason, want)
	}
	side := findChange(t, p, "pi", "models.magic.json")
	if side.Action != ActionCreate || !strings.Contains(side.Reason, "provider 投影 models.magic.json") {
		t.Fatalf("sidecar change = %+v", side)
	}

	applyLayers(t, pi.New(), opts)
	requirePlanMatchesDisk(t, p)

	// A dirty sidecar is drift even though no library file backs it.
	sidePath := filepath.Join(userHome, ".pi", "agent", "models.magic.json")
	writeFile(t, sidePath, `{"providers":{"someone-else":{"api":"x"}}}`)
	p, err = BuildPlan(adapters, opts)
	if err != nil {
		t.Fatal(err)
	}
	if c := findChange(t, p, "pi", "models.magic.json"); c.Action != ActionUpdate {
		t.Fatalf("dirty sidecar: action = %s, want update", c.Action)
	}
	if c := findChange(t, p, "pi", "models.json"); c.Action != ActionUnchanged {
		t.Fatalf("models.json: action = %s, want unchanged", c.Action)
	}
}

// The MCP projection is part of what apply writes: after apply the plan is
// clean, and a changed server block in ~/.claude.json is drift.
func TestPlanMCPMatchesApply(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(magicHome, "packs", "mcp", "mcp.yaml"), `mcp:
  docs:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-docs"]
`)
	opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome, Only: "claude"}
	adapters := []adapter.Adapter{claude.New()}

	p, err := BuildPlan(adapters, opts)
	if err != nil {
		t.Fatal(err)
	}
	c := findChange(t, p, "claude", "_claude.json")
	if c.Action != ActionCreate || !strings.Contains(c.Reason, "mcp 投影") {
		t.Fatalf("change = %+v, want an mcp create", c)
	}

	applyLayers(t, claude.New(), opts)
	requirePlanMatchesDisk(t, p)

	p, err = BuildPlan(adapters, opts)
	if err != nil {
		t.Fatal(err)
	}
	if c := findChange(t, p, "claude", "_claude.json"); c.Action != ActionUnchanged {
		t.Fatalf("after apply: action = %s, want unchanged", c.Action)
	}

	// Editing the projected server block is drift.
	claudeJSON := filepath.Join(userHome, ".claude.json")
	var root map[string]any
	if err := json.Unmarshal([]byte(readFile(t, claudeJSON)), &root); err != nil {
		t.Fatal(err)
	}
	servers := root["mcpServers"].(map[string]any)
	servers["docs"] = map[string]any{"command": "npx", "args": []any{"--edited"}}
	raw, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, claudeJSON, string(raw)+"\n")

	p, err = BuildPlan(adapters, opts)
	if err != nil {
		t.Fatal(err)
	}
	c = findChange(t, p, "claude", "_claude.json")
	if c.Action != ActionUpdate || !strings.Contains(c.Reason, "mcp 投影") {
		t.Fatalf("dirty server block: change = %+v, want an mcp update", c)
	}
}

// Every layer of apply can touch one file. The plan reports the chain and
// predicts the final bytes.
func TestPlanStackedLayersMatchApply(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeProviderLibrary(t, magicHome, "sk-live-abcdef123456")
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "claude", "settings.json"),
		`{"env":{"ANTHROPIC_BASE_URL":"https://library.example"}}`)
	writeFile(t, filepath.Join(userHome, ".claude", "settings.json"), `{"statusLine":{"type":"command"}}`)
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "claude", "claude.json.partial.json"),
		`{"mcpServers":{"library":{"command":"npx"}}}`)
	writeFile(t, filepath.Join(magicHome, "packs", "mcp", "mcp.yaml"), `mcp:
  docs:
    command: npx
`)
	opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome, Only: "claude"}
	adapters := []adapter.Adapter{claude.New()}

	p, err := BuildPlan(adapters, opts)
	if err != nil {
		t.Fatal(err)
	}
	settings := findChange(t, p, "claude", "settings.json")
	if want := "镜像 JSON 合并 → provider 投影 env"; settings.Reason != want {
		t.Fatalf("settings.json reason = %q, want %q", settings.Reason, want)
	}
	claudeJSON := findChange(t, p, "claude", "_claude.json")
	if want := "claude partial 深合并 → mcp 投影"; claudeJSON.Reason != want {
		t.Fatalf("~/.claude.json reason = %q, want %q", claudeJSON.Reason, want)
	}

	applyLayers(t, claude.New(), opts)
	requirePlanMatchesDisk(t, p)

	// Both projections landed in one file without losing the user's keys.
	body := readFile(t, filepath.Join(userHome, ".claude.json"))
	if !strings.Contains(body, `"library"`) || !strings.Contains(body, `"docs"`) {
		t.Fatalf("~/.claude.json lost a layer:\n%s", body)
	}
	settingsBody := readFile(t, filepath.Join(userHome, ".claude", "settings.json"))
	if !strings.Contains(settingsBody, "statusLine") || !strings.Contains(settingsBody, "ANTHROPIC_AUTH_TOKEN") {
		t.Fatalf("settings.json lost a layer:\n%s", settingsBody)
	}
}

// A TOML target written by both the mirror and the MCP projection reports one
// line diff of the final content.
func TestPlanCodexTOMLWithMCPMatchesApply(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "codex", "config.toml"),
		"model = \"kimi-k2\"\n")
	writeFile(t, filepath.Join(userHome, ".codex", "config.toml"),
		"model = \"gpt-5\"\napproval_policy = \"never\"\n")
	writeFile(t, filepath.Join(magicHome, "packs", "mcp", "mcp.yaml"), `mcp:
  docs:
    command: npx
`)
	opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome, Only: "codex"}
	adapters := []adapter.Adapter{codex.New()}

	p, err := BuildPlan(adapters, opts)
	if err != nil {
		t.Fatal(err)
	}
	c := findChange(t, p, "codex", "config.toml")
	if want := "镜像 TOML 合并 → mcp 投影"; c.Reason != want {
		t.Fatalf("reason = %q, want %q", c.Reason, want)
	}
	if c.Lines == nil || c.Lines.Added == 0 {
		t.Fatalf("lines = %+v, want the MCP section counted", c.Lines)
	}

	applyLayers(t, codex.New(), opts)
	requirePlanMatchesDisk(t, p)
	body := readFile(t, filepath.Join(userHome, ".codex", "config.toml"))
	if !strings.Contains(body, "[mcp_servers.docs]") {
		t.Fatalf("mcp section missing after apply:\n%s", body)
	}
}

// A library without packs/harness/codex still gets the codex MCP projection:
// most libraries hold no codex mirror tree, yet apply writes
// ~/.codex/config.toml, so the missing tree must not skip the whole adapter.
func TestPlanCodexMCPWithoutMirror(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	// No packs/harness/codex on purpose.
	writeFile(t, filepath.Join(magicHome, "packs", "mcp", "mcp.yaml"), `mcp:
  docs:
    command: npx
`)
	writeFile(t, filepath.Join(userHome, ".codex", "config.toml"), "model = \"gpt-5\"\n")
	opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome, Only: "codex"}

	p, err := BuildPlan([]adapter.Adapter{codex.New()}, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Adapters) != 1 || p.Adapters[0].Skipped != "" {
		t.Fatalf("adapters = %+v, want codex checked, not skipped", p.Adapters)
	}
	c := findChange(t, p, "codex", "config.toml")
	if c.Action != ActionUpdate {
		t.Fatalf("action = %s, want update (reason %q)", c.Action, c.Reason)
	}
	if want := "mcp 投影"; c.Reason != want {
		t.Fatalf("reason = %q, want %q", c.Reason, want)
	}
	if c.Lines == nil || c.Lines.Added == 0 {
		t.Fatalf("lines = %+v, want the MCP section counted", c.Lines)
	}

	applyLayers(t, codex.New(), opts)
	requirePlanMatchesDisk(t, p)
}

// noMirrorRoot is codex without a mirror root: the interface allows an adapter
// to declare none, and that must not cost it its MCP projection either.
type noMirrorRoot struct{ adapter.Adapter }

func (noMirrorRoot) MirrorRoot(string) string { return "" }

func TestPlanNoMirrorRootStillProjectsMCP(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(magicHome, "packs", "mcp", "mcp.yaml"), `mcp:
  docs:
    command: npx
`)
	writeFile(t, filepath.Join(userHome, ".codex", "config.toml"), "model = \"gpt-5\"\n")

	p, err := BuildPlan([]adapter.Adapter{noMirrorRoot{codex.New()}},
		ApplyOptions{MagicHome: magicHome, UserHome: userHome, Only: "codex"})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Adapters) != 1 || p.Adapters[0].Skipped != "" {
		t.Fatalf("adapters = %+v, want the MCP projection listed, not skipped", p.Adapters)
	}
	if c := findChange(t, p, "codex", ".codex/config.toml"); c.Reason != "mcp 投影" {
		t.Fatalf("reason = %q, want %q", c.Reason, "mcp 投影")
	}
}

// A skipped adapter says every fact that decided it: the missing mirror tree on
// its own would read as "add packs/harness/codex and apply manages codex", while
// apply never projects a provider into codex — only `magic provider use` writes
// it. A harness apply does project has nothing to add.
func TestPlanSkipReasonNamesProviderProjectionGap(t *testing.T) {
	p, err := BuildPlan([]adapter.Adapter{codex.New(), pi.New()}, ApplyOptions{MagicHome: t.TempDir(), UserHome: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	skipped := map[adapter.ID]string{}
	for _, ap := range p.Adapters {
		skipped[ap.Adapter] = ap.Skipped
	}
	for id, want := range map[adapter.ID]string{
		"codex": "配置库里没有镜像目录",
		"pi":    "配置库里没有镜像目录",
	} {
		if !strings.Contains(skipped[id], want) {
			t.Fatalf("%s skipped = %q, want it to name %q", id, skipped[id], want)
		}
	}
	if !strings.Contains(skipped["codex"], "magic provider use") {
		t.Fatalf("codex skipped = %q, want it to name the provider projection gap (magic provider use)", skipped["codex"])
	}
	if strings.Contains(skipped["pi"], "magic provider use") {
		t.Fatalf("pi skipped = %q, want no provider projection gap: apply does project pi", skipped["pi"])
	}
}

// The opencode provider switch is a projection like any other now: plan computes
// it through the same SwitchWriter.Files apply writes, and opencode.json carries
// both the provider block and the MCP servers.
func TestPlanOpencodeProviderMatchesApply(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeProviderLibrary(t, magicHome, "sk-live-abcdef123456")
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "opencode", "AGENTS.md"), "# OpenCode\n")
	// The provider switch and the MCP projection both write opencode.json.
	writeFile(t, filepath.Join(magicHome, "packs", "mcp", "mcp.yaml"), `mcp:
  docs:
    command: npx
`)
	if err := os.MkdirAll(filepath.Join(userHome, ".config", "opencode"), 0o755); err != nil {
		t.Fatal(err)
	}
	opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome, Only: "opencode"}
	adapters := []adapter.Adapter{opencode.New()}

	p, err := BuildPlan(adapters, opts)
	if err != nil {
		t.Fatal(err)
	}
	if c := findChange(t, p, "opencode", "AGENTS.md"); c.Action != ActionCreate {
		t.Fatalf("AGENTS.md action = %s, want create", c.Action)
	}
	c := findChange(t, p, "opencode", "opencode.json")
	if want := "provider 投影 opencode.json → mcp 投影"; c.Reason != want {
		t.Fatalf("opencode.json reason = %q, want %q", c.Reason, want)
	}
	if c.Action != ActionCreate {
		t.Fatalf("opencode.json action = %s, want create", c.Action)
	}

	applyLayers(t, opencode.New(), opts)
	requirePlanMatchesDisk(t, p)

	p, err = BuildPlan(adapters, opts)
	if err != nil {
		t.Fatal(err)
	}
	if c := findChange(t, p, "opencode", "opencode.json"); c.Action != ActionUnchanged {
		t.Fatalf("after apply: action = %s, want unchanged (reason %q)", c.Action, c.Reason)
	}
}

// A harness apply does not project keeps no provider layer, even though it has a
// SwitchWriter for `magic provider use`: codex config.toml stays the user's.
func TestPlanLeavesCodexProviderToUse(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeProviderLibrary(t, magicHome, "sk-live-abcdef123456")
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "codex", "config.toml"), "model = \"kimi-k2\"\n")
	writeFile(t, filepath.Join(userHome, ".codex", "config.toml"), "model = \"gpt-5\"\n")
	opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome, Only: "codex"}

	p, err := BuildPlan([]adapter.Adapter{codex.New()}, opts)
	if err != nil {
		t.Fatal(err)
	}
	c := findChange(t, p, "codex", "config.toml")
	if want := "镜像 TOML 合并"; c.Reason != want {
		t.Fatalf("reason = %q, want %q (apply must not project codex providers)", c.Reason, want)
	}
}

// hasKey returns the recorded op for a key, or nil when the plan does not list it.
func hasKey(keys []KeyChange, want string) *KeyOp {
	for _, k := range keys {
		if k.Key == want {
			op := k.Op
			return &op
		}
	}
	return nil
}

// Every harness apply projects a provider into is predicted the same way: the
// plan's after is the bytes apply leaves, and the same plan then finds nothing to
// write.
func TestPlanProviderLayerMatchesApplyPerHarness(t *testing.T) {
	cases := []struct {
		id      adapter.ID
		a       adapter.Adapter
		seed    string // user-home relative file written before apply
		content string
	}{
		{id: "claude", a: claude.New(), seed: ".claude/settings.json", content: `{"env":{"KEEP":"1"}}`},
		{id: "pi", a: pi.New(), seed: ".pi/agent/models.json", content: `{"providers":{"user":{"api":"https://user.example/v1"}}}`},
		{id: "opencode", a: opencode.New(), seed: ".config/opencode/opencode.json", content: `{"theme":"dark"}`},
		{id: "workbuddy", a: workbuddy.New(), seed: ".codebuddy/models.json", content: `{"models":[{"id":"user-local"}]}`},
	}
	for _, tc := range cases {
		t.Run(string(tc.id), func(t *testing.T) {
			magicHome, userHome := t.TempDir(), t.TempDir()
			writeProviderLibrary(t, magicHome, "sk-live-abcdef123456")
			writeFile(t, filepath.Join(userHome, filepath.FromSlash(tc.seed)), tc.content)
			opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome, Only: string(tc.id)}

			p, err := BuildPlan([]adapter.Adapter{tc.a}, opts)
			if err != nil {
				t.Fatal(err)
			}
			var providerChange *Change
			for i := range p.Adapters[0].Changes {
				if strings.Contains(p.Adapters[0].Changes[i].Reason, "provider 投影") {
					providerChange = &p.Adapters[0].Changes[i]
				}
			}
			if providerChange == nil {
				t.Fatalf("plan has no provider layer: %+v", p.Adapters[0].Changes)
			}
			applyLayers(t, tc.a, opts)
			requirePlanMatchesDisk(t, p)

			p, err = BuildPlan([]adapter.Adapter{tc.a}, opts)
			if err != nil {
				t.Fatal(err)
			}
			for _, c := range p.Adapters[0].Changes {
				if c.Action != ActionUnchanged {
					t.Fatalf("%s: action = %s after apply (reason %q)", c.Rel, c.Action, c.Reason)
				}
				if len(c.Keys) != 0 {
					t.Fatalf("%s: unchanged target reports key changes: %+v", c.Rel, c.Keys)
				}
			}
		})
	}
}
