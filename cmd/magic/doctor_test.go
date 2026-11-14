package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"magic/internal/catalog"
	"magic/internal/provider"
)

// doctorFixtureProviders writes a one-provider library whose active provider has
// exactly the endpoints given, plus a stored key.
func doctorFixtureProviders(t *testing.T, magicHome, endpoints string) {
	t.Helper()
	writeFileAt(t, filepath.Join(magicHome, "providers.yaml"), `active: example
providers:
  example:
    key_env: EXAMPLE_API_KEY
    pick: [example-model]
    endpoints:
`+endpoints)
	writeFileAt(t, filepath.Join(magicHome, "secrets", "providers.json"), `{"EXAMPLE_API_KEY":"sk-live-abcdef123456"}`)
}

// doctorFixtureMagicYAML enables exactly the given adapters.
func doctorFixtureMagicYAML(t *testing.T, magicHome string, ids ...string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("version: \"1.0\"\nschema_version: 1\nadapters:\n  enabled:\n")
	for _, id := range ids {
		b.WriteString("    - " + id + "\n")
	}
	writeFileAt(t, filepath.Join(magicHome, "magic.yaml"), b.String())
}

// anthropicEndpoint is what Claude can project.
const anthropicEndpoint = `      - id: anthropic
        base_url: https://api.example.com/anthropic
        protocol: anthropic-messages
`

// openaiChatOnlyEndpoint is what Claude cannot project: doctor must say so.
const openaiChatOnlyEndpoint = `      - id: openai-chat
        base_url: https://api.example.com/v1
        protocol: openai-chat-completions
`

func runDoctorCmd(t *testing.T, magicHome, userHome, only string, asJSON bool) (string, error) {
	t.Helper()
	args := []string{"--home", userHome}
	if only != "" {
		args = append(args, "--only", only)
	}
	if asJSON {
		args = append(args, "--json")
	}
	return runMagic(t, magicHome, cmdDoctor(), args...)
}

// An empty library is not a healthy one: no provider means nothing can be
// projected, and doctor must say so with a non-zero exit code.
func TestDoctorFailsOnEmptyLibrary(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	out, err := runDoctorCmd(t, magicHome, userHome, "", false)
	if err == nil {
		t.Fatalf("doctor passed on an empty library:\n%s", out)
	}
	if !strings.Contains(err.Error(), "FAIL") {
		t.Fatalf("error = %v, want the FAIL summary", err)
	}
	if !strings.Contains(out, "FAIL") || !strings.Contains(out, "没有任何 provider") {
		t.Fatalf("doctor did not report the missing provider:\n%s", out)
	}
}

// A magic.yaml that does not parse is a FAIL of the library check: apply would
// fall back to the built-in default instead of the user's file.
func TestDoctorFailsOnUnparsableMagicYAML(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFileAt(t, filepath.Join(magicHome, "magic.yaml"), "adapters: [oops\n")
	out, err := runDoctorCmd(t, magicHome, userHome, "", false)
	if err == nil || !strings.Contains(out, "magic.yaml 无法解析") {
		t.Fatalf("doctor = %v, want the magic.yaml parse failure:\n%s", err, out)
	}
}

// Every key_env in providers.yaml must have a value, and the finding names the
// command that sets it.
func TestDoctorFailsOnMissingSecret(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFileAt(t, filepath.Join(magicHome, "providers.yaml"), `active: example
providers:
  example:
    key_env: EXAMPLE_API_KEY
    endpoints:
      - id: anthropic
        base_url: https://api.example.com/anthropic
        protocol: anthropic-messages
`)
	out, err := runDoctorCmd(t, magicHome, userHome, "", false)
	if err == nil {
		t.Fatalf("doctor passed without a secret:\n%s", out)
	}
	if !strings.Contains(out, "缺少密钥") || !strings.Contains(out, "magic secrets set EXAMPLE_API_KEY") {
		t.Fatalf("doctor did not name the missing key and its command:\n%s", out)
	}
}

