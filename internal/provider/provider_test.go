package provider

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"magic/internal/catalog"
	"magic/internal/secrets"
)

func TestResolveCompanyIsActiveAlias(t *testing.T) {
	f := &catalog.File{
		Active: "shu",
		Providers: map[string]catalog.Provider{
			"shu": {KeyEnv: "SHU_API_KEY", Endpoints: []catalog.Endpoint{
				{ID: "openai-responses", BaseURL: "https://api.shu.cool/v1", Protocol: catalog.ProtocolOpenAIResponses},
			}},
		},
	}
	rt, err := ResolveUseTarget(f, "company")
	if err != nil {
		t.Fatal(err)
	}
	if rt.Mode != ModeProvider || rt.Name != "shu" {
		t.Fatalf("%+v", rt)
	}
	_, err = ResolveUseTarget(&catalog.File{Providers: map[string]catalog.Provider{"x": {}}}, "company")
	if err == nil {
		t.Fatal("expected error when active unset")
	}
	off, err := ResolveUseTarget(f, "official")
	if err != nil || off.Mode != ModeOfficial {
		t.Fatalf("%+v %v", off, err)
	}
}

func TestCodexTOMLCompanyFromLibraryName(t *testing.T) {
	p := catalog.Provider{KeyEnv: "SHU_API_KEY"}
	ep := catalog.Endpoint{BaseURL: "https://api.shu.cool/v1", Protocol: catalog.ProtocolOpenAIResponses}
	out, err := SetCodexManagedConfig("", "myrelay", "myrelay", p, ep, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `model_provider = "myrelay"`) {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(out, "[model_providers.myrelay]") {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(out, `base_url = "https://api.shu.cool/v1"`) {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(out, `wire_api = "responses"`) {
		t.Fatalf("%s", out)
	}
	// official
	out2, err := SetCodexManagedConfig(out, "openai", "", catalog.Provider{}, catalog.Endpoint{}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if GetCodexProvider(out2) != "openai" {
		t.Fatalf("%s", out2)
	}
}

func TestClaudeUpsertOfficialClearsManaged(t *testing.T) {
	existing := []byte(`{"env":{"ANTHROPIC_BASE_URL":"https://x","ANTHROPIC_AUTH_TOKEN":"sk","ANTHROPIC_API_KEY":"sk","CLAUDE_CODE_SUBAGENT_MODEL":"flash","CLAUDE_CODE_EFFORT_LEVEL":"max","CLAUDE_CODE_AUTO_COMPACT_WINDOW":"786432","KEEP":"1"}}`)
	out, err := UpsertClaudeSettings(existing, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, k := range []string{"ANTHROPIC_BASE_URL", "ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_API_KEY", "CLAUDE_CODE_SUBAGENT_MODEL", "CLAUDE_CODE_EFFORT_LEVEL", "CLAUDE_CODE_AUTO_COMPACT_WINDOW"} {
		if strings.Contains(s, k) {
			t.Fatalf("managed key %s not cleared: %s", k, s)
		}
	}
	if !strings.Contains(s, `"KEEP": "1"`) {
		t.Fatalf("unrelated wiped: %s", s)
	}
}

func TestClaudeApplyAndDetect(t *testing.T) {
	magic := t.TempDir()
	home := t.TempDir()
	_ = os.MkdirAll(filepath.Join(magic, "secrets"), 0o700)
	os.Setenv("MAGIC_PASSPHRASE", "test-pass-provider")
	defer os.Unsetenv("MAGIC_PASSPHRASE")
	if err := secrets.SetProviderSecret(filepath.Join(magic, "secrets"), "SHU_API_KEY", "sk-test-key-1234"); err != nil {
		t.Fatal(err)
	}
	f := &catalog.File{
		Active: "shu",
		Providers: map[string]catalog.Provider{
			"shu": {
				KeyEnv: "SHU_API_KEY",
				Pick:   []string{"gpt-5.6-sol"},
				Models: map[string]string{"default": "gpt-5.6-sol", "haiku": "gpt-5.6-luna"},
				Endpoints: []catalog.Endpoint{
					{ID: "anthropic", BaseURL: "https://api.shu.cool", Protocol: catalog.ProtocolAnthropicMessages},
				},
			},
		},
	}
	_ = catalog.Save(filepath.Join(magic, "providers.yaml"), f)
	ctx := Context{MagicHome: magic, UserHome: home, Catalog: f}
	msg, err := Use(ctx, HarnessClaude, "company")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(msg, "shu") {
		t.Fatalf("%s", msg)
	}
	raw, _ := os.ReadFile(ClaudeSettingsPath(home))
	if DetectClaudeActive(raw, f) != "shu" {
		t.Fatalf("detect=%s body=%s", DetectClaudeActive(raw, f), raw)
	}
	s := string(raw)
	if !strings.Contains(s, "ANTHROPIC_AUTH_TOKEN") {
		t.Fatalf("missing AUTH_TOKEN: %s", raw)
	}
	if !strings.Contains(s, "ANTHROPIC_API_KEY") {
		t.Fatalf("missing API_KEY compat: %s", raw)
	}
	if !strings.Contains(s, "ANTHROPIC_DEFAULT_HAIKU_MODEL") {
		t.Fatalf("missing role env: %s", raw)
	}
	_, err = Use(ctx, HarnessClaude, "official")
	if err != nil {
		t.Fatal(err)
	}
	raw2, _ := os.ReadFile(ClaudeSettingsPath(home))
	if DetectClaudeActive(raw2, f) != ModeOfficial {
		t.Fatalf("%s", raw2)
	}
	if strings.Contains(string(raw2), "CLAUDE_CODE_") || strings.Contains(string(raw2), "ANTHROPIC_AUTH_TOKEN") {
		t.Fatalf("official should clear managed keys: %s", raw2)
	}
}

func TestGrokManagedMarkers(t *testing.T) {
	req := ApplyRequest{
		Name: "shu",
		Provider: catalog.Provider{
			KeyEnv: "SHU_API_KEY",
			Pick:   []string{"gpt-5.6-sol", "gpt-5.6-terra"},
		},
		Endpoint: catalog.Endpoint{ID: "openai-responses", BaseURL: "https://api.shu.cool/v1", Protocol: catalog.ProtocolOpenAIResponses},
	}
	out, err := SetGrokManagedConfig("", "provider", "", req)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, grokManagedStart) || !strings.Contains(out, grokManagedEnd) {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(out, `api_backend = "responses"`) {
		t.Fatalf("%s", out)
	}
	if GetGrokDefaultModel(out) != "gpt-5.6-sol" {
		t.Fatalf("default=%s", GetGrokDefaultModel(out))
	}
	out2, err := SetGrokManagedConfig(out, "official", "", ApplyRequest{Official: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out2, grokManagedStart) {
		t.Fatalf("managed block not cleared: %s", out2)
	}
	if GetGrokDefaultModel(out2) != grokOfficialModel {
		t.Fatalf("%s", out2)
	}
}

func TestPiMergeAndOfficialClear(t *testing.T) {
	magic := t.TempDir()
	home := t.TempDir()
	f := &catalog.File{
		Active: "shu",
		Providers: map[string]catalog.Provider{
			"shu": {
				KeyEnv: "SHU_API_KEY",
				Pick:   []string{"gpt-5.6-sol"},
				Endpoints: []catalog.Endpoint{
					{ID: "openai-responses", BaseURL: "https://api.shu.cool/v1", Protocol: catalog.ProtocolOpenAIResponses},
				},
			},
		},
	}
	ctx := Context{MagicHome: magic, UserHome: home, Catalog: f}
	w := PiWriter{}
	req := ApplyRequest{Name: "shu", APIKey: "sk-real-pi-key", Provider: f.Providers["shu"], Endpoint: f.Providers["shu"].Endpoints[0]}
	if _, err := ApplyFiles(ctx, w, req); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(w.ConfigPath(home))
	if !strings.Contains(string(raw), "magic-shu") || !strings.Contains(string(raw), "sk-real-pi-key") {
		t.Fatalf("%s", raw)
	}
	if _, err := ApplyFiles(ctx, w, ApplyRequest{Official: true}); err != nil {
		t.Fatal(err)
	}
	raw2, _ := os.ReadFile(w.ConfigPath(home))
	if strings.Contains(string(raw2), "magic-shu") {
		t.Fatalf("%s", raw2)
	}
}

func TestGatewayHTTPtest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-ok" {
			w.WriteHeader(401)
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()
	msg, err := TestGateway(srv.Client(), srv.URL+"/v1/models", "sk-ok", "bearer")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(msg, "HTTP 200") {
		t.Fatalf("%s", msg)
	}
	_, err = TestGateway(srv.Client(), srv.URL+"/v1/models", "bad", "bearer")
	if err == nil {
		t.Fatal("expected 401")
	}
}

func TestSeedAndBackupLimit(t *testing.T) {
	magic := t.TempDir()
	added, err := SeedDefaults(magic)
	if err != nil {
		t.Fatal(err)
	}
	if len(added) == 0 {
		t.Fatal("expected seeded providers")
	}
	f, err := LoadCatalog(magic)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := f.Providers["deepseek"]; !ok {
		t.Fatal("deepseek missing")
	}
	if _, ok := f.Providers["shu"]; ok {
		t.Fatal("relays must not be seeded")
	}
	// backup trim
	src := filepath.Join(t.TempDir(), "settings.json")
	_ = os.WriteFile(src, []byte(`{}`), 0o600)
	for i := 0; i < 5; i++ {
		_, err := CreateBackup(magic, HarnessClaude, src, "settings", ".json")
		if err != nil {
			t.Fatal(err)
		}
	}
	files, _ := ListBackups(magic, HarnessClaude)
	if len(files) > BackupLimit {
		t.Fatalf("limit=%d got=%d", BackupLimit, len(files))
	}
}

func TestStubNotImplemented(t *testing.T) {
	w := Writer("gemini")
	if w == nil || w.Implemented() {
		t.Fatal(w)
	}
	_, err := w.Files(Context{}, ApplyRequest{Official: true})
	if err == nil {
		t.Fatal("expected not implemented")
	}
}

func TestOpenCodeApplyAndOfficialClear(t *testing.T) {
	tmp := t.TempDir()
	magic := filepath.Join(tmp, "lib")
	home := filepath.Join(tmp, "home")
	if err := os.MkdirAll(filepath.Dir(OpenCodeWriter{}.ConfigPath(home)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(magic, 0o755); err != nil {
		t.Fatal(err)
	}
	prov := []byte("active: shu\nproviders:\n  shu:\n    key_env: SHU_API_KEY\n    pick: [gpt-5.6-sol]\n    base_url: https://api.example.com/v1\n    protocol: openai-responses\n")
	if err := os.WriteFile(filepath.Join(magic, "providers.yaml"), prov, 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := Context{MagicHome: magic, UserHome: home}
	req := ApplyRequest{
		Name:     "shu",
		Provider: mustProvider(t, ctx, "shu"),
		Endpoint: catalog.Endpoint{ID: "openai-responses", BaseURL: "https://api.example.com/v1", Protocol: "openai-responses"},
		APIKey:   "sk-test",
	}
	if _, err := ApplyFiles(ctx, OpenCodeWriter{}, req); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile((OpenCodeWriter{}).ConfigPath(home))
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, `"magic-shu"`) || !strings.Contains(s, "https://api.example.com/v1") {
		t.Fatalf("missing magic provider:\n%s", s)
	}
	if _, err := ApplyFiles(ctx, OpenCodeWriter{}, ApplyRequest{Official: true}); err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(OpenCodeWriter{}.ConfigPath(home))
	if strings.Contains(string(raw), "magic-shu") {
		t.Fatalf("official did not clear:\n%s", raw)
	}
}

func TestWorkBuddyApplyWritesRealKey(t *testing.T) {
	tmp := t.TempDir()
	magic := filepath.Join(tmp, "lib")
	home := filepath.Join(tmp, "home")
	if err := os.MkdirAll(filepath.Join(home, ".codebuddy"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".codebuddy", "models.json"), []byte(`{
  "models": [{"id": "user-local", "name": "Local"}],
  "availableModels": ["user-local"]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(magic, 0o755); err != nil {
		t.Fatal(err)
	}
	ctx := Context{MagicHome: magic, UserHome: home}
	req := ApplyRequest{
		Name:   "deepseek",
		APIKey: "sk-deepseek-real-1234",
		Provider: catalog.Provider{
			KeyEnv: "DEEPSEEK_API_KEY",
			Pick:   []string{"deepseek-v4-pro"},
		},
		Endpoint: catalog.Endpoint{BaseURL: "https://api.deepseek.com", Protocol: "openai-chat-completions"},
	}
	if _, err := ApplyFiles(ctx, WorkBuddyWriter{}, req); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile((WorkBuddyWriter{}).ConfigPath(home))
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "sk-deepseek-real-1234") || !strings.Contains(s, "chat/completions") {
		t.Fatalf("unexpected models.json:\n%s", s)
	}
	if !strings.Contains(s, "user-local") {
		t.Fatalf("user availableModels dropped:\n%s", s)
	}
	if _, err := ApplyFiles(ctx, WorkBuddyWriter{}, ApplyRequest{Official: true}); err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(WorkBuddyWriter{}.ConfigPath(home))
	if strings.Contains(string(raw), "magic-deepseek") {
		t.Fatalf("official did not clear:\n%s", raw)
	}
}

func mustProvider(t *testing.T, ctx Context, name string) catalog.Provider {
	t.Helper()
	f, err := ctx.File()
	if err != nil {
		t.Fatal(err)
	}
	p, ok := f.Providers[name]
	if !ok {
		t.Fatalf("missing provider %s", name)
	}
	return p
}

// OpenCode reads $XDG_CONFIG_HOME, so its provider projection must land there
// rather than in ~/.config: writing to a directory the harness does not read is
// a silent no-op for the user.
func TestOpenCodeConfigPathFollowsXDGConfigHome(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(t.TempDir(), "xdg-config")
	t.Setenv("XDG_CONFIG_HOME", xdg)

	if got, want := (OpenCodeWriter{}).ConfigPath(home), filepath.Join(xdg, "opencode", "opencode.json"); got != want {
		t.Errorf("ConfigPath = %q, want %q", got, want)
	}

	t.Setenv("XDG_CONFIG_HOME", filepath.Join("relative", "xdg-config"))
	if got, want := (OpenCodeWriter{}).ConfigPath(home), filepath.Join(home, ".config", "opencode", "opencode.json"); got != want {
		t.Errorf("relative XDG: ConfigPath = %q, want fallback %q", got, want)
	}
}
