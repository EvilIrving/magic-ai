package login

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"magic/internal/catalog"
	"magic/internal/provider"
	"magic/internal/secrets"
)

func writeProviders(t *testing.T, magicHome, body string) {
	t.Helper()
	if err := os.MkdirAll(magicHome, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(magicHome, "providers.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLookupProviderFromBuiltin(t *testing.T) {
	magic := t.TempDir()
	src, ok, err := LookupProvider(magic, "deepseek")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if src.KeyEnv != "DEEPSEEK_API_KEY" || len(src.Endpoints) == 0 || !src.FromBuiltin {
		t.Fatalf("%+v", src)
	}
}

func TestResolveFallbackToBuiltinPick(t *testing.T) {
	magic := t.TempDir()
	plan, err := Resolve(Options{MagicHome: magic, Provider: "deepseek", NoLive: true})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Source != "builtin-pick" || len(plan.Models) == 0 {
		t.Fatalf("source=%q models=%+v", plan.Source, plan.Models)
	}
}

func TestResolveLiveFromEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-live" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"m1"},{"id":"m2"}]}`))
	}))
	defer srv.Close()

	magic := t.TempDir()
	writeProviders(t, magic, "providers:\n  testprov:\n    key_env: TEST_API_KEY\n    endpoints:\n      - id: ep\n        base_url: "+srv.URL+"\n        protocol: openai-chat-completions\n")

	plan, err := Resolve(Options{MagicHome: magic, Provider: "testprov", APIKey: "sk-live"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Source != "live" || len(plan.Models) != 2 {
		t.Fatalf("source=%q models=%+v", plan.Source, plan.Models)
	}
}

func TestResolveLiveFailureFallsBackToRegistry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	magic := t.TempDir()
	writeProviders(t, magic, "providers:\n  testprov:\n    key_env: TEST_API_KEY\n    endpoints:\n      - id: ep\n        base_url: "+srv.URL+"\n        protocol: openai-chat-completions\n")
	// No builtin pick (testprov is user-only), so registry is the next fallback.
	if err := os.MkdirAll(filepath.Join(magic, "registry"), 0o755); err != nil {
		t.Fatal(err)
	}
	reg := `{"version":1,"source":"test","models":[{"id":"reg-1","provider":"testprov"},{"id":"reg-2","provider":"testprov"}]}`
	if err := os.WriteFile(filepath.Join(magic, "registry", "models.json"), []byte(reg), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := Resolve(Options{MagicHome: magic, Provider: "testprov", APIKey: "sk-bad"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Source != "registry" || len(plan.Models) != 2 {
		t.Fatalf("source=%q models=%+v notes=%v", plan.Source, plan.Models, plan.Notes)
	}
}

func TestApplyWritesPickAndSecret(t *testing.T) {
	magic := t.TempDir()
	writeProviders(t, magic, "providers:\n  testprov:\n    key_env: TEST_API_KEY\n    endpoints:\n      - id: ep\n        base_url: https://example.com/v1\n        protocol: openai-chat-completions\n")
	plan := Plan{Provider: "testprov", KeyEnv: "TEST_API_KEY", APIKey: "sk-store-me", Endpoints: []catalog.Endpoint{{ID: "template", BaseURL: "https://template", Protocol: catalog.ProtocolOpenAIResponses}}}

	path, err := Apply(magic, plan, []string{"m1", "m2", "m1"})
	if err != nil {
		t.Fatal(err)
	}
	f, err := catalog.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	p := f.Providers["testprov"]
	if len(p.Pick) != 2 || p.Pick[0] != "m1" || p.Pick[1] != "m2" {
		t.Fatalf("pick=%v", p.Pick)
	}
	// User endpoints win over the template endpoints.
	if len(p.Endpoints) != 1 || p.Endpoints[0].ID != "ep" {
		t.Fatalf("endpoints=%+v", p.Endpoints)
	}
	v, ok, err := secrets.GetProviderSecret(filepath.Join(magic, "secrets"), "TEST_API_KEY")
	if err != nil || !ok || v != "sk-store-me" {
		t.Fatalf("secret v=%q ok=%v err=%v", v, ok, err)
	}
}

func TestMergeEntryUserWins(t *testing.T) {
	existing := catalog.Provider{
		KeyEnv:    "USER_KEY",
		Endpoints: []catalog.Endpoint{{ID: "user", BaseURL: "https://user", Protocol: catalog.ProtocolOpenAIChatCompletions}},
		Models:    map[string]string{"haiku": "h"},
	}
	plan := Plan{KeyEnv: "TEMPLATE_KEY", Endpoints: []catalog.Endpoint{{ID: "tpl", BaseURL: "https://tpl"}}}
	out := MergeEntry(existing, plan, []string{"new-model"})
	if out.KeyEnv != "USER_KEY" {
		t.Fatalf("key_env=%q", out.KeyEnv)
	}
	if len(out.Endpoints) != 1 || out.Endpoints[0].ID != "user" {
		t.Fatalf("endpoints=%+v", out.Endpoints)
	}
	if out.Models["haiku"] != "h" {
		t.Fatalf("models=%+v", out.Models)
	}
	if len(out.Pick) != 1 || out.Pick[0] != "new-model" {
		t.Fatalf("pick=%+v", out.Pick)
	}
}

func TestDefaultSelection(t *testing.T) {
	models := []provider.RemoteModel{{ID: "a"}, {ID: "b"}}
	sel := DefaultSelection(models, []string{"b", "c"})
	if len(sel) != 1 || sel[0] != "b" {
		t.Fatalf("%v", sel)
	}
}