// The protocol check is the "400 after switching provider" check: a provider with
// only an openai-chat endpoint cannot be projected into Claude.
func TestDoctorFailsOnProtocolMismatch(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	doctorFixtureMagicYAML(t, magicHome, "claude")
	doctorFixtureProviders(t, magicHome, openaiChatOnlyEndpoint)
	if err := os.MkdirAll(filepath.Join(userHome, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := runDoctorCmd(t, magicHome, userHome, "claude", false)
	if err == nil {
		t.Fatalf("doctor passed a provider Claude cannot project:\n%s", out)
	}
	if !strings.Contains(out, "Claude Code 不支持 openai-chat-completions") {
		t.Fatalf("doctor did not report the protocol mismatch:\n%s", out)
	}
	if !strings.Contains(out, "加一个 anthropic-messages 的 endpoint") {
		t.Fatalf("doctor did not name the endpoint to add:\n%s", out)
	}
}

// A configured library with an applied session is all OK: no FAIL, exit code 0.
func TestDoctorPassesAfterApply(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	doctorFixtureMagicYAML(t, magicHome, "claude")
	doctorFixtureProviders(t, magicHome, anthropicEndpoint)
	if err := os.MkdirAll(filepath.Join(userHome, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	runApply(t, magicHome, userHome, "claude")

	out, err := runDoctorCmd(t, magicHome, userHome, "claude", false)
	if err != nil {
		t.Fatalf("doctor after apply = %v, want no FAIL:\n%s", err, out)
	}
	if !strings.Contains(out, "0 WARN / 0 FAIL") {
		t.Fatalf("doctor after apply still warns:\n%s", out)
	}
}

// An installed software whose provider write is a stub is a WARN carrying the
// reason recorded in internal/provider/stubdialects.go.
func TestDoctorWarnsOnStubSoftware(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	doctorFixtureMagicYAML(t, magicHome, "zcode")
	doctorFixtureProviders(t, magicHome, anthropicEndpoint)
	if err := os.MkdirAll(filepath.Join(userHome, ".zcode"), 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := runDoctorCmd(t, magicHome, userHome, "zcode", false)
	if err != nil {
		t.Fatalf("doctor = %v, want WARN only:\n%s", err, out)
	}
	reason, ok := provider.StubReason("zcode")
	if !ok {
		t.Fatal("stubdialects.go has no reason for zcode")
	}
	if !strings.Contains(out, reason) {
		t.Fatalf("doctor did not print the zcode stub reason:\n%s", out)
	}
}

// A target whose parent directory has no write bit is a FAIL: apply cannot write
// it, and doctor names the directory instead of the file.
func TestDoctorFailsOnUnwritableTarget(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	doctorFixtureMagicYAML(t, magicHome, "claude")
	doctorFixtureProviders(t, magicHome, anthropicEndpoint)
	claudeDir := filepath.Join(userHome, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(claudeDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(claudeDir, 0o755) })

	out, err := runDoctorCmd(t, magicHome, userHome, "claude", false)
	if err == nil {
		t.Fatalf("doctor passed an unwritable target:\n%s", out)
	}
	if !strings.Contains(out, "父目录不可写") || !strings.Contains(out, claudeDir) {
		t.Fatalf("doctor did not report the unwritable directory:\n%s", out)
	}
}

// --json is the machine-readable view of the same findings: every entry carries
// id / status / message, and the per-software ids name the adapter.
func TestDoctorJSONCarriesEveryCheck(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	doctorFixtureMagicYAML(t, magicHome, "zcode")
	doctorFixtureProviders(t, magicHome, anthropicEndpoint)
	if err := os.MkdirAll(filepath.Join(userHome, ".zcode"), 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := runDoctorCmd(t, magicHome, userHome, "zcode", true)
	if err != nil {
		t.Fatalf("doctor --json = %v:\n%s", err, out)
	}
	var rep struct {
		Results []struct {
			ID      string `json:"id"`
			Status  string `json:"status"`
			Message string `json:"message"`
			Action  string `json:"action"`
		} `json:"results"`
		OK   int `json:"ok"`
		Warn int `json:"warn"`
		Fail int `json:"fail"`
	}
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatalf("parse doctor JSON: %v\n%s", err, out)
	}
	if len(rep.Results) < 7 {
		t.Fatalf("doctor reported %d findings, want at least one per check:\n%s", len(rep.Results), out)
	}
	statuses := map[string]bool{doctorOK: true, doctorWarn: true, doctorFail: true}
	byID := map[string]string{}
	for _, r := range rep.Results {
		if !statuses[r.Status] {
			t.Errorf("finding %q has status %q", r.ID, r.Status)
		}
		if r.ID == "" || r.Message == "" {
			t.Errorf("finding %+v is missing id or message", r)
		}
		byID[r.ID] = r.Status
	}
	if byID["stubs.zcode"] != doctorWarn {
		t.Fatalf("stubs.zcode = %q, want WARN (findings: %v)", byID["stubs.zcode"], byID)
	}
	if byID["library.home"] != doctorOK {
		t.Fatalf("library.home = %q, want OK", byID["library.home"])
	}
	if byID["backups"] != doctorWarn {
		t.Fatalf("backups = %q, want WARN before the first apply", byID["backups"])
	}
	if rep.Fail != 0 || rep.OK+rep.Warn != len(rep.Results) {
		t.Fatalf("counts do not add up: %+v", rep)
	}
}

// doctor rejects an adapter id that does not exist instead of silently checking
// nothing.
func TestDoctorRejectsUnknownOnly(t *testing.T) {
	_, err := runDoctorCmd(t, t.TempDir(), t.TempDir(), "no-such-adapter", false)
	if err == nil || !strings.Contains(err.Error(), "unknown adapter") {
		t.Fatalf("error = %v, want the unknown adapter failure", err)
	}
}

// The protocol check answers with the endpoint PreferredEndpoint actually picks,
// so doctor and `provider use` cannot disagree.
func TestDoctorProtocolMessageNamesSelectedEndpoint(t *testing.T) {
	p := catalog.Provider{Endpoints: []catalog.Endpoint{
		{ID: "chat", BaseURL: "https://api.example.com/v1", Protocol: catalog.ProtocolOpenAIChatCompletions},
		{ID: "responses", BaseURL: "https://api.example.com/v1", Protocol: catalog.ProtocolOpenAIResponses},
	}}
	ep, err := provider.PreferredEndpoint(p, "codex")
	if err != nil {
		t.Fatal(err)
	}
	if ep.ID != "responses" {
		t.Fatalf("PreferredEndpoint picked %q, want responses", ep.ID)
	}
}
