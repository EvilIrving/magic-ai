package provider

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"magic/internal/backup"
	"magic/internal/catalog"
)

// filesFixture is one harness's user home seed: the files a real install has on
// disk before Magic projects a provider into it.
type filesFixture struct {
	harness string
	files   map[string]string // user-home relative path → content
}

// writeFixture writes the seed under a fresh user home and returns the context.
func writeFixture(t *testing.T, fx filesFixture) Context {
	t.Helper()
	clearHarnessHomeEnv(t)
	magicHome, userHome := t.TempDir(), t.TempDir()
	for rel, content := range fx.files {
		path := filepath.Join(userHome, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return Context{MagicHome: magicHome, UserHome: userHome}
}

// clearHarnessHomeEnv pins every harness root to the fixture home: the writer
// paths come from the adapters, and those honor the harness's own env override.
func clearHarnessHomeEnv(t *testing.T) {
	t.Helper()
	for _, env := range []string{
		"KIMI_CODE_HOME", "VIBE_HOME", "HERMES_HOME", "PI_CONFIG_DIR", "PI_CODING_AGENT_DIR",
		"QODER_CONFIG_DIR", "MINIMAX_DATA_DIR", "MAVIS_DATA_DIR",
	} {
		t.Setenv(env, "")
	}
}

// providerRequest is the request every harness gets: one provider, one endpoint,
// one key.
func providerRequest(t *testing.T, protocol string) ApplyRequest {
	t.Helper()
	p := catalog.Provider{
		KeyEnv: "EXAMPLE_API_KEY",
		Pick:   []string{"example-model"},
		Models: map[string]string{"default": "example-model"},
	}
	return ApplyRequest{
		Name:     "example",
		Provider: p,
		Endpoint: catalog.Endpoint{ID: "example", BaseURL: "https://api.example.com/v1", Protocol: protocol},
		APIKey:   "sk-live-abcdef123456",
	}
}

// fileWritePaths names what each harness's projection must produce.
var filesHarnessPaths = map[string][]string{
	"claude":    {".claude/settings.json"},
	"codex":     {".codex/models.json", ".codex/config.toml"},
	"grok":      {".grok/config.toml", ".grok/provider-mode"},
	"pi":        {".pi/agent/models.json", ".pi/agent/models.magic.json"},
	"opencode":  {".config/opencode/opencode.json"},
	"workbuddy": {".codebuddy/models.json"},
	"kimi-code": {".kimi-code/config.toml", ".kimi-code/magic-provider-state.json"},
	"vibe":      {".vibe/config.toml", ".vibe/magic-provider-state.json"},
	"hermes":    {".hermes/config.yaml", ".hermes/magic-provider-state.json"},
	"omp":       {".omp/agent/models.yml"},
	"droid":     {".factory/settings.json"},
	"qoder":     {".qoder/settings.json", ".qoder/magic-provider-state.json"},
	"mmx":       {".minimax/config.yaml", ".minimax/magic-provider-state.json"},
}

var filesHarnessSeed = map[string]filesFixture{
	"claude": {"claude", map[string]string{
		".claude/settings.json": `{"env":{"KEEP":"1"},"statusLine":{"type":"command"}}`,
	}},
	"codex": {"codex", map[string]string{
		".codex/config.toml": "model = \"gpt-5\"\napproval_policy = \"never\"\n",
	}},
	"grok": {"grok", map[string]string{
		".grok/config.toml": "[models]\ndefault = \"grok-4.5\"\n",
	}},
	"pi": {"pi", map[string]string{
		".pi/agent/models.json": `{"providers":{"user":{"api":"https://user.example/v1"}}}`,
	}},
	"opencode": {"opencode", map[string]string{
		".config/opencode/opencode.json": `{"theme":"dark"}`,
	}},
	"workbuddy": {"workbuddy", map[string]string{
		".codebuddy/models.json": `{"models":[{"id":"user-local","name":"Local"}],"availableModels":["user-local"]}`,
	}},
	"kimi-code": {"kimi-code", map[string]string{
		".kimi-code/config.toml": "default_model = \"kimi-code/k3\"\n\n[providers.\"managed:kimi-code\"]\ntype = \"kimi\"\nbase_url = \"https://api.kimi.com/coding/v1\"\napi_key = \"\"\n\n[models.\"kimi-code/k3\"]\nprovider = \"managed:kimi-code\"\nmodel = \"k3\"\nmax_context_size = 1048576\n",
	}},
	"vibe": {"vibe", map[string]string{
		".vibe/config.toml": "active_model = \"mistral-medium-3.5\"\n\n[[providers]]\nname = \"mistral\"\napi_base = \"https://api.mistral.ai/v1\"\napi_key_env_var = \"MISTRAL_API_KEY\"\nbackend = \"mistral\"\n",
	}},
	"hermes": {"hermes", map[string]string{
		".hermes/config.yaml": "model:\n  provider: openrouter\n  default: anthropic/claude-opus-4.6\n  base_url: https://openrouter.ai/api/v1\n",
	}},
	"omp": {"omp", map[string]string{
		".omp/agent/models.yml": "providers:\n  user:\n    baseUrl: https://user.example/v1\n    apiKey: user-key\n    api: openai-completions\n",
	}},
	"droid": {"droid", map[string]string{
		".factory/settings.json": `{"model":"claude-opus-5-5","customModels":[{"model":"local","displayName":"Local","baseUrl":"https://local.example/v1","provider":"generic-chat-completion-api"}]}`,
	}},
	"qoder": {"qoder", map[string]string{
		".qoder/settings.json": `{"theme":"dark","model":{"name":"qwen3-max"},"providers":{"user-local":{"type":"openai-compatible","protocol":"openai","baseUrl":"https://user.example/v1","apiKey":"user-key","models":[{"model":"u1"}]}}}`,
	}},
	"mmx": {"mmx", map[string]string{
		".minimax/config.yaml": "defaultModel: minimax/MiniMax-M3\ncustom_provider:\n  user:\n    name: user\n    kind: custom\n    options:\n      apiKey: user-key\n      baseURL: https://user.example/v1\n",
	}},
}

var filesHarnessProtocol = map[string]string{
	"claude":    catalog.ProtocolAnthropicMessages,
	"codex":     catalog.ProtocolOpenAIResponses,
	"grok":      catalog.ProtocolOpenAIResponses,
	"pi":        catalog.ProtocolOpenAIResponses,
	"opencode":  catalog.ProtocolOpenAIResponses,
	"workbuddy": catalog.ProtocolOpenAIChatCompletions,
	"kimi-code": catalog.ProtocolOpenAIChatCompletions,
	"vibe":      catalog.ProtocolOpenAIChatCompletions,
	"hermes":    catalog.ProtocolOpenAIChatCompletions,
	"omp":       catalog.ProtocolOpenAIResponses,
	"droid":     catalog.ProtocolOpenAIChatCompletions,
	"qoder":     catalog.ProtocolOpenAIResponses,
	"mmx":       catalog.ProtocolAnthropicMessages,
}

// Every harness's Files output is exactly what lands on disk: the projection
// writes the bytes Files computed, at the mode and path Files declared, and a
// second run leaves the files untouched.
func TestFilesLandWhatApplyWrites(t *testing.T) {
	for harness, seed := range filesHarnessSeed {
		t.Run(harness, func(t *testing.T) {
			ctx := writeFixture(t, seed)
			w := Writer(harness)
			if w == nil || !w.Implemented() {
				t.Fatalf("no implemented writer for %s", harness)
			}
			req := providerRequest(t, filesHarnessProtocol[harness])
			writes, err := w.Files(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			// Files is idempotent: the same request twice is the same bytes.
			// Map iteration order in a projection would make plan compute bytes
			// apply never writes, and verify report drift that is not there.
			again, err := w.Files(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			if len(again) != len(writes) {
				t.Fatalf("a repeated Files returns %d writes, first returned %d", len(again), len(writes))
			}
			for i := range writes {
				if again[i].Path != writes[i].Path || again[i].Mode != writes[i].Mode || again[i].Remove != writes[i].Remove {
					t.Fatalf("a repeated Files changed write %d: %s → %s", i, writes[i].Path, again[i].Path)
				}
				if !bytes.Equal(again[i].Data, writes[i].Data) {
					t.Fatalf("%s: Files is not idempotent:\nfirst:  %s\nsecond: %s", writes[i].Path, writes[i].Data, again[i].Data)
				}
			}
			var got []string
			for _, fw := range writes {
				if fw.Remove {
					t.Fatalf("%s: a provider projection must not remove files", harness)
				}
				rel, err := filepath.Rel(ctx.UserHome, fw.Path)
				if err != nil {
					t.Fatal(err)
				}
				got = append(got, filepath.ToSlash(rel))
			}
			if want := filesHarnessPaths[harness]; strings.Join(got, ",") != strings.Join(want, ",") {
				t.Fatalf("Files paths = %v, want %v", got, want)
			}

			if _, err := ApplyFiles(ctx, w, req); err != nil {
				t.Fatal(err)
			}
			for _, fw := range writes {
				onDisk, err := os.ReadFile(fw.Path)
				if err != nil {
					t.Fatalf("%s: %v", fw.Path, err)
				}
				if !bytes.Equal(onDisk, fw.Data) {
					t.Fatalf("%s: disk bytes differ from Files bytes:\ndisk: %s\nfiles: %s", fw.Path, onDisk, fw.Data)
				}
				info, err := os.Stat(fw.Path)
				if err != nil {
					t.Fatal(err)
				}
				if info.Mode().Perm() != fw.Mode.Perm() {
					t.Fatalf("%s: mode = %04o, want %04o", fw.Path, info.Mode().Perm(), fw.Mode.Perm())
				}
			}

			// A second projection writes nothing: same content, same permission.
			before := statAll(t, writes)
			if _, err := ApplyFiles(ctx, w, req); err != nil {
				t.Fatal(err)
			}
			after := statAll(t, writes)
			if before != after {
				t.Fatalf("a repeated projection rewrote files:\nbefore %v\nafter  %v", before, after)
			}
		})
	}
}

// statAll renders inode+mtime of every target, so a rewrite is visible even when
// the bytes are the same.
func statAll(t *testing.T, writes []FileWrite) string {
	t.Helper()
	var b strings.Builder
	for _, fw := range writes {
		info, err := os.Stat(fw.Path)
		if err != nil {
			t.Fatal(err)
		}
		b.WriteString(fw.Path + " " + info.ModTime().String() + " ")
	}
	return b.String()
}

// The Claude projection writes the documented env block and keeps the user's own
// settings and env keys.
func TestFilesClaudeWritesManagedEnv(t *testing.T) {
	ctx := writeFixture(t, filesHarnessSeed["claude"])
	writes, err := ClaudeWriter{}.Files(ctx, providerRequest(t, catalog.ProtocolAnthropicMessages))
	if err != nil {
		t.Fatal(err)
	}
	got := string(writes[0].Data)
	for _, want := range []string{
		`"ANTHROPIC_BASE_URL": "https://api.example.com/v1"`,
		`"ANTHROPIC_AUTH_TOKEN": "sk-live-abcdef123456"`,
		`"ANTHROPIC_API_KEY": "sk-live-abcdef123456"`,
		`"ANTHROPIC_MODEL": "example-model"`,
		`"KEEP": "1"`,
		`"statusLine"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("settings.json missing %s:\n%s", want, got)
		}
	}
}

// Official mode returns the Magic-managed entries cleared. Pi and OpenCode arrive
// here through `provider use none` (they have no first-party path), the others
// through `provider use official`; both mean "clear what Magic manages here".
func TestFilesOfficialClearsManagedEntries(t *testing.T) {
	// Seed the Magic entries the projection owns, so clearing has something to do.
	seeds := map[string]filesFixture{
		"claude": {"claude", map[string]string{".claude/settings.json": `{"env":{"ANTHROPIC_BASE_URL":"https://magic.example","ANTHROPIC_AUTH_TOKEN":"sk-magic","KEEP":"1"}}`}},
		"codex": {"codex", map[string]string{
			".codex/config.toml": "model_provider = \"example\"\nmodel_catalog_json = \"/home/u/.codex/models.json\"\n",
			".codex/models.json": `{"models":[{"slug":"example-model"}]}`,
		}},
		"grok":      {"grok", map[string]string{".grok/config.toml": "# magic-managed-catalog-start\n[model.\"example-model\"]\nmodel = \"example-model\"\n# magic-managed-catalog-end\n[models]\ndefault = \"example-model\"\n"}},
		"pi":        {"pi", map[string]string{".pi/agent/models.json": `{"providers":{"magic-example":{"baseUrl":"https://magic.example"},"user":{"api":"x"}}}`}},
		"opencode":  {"opencode", map[string]string{".config/opencode/opencode.json": `{"model":"magic-example/example-model","provider":{"magic-example":{"options":{}},"keep":{"options":{}}}}`}},
		"workbuddy": {"workbuddy", map[string]string{".codebuddy/models.json": `{"models":[{"id":"magic-example-example-model"},{"id":"user-local"}],"availableModels":["magic-example-example-model","user-local"]}`}},
	}
	for harness, seed := range seeds {
		t.Run(harness, func(t *testing.T) {
			ctx := writeFixture(t, seed)
			// Official and none both arrive as Official: clear what Magic manages.
			w := Writer(harness)
			writes, err := w.Files(ctx, ApplyRequest{Official: true})
			if err != nil {
				t.Fatal(err)
			}
			for _, fw := range writes {
				body := string(fw.Data)
				switch harness {
				case "claude":
					if strings.Contains(body, "ANTHROPIC_") {
						t.Fatalf("managed env keys survive official mode:\n%s", body)
					}
					if !strings.Contains(body, `"KEEP": "1"`) {
						t.Fatalf("official mode dropped a user env key:\n%s", body)
					}
				case "codex":
					if fw.Remove {
						if filepath.Base(fw.Path) != "models.json" {
							t.Fatalf("official mode removes %s", fw.Path)
						}
						continue
					}
					if strings.Contains(body, "model_catalog_json") || strings.Contains(body, "magic") {
						t.Fatalf("official mode keeps the Magic catalog:\n%s", body)
					}
					if !strings.Contains(body, `model_provider = "openai"`) {
						t.Fatalf("official mode must switch to the first-party provider:\n%s", body)
					}
				case "grok":
					if filepath.Base(fw.Path) == "provider-mode" {
						if strings.TrimSpace(body) != "official" {
							t.Fatalf("provider-mode = %q, want official", body)
						}
						continue
					}
					if strings.Contains(body, "# magic-managed-catalog-start") {
						t.Fatalf("managed block survives official mode:\n%s", body)
					}
					if !strings.Contains(body, `default = "grok-4.5"`) {
						t.Fatalf("official mode must restore the default model:\n%s", body)
					}
				case "pi":
					if strings.Contains(body, "magic-example") {
						t.Fatalf("official mode keeps the Magic provider:\n%s", body)
					}
					if !strings.Contains(body, `"user"`) {
						t.Fatalf("official mode dropped a user provider:\n%s", body)
					}
				case "opencode":
					if strings.Contains(body, "magic-example") {
						t.Fatalf("official mode keeps the Magic provider:\n%s", body)
					}
					if !strings.Contains(body, `"keep"`) {
						t.Fatalf("official mode dropped a user provider:\n%s", body)
					}
				case "workbuddy":
					if strings.Contains(body, "magic-example-example-model") {
						t.Fatalf("official mode keeps the Magic models:\n%s", body)
					}
					if !strings.Contains(body, "user-local") {
						t.Fatalf("official mode dropped a user model:\n%s", body)
					}
				}
			}
		})
	}
}

// Pi writes the models.magic.json sidecar next to models.json; it carries only
// the Magic-managed provider.
func TestFilesPiCarriesSidecar(t *testing.T) {
	ctx := writeFixture(t, filesHarnessSeed["pi"])
	writes, err := PiWriter{}.Files(ctx, providerRequest(t, catalog.ProtocolOpenAIResponses))
	if err != nil {
		t.Fatal(err)
	}
	var side *FileWrite
	for i := range writes {
		if writes[i].Role == RoleSidecar {
			side = &writes[i]
		}
	}
	if side == nil {
		t.Fatal("no sidecar in the Pi projection")
	}
	if want := filepath.Join(ctx.UserHome, ".pi", "agent", "models.magic.json"); side.Path != want {
		t.Fatalf("sidecar path = %s, want %s", side.Path, want)
	}
	var root map[string]any
	if err := json.Unmarshal(side.Data, &root); err != nil {
		t.Fatal(err)
	}
	providers := root["providers"].(map[string]any)
	if _, ok := providers["magic-example"]; !ok {
		t.Fatalf("sidecar does not carry the Magic provider: %s", side.Data)
	}
	if _, ok := providers["user"]; ok {
		t.Fatalf("sidecar carries a provider Magic does not manage: %s", side.Data)
	}
}

// Codex official mode removes the Magic-managed catalog: the plan can say so, and
// the removal is backed up so rollback can bring the file back.
func TestApplyFilesRemovesCodexCatalogWithBackup(t *testing.T) {
	ctx := writeFixture(t, filesHarnessSeed["codex"])
	models := filepath.Join(ctx.UserHome, ".codex", "models.json")
	if err := os.WriteFile(models, []byte(`{"models":[{"slug":"example-model"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx.BackupDir = filepath.Join(ctx.MagicHome, "backups", "20260101-000000")
	before, err := os.ReadFile(models)
	if err != nil {
		t.Fatal(err)
	}

	writes, err := CodexWriter{}.Files(ctx, ApplyRequest{Official: true})
	if err != nil {
		t.Fatal(err)
	}
	removals := 0
	for _, fw := range writes {
		if fw.Remove {
			removals++
			if fw.Data != nil {
				t.Fatalf("a removal carries data: %s", fw.Data)
			}
		}
	}
	if removals != 1 {
		t.Fatalf("removals = %d, want the Magic-managed catalog", removals)
	}
	if _, err := ApplyFiles(ctx, CodexWriter{}, ApplyRequest{Official: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(models); !os.IsNotExist(err) {
		t.Fatalf("official mode left %s behind (err=%v)", models, err)
	}

	m, err := backup.Read(ctx.BackupDir)
	if err != nil {
		t.Fatal(err)
	}
	if m == nil {
		t.Fatal("no manifest: rollback could not restore the removed catalog")
	}
	var found *backup.Entry
	for i := range m.Entries {
		if m.Entries[i].Target == models {
			found = &m.Entries[i]
		}
	}
	if found == nil {
		t.Fatalf("manifest does not record %s", models)
	}
	if found.Action != backup.ActionUpdate {
		t.Fatalf("action = %s, want update (the removed content is in the set)", found.Action)
	}
	saved, err := os.ReadFile(filepath.Join(ctx.BackupDir, found.Backup))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(saved, before) {
		t.Fatalf("backup copy is not the removed content: %s", saved)
	}
}

// Files is pure: it computes what a projection would write and leaves the disk
// alone, which is what lets plan call it.
func TestFilesWritesNothing(t *testing.T) {
	clearHarnessHomeEnv(t)
	magicHome, userHome := t.TempDir(), t.TempDir()
	ctx := Context{MagicHome: magicHome, UserHome: userHome}
	for harness, protocol := range filesHarnessProtocol {
		w := Writer(harness)
		writes, err := w.Files(ctx, providerRequest(t, protocol))
		if err != nil {
			t.Fatalf("%s: %v", harness, err)
		}
		if len(writes) == 0 {
			t.Fatalf("%s: no files to write", harness)
		}
		for _, fw := range writes {
			if _, err := os.Stat(fw.Path); !os.IsNotExist(err) {
				t.Fatalf("%s: Files created %s (err=%v)", harness, fw.Path, err)
			}
		}
	}
	if entries, err := os.ReadDir(userHome); err != nil || len(entries) != 0 {
		t.Fatalf("Files touched the user home: %v (err=%v)", entries, err)
	}
}

// The Claude merge keeps everything Magic does not manage: unrelated settings,
// env keys and hooks survive a provider projection.
func TestUpsertClaudeSettingsPreservesUnrelated(t *testing.T) {
	existing := []byte(`{
  "permissions": {"allow": ["Bash"]},
  "hooks": {"PreToolUse": []},
  "env": {"KEEP_ME": "yes", "ANTHROPIC_BASE_URL": "https://old.example"}
}
`)
	env := map[string]string{
		"ANTHROPIC_BASE_URL":   "https://api.example.com/anthropic",
		"ANTHROPIC_AUTH_TOKEN": "sk-SECRET-TOKEN99",
	}
	out, err := UpsertClaudeSettings(existing, env, false)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	if err := json.Unmarshal(out, &root); err != nil {
		t.Fatal(err)
	}
	if _, ok := root["permissions"]; !ok {
		t.Fatal("permissions wiped")
	}
	if _, ok := root["hooks"]; !ok {
		t.Fatal("hooks wiped")
	}
	envObj := root["env"].(map[string]any)
	if envObj["KEEP_ME"] != "yes" {
		t.Fatalf("KEEP_ME lost: %v", envObj)
	}
	if envObj["ANTHROPIC_BASE_URL"] != "https://api.example.com/anthropic" {
		t.Fatalf("base url: %v", envObj)
	}
	if envObj["ANTHROPIC_AUTH_TOKEN"] != "sk-SECRET-TOKEN99" {
		t.Fatalf("auth token: %v", envObj)
	}
}

// clearResidueCase is one harness's clear expectations: the fragments provider
// mode must have written, and what must survive `none`.
type clearResidueCase struct {
	// projected are fragments the provider projection puts in the harness config.
	projected []string
	// kept are the user's own fragments: the clear must put them back.
	kept []string
	// gone are fragments the clear must not leave behind.
	gone []string
}

var clearResidueCases = map[string]clearResidueCase{
	"kimi-code": {
		projected: []string{`[providers."magic-example"]`, `type = "openai"`, `[models."magic-example/example-model"]`, `default_model = "magic-example/example-model"`},
		kept:      []string{`default_model = "kimi-code/k3"`, `[providers."managed:kimi-code"]`, `[models."kimi-code/k3"]`},
		gone:      []string{"magic-", "provider: custom"},
	},
	"vibe": {
		projected: []string{"# magic-managed-start", "[[providers]]", `name = "magic-example"`, `active_model = "magic-example/example-model"`},
		kept:      []string{`active_model = "mistral-medium-3.5"`, `name = "mistral"`, `api_key_env_var = "MISTRAL_API_KEY"`},
		gone:      []string{"magic-managed-start", "magic-example"},
	},
	"hermes": {
		projected: []string{"provider: custom", "default: example-model", "base_url: https://api.example.com/v1", "api_key: sk-live-abcdef123456", "api_mode: chat_completions"},
		kept:      []string{"provider: openrouter", "default: anthropic/claude-opus-4.6", "base_url: https://openrouter.ai/api/v1"},
		gone:      []string{"api_key", "api_mode", "provider: custom"},
	},
	"omp": {
		projected: []string{"magic-example:", "baseUrl: https://api.example.com/v1", "apiKey: sk-live-abcdef123456"},
		kept:      []string{"user:", "baseUrl: https://user.example/v1", "apiKey: user-key"},
		gone:      []string{"magic-", "api.example.com"},
	},
	"qoder": {
		projected: []string{`"name": "magic-example/example-model"`, `"magic-example": {`, `"displayName": "magic-example"`, `"apiKey": "sk-live-abcdef123456"`},
		kept:      []string{`"theme": "dark"`, `"user-local"`, `"apiKey": "user-key"`, `"name": "qwen3-max"`},
		gone:      []string{"magic-example", "api.example.com"},
	},
	"mmx": {
		projected: []string{"defaultModel: custom_provider:magic-example/example-model", "magic-example:", "apiKey: sk-live-abcdef123456", "baseURL: https://api.example.com/v1", "api: anthropic-messages"},
		kept:      []string{"defaultModel: minimax/MiniMax-M3", "user:", "apiKey: user-key", "baseURL: https://user.example/v1"},
		gone:      []string{"magic-", "api.example.com"},
	},
	"droid": {
		projected: []string{`"displayName": "magic-example/example-model"`, `"baseUrl": "https://api.example.com/v1"`, `"apiKey": "sk-live-abcdef123456"`},
		kept:      []string{`"displayName": "Local"`, `"model": "claude-opus-5-5"`},
		gone:      []string{"magic-example", "api.example.com"},
	},
}

// Provider mode writes the dialect, `none` takes it back: the harness config
// carries no Magic marker afterwards, the sidecar is gone, and the user's own
// providers/defaults are exactly what they were before the projection.
func TestFilesClearLeavesNoMagicResidue(t *testing.T) {
	for harness, tc := range clearResidueCases {
		t.Run(harness, func(t *testing.T) {
			ctx := writeFixture(t, filesHarnessSeed[harness])
			w := Writer(harness)
			if _, err := ApplyFiles(ctx, w, providerRequest(t, filesHarnessProtocol[harness])); err != nil {
				t.Fatal(err)
			}
			body := readHarnessConfig(t, ctx, filesHarnessPaths[harness][0])
			for _, want := range tc.projected {
				if !strings.Contains(body, want) {
					t.Fatalf("provider mode did not write %q:\n%s", want, body)
				}
			}

			if _, err := ApplyFiles(ctx, w, ApplyRequest{Official: true}); err != nil {
				t.Fatal(err)
			}
			body = readHarnessConfig(t, ctx, filesHarnessPaths[harness][0])
			for _, gone := range tc.gone {
				if strings.Contains(body, gone) {
					t.Fatalf("clear left %q behind:\n%s", gone, body)
				}
			}
			for _, kept := range tc.kept {
				if !strings.Contains(body, kept) {
					t.Fatalf("clear dropped the user's %q:\n%s", kept, body)
				}
			}
			if strings.Contains(strings.ToLower(body), "magic") {
				t.Fatalf("clear left Magic in the harness config:\n%s", body)
			}
			// The bookkeeping sidecar goes with the projection it tracked.
			for _, rel := range filesHarnessPaths[harness][1:] {
				if _, err := os.Stat(filepath.Join(ctx.UserHome, filepath.FromSlash(rel))); !os.IsNotExist(err) {
					t.Fatalf("clear left %s behind (err=%v)", rel, err)
				}
			}
		})
	}
}

// readHarnessConfig reads one fixture-relative file, failing when it is missing.
func readHarnessConfig(t *testing.T, ctx Context, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(ctx.UserHome, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// Hermes ships `model: ""` until the first setup. Projecting into that sentinel
// must stay one shape on every run: the whole projection (config + sidecar) is
// byte-identical the second time, so plan cannot disagree with apply.
func TestFilesHermesSentinelProjectionIsIdempotent(t *testing.T) {
	ctx := writeFixture(t, filesFixture{"hermes", map[string]string{
		".hermes/config.yaml": `model: ""` + "\n",
	}})
	w := Writer("hermes")
	req := providerRequest(t, catalog.ProtocolOpenAIChatCompletions)
	first, err := w.Files(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyFiles(ctx, w, req); err != nil {
		t.Fatal(err)
	}
	second, err := w.Files(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	for i := range first {
		if !bytes.Equal(first[i].Data, second[i].Data) {
			t.Fatalf("%s: the sentinel projection is not stable:\nfirst:  %s\nsecond: %s", first[i].Path, first[i].Data, second[i].Data)
		}
	}
	// `none` puts the sentinel back, and drops the sidecar.
	if _, err := ApplyFiles(ctx, w, ApplyRequest{Official: true}); err != nil {
		t.Fatal(err)
	}
	if body := readHarnessConfig(t, ctx, ".hermes/config.yaml"); strings.TrimSpace(body) != `model: ""` {
		t.Fatalf("clear did not restore the sentinel:\n%s", body)
	}
	if _, err := os.Stat(filepath.Join(ctx.UserHome, ".hermes", ProjectionStateFile)); !os.IsNotExist(err) {
		t.Fatalf("clear left the sidecar behind (err=%v)", err)
	}
}

// A clear on a harness Magic never wrote must return no writes at all: the
// YAML/JSON writers would otherwise re-serialize (and reformat) a file Magic
// does not own, and the TOML writers would delete the state sidecar that is not
// there. This is what `provider use none` on a fresh install does.
func TestFilesClearOnUntouchedHarnessWritesNothing(t *testing.T) {
	for _, harness := range []string{"kimi-code", "vibe", "hermes", "omp", "droid", "qoder", "mmx"} {
		t.Run(harness, func(t *testing.T) {
			ctx := writeFixture(t, filesHarnessSeed[harness])
			before := readHarnessConfig(t, ctx, filesHarnessPaths[harness][0])
			writes, err := Writer(harness).Files(ctx, ApplyRequest{Official: true})
			if err != nil {
				t.Fatal(err)
			}
			if len(writes) != 0 {
				t.Fatalf("clear on an untouched harness writes %d file(s): %v", len(writes), writePaths(writes))
			}
			if after := readHarnessConfig(t, ctx, filesHarnessPaths[harness][0]); after != before {
				t.Fatalf("clear changed a file it does not own:\nbefore:\n%s\nafter:\n%s", before, after)
			}
		})
	}
}

func writePaths(writes []FileWrite) []string {
	out := make([]string, 0, len(writes))
	for _, fw := range writes {
		out = append(out, fw.Path)
	}
	return out
}
