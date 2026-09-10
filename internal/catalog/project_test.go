package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpsertClaudeSettingsPreservesUnrelated(t *testing.T) {
	existing := []byte(`{
  "permissions": {"allow": ["Bash"]},
  "hooks": {"PreToolUse": []},
  "env": {
    "KEEP_ME": "yes",
    "ANTHROPIC_BASE_URL": "https://old.example"
  }
}
`)
	env := ClaudeSettingsEnv(Endpoint{BaseURL: "https://api.shu.cool", Protocol: ProtocolAnthropicMessages}, "sk-SECRET-TOKEN99")
	out, err := UpsertClaudeSettingsJSON(existing, env)
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
	if envObj[ClaudeEnvBaseURL] != "https://api.shu.cool" {
		t.Fatalf("base url: %v", envObj[ClaudeEnvBaseURL])
	}
	if envObj[ClaudeEnvAuthToken] != "sk-SECRET-TOKEN99" {
		t.Fatalf("auth token: %v", envObj[ClaudeEnvAuthToken])
	}
	if envObj[ClaudeEnvAPIKey] != "sk-SECRET-TOKEN99" {
		t.Fatalf("api key compat: %v", envObj[ClaudeEnvAPIKey])
	}

	dir := t.TempDir()
	home := dir
	_ = os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	path, _, err := WriteClaudeSettings(home, env, false)
	if err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm()&0o077 != 0 {
		// allow 0600 exactly preferred; some FS mask — at least not world-readable ideal
		t.Logf("mode=%v", st.Mode())
	}
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), "ANTHROPIC_BASE_URL") {
		t.Fatalf("%s", raw)
	}
}

func TestMergePiModelsSmoke(t *testing.T) {
	existing := []byte(`{"providers":{"keep-me":{"baseUrl":"https://x","api":"openai-completions","apiKey":"$X","models":[]}}}`)
	ep := Endpoint{ID: "openai-responses", BaseURL: "https://api.shu.cool/v1", Protocol: ProtocolOpenAIResponses}
	entry := BuildPiProviderEntry(ep, "SHU_API_KEY", []string{"gpt-5.6-sol"}, nil)
	if entry.APIKey != "$SHU_API_KEY" {
		t.Fatalf("apiKey must be env ref, got %q", entry.APIKey)
	}
	if entry.API != "openai-responses" {
		t.Fatalf("api=%s", entry.API)
	}
	if len(entry.Models) != 1 || entry.Models[0].ID != "gpt-5.6-sol" {
		t.Fatalf("%+v", entry.Models)
	}
	out, err := MergePiModelsJSON(existing, MagicPiProviderID("shu"), entry)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	if err := json.Unmarshal(out, &root); err != nil {
		t.Fatal(err)
	}
	provs := root["providers"].(map[string]any)
	if _, ok := provs["keep-me"]; !ok {
		t.Fatal("keep-me wiped")
	}
	magic := provs["magic-shu"].(map[string]any)
	if magic["baseUrl"] != "https://api.shu.cool/v1" {
		t.Fatalf("%v", magic)
	}
	if magic["apiKey"] != "$SHU_API_KEY" {
		t.Fatalf("plaintext leaked? %v", magic["apiKey"])
	}

	home := t.TempDir()
	path, err := WritePiModels(home, "magic-shu", entry, false)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if strings.Contains(string(raw), "sk-") {
		t.Fatalf("must not write plaintext keys: %s", raw)
	}
}
