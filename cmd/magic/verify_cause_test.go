package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// verifyReport is the --json view verify already had, decoded with the added
// causes field. The old fields keep their names.
type verifyReport struct {
	Checked int `json:"checked"`
	Drift   []struct {
		Adapter string `json:"adapter"`
		Rel     string `json:"rel"`
		Kind    string `json:"kind"`
		Causes  []struct {
			ID     string `json:"id"`
			Reason string `json:"reason"`
			Action string `json:"action"`
		} `json:"causes"`
	} `json:"drift"`
}

func decodeVerifyReport(t *testing.T, out string) verifyReport {
	t.Helper()
	var rep verifyReport
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatalf("parse verify JSON: %v\n%s", err, out)
	}
	return rep
}

// causeOf returns the cause with id, or fails the test.
func (r verifyReport) causeOf(t *testing.T, causeID string) (string, string) {
	t.Helper()
	for _, d := range r.Drift {
		for _, c := range d.Causes {
			if c.ID == causeID {
				return c.Reason, c.Action
			}
		}
	}
	t.Fatalf("no %q cause in %+v", causeID, r.Drift)
	return "", ""
}

// A missing target never got applied (or was deleted): the next step is apply.
func TestVerifyCauseNeverApplied(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFileAt(t, filepath.Join(magicHome, "packs", "harness", "pi", "AGENTS.md"), "# Pi\n")

	out, err := runMagic(t, magicHome, cmdVerify(), "--home", userHome, "--only", "pi", "--json")
	if err == nil {
		t.Fatalf("verify passed without the target:\n%s", out)
	}
	rep := decodeVerifyReport(t, out)
	if len(rep.Drift) != 1 || rep.Drift[0].Kind != "missing" {
		t.Fatalf("drift = %+v, want one missing target", rep.Drift)
	}
	reason, action := rep.causeOf(t, "never-applied")
	if !strings.Contains(reason, "从未 apply") {
		t.Fatalf("reason = %q, want the never-applied explanation", reason)
	}
	if action != "magic apply --only pi" {
		t.Fatalf("action = %q, want apply --only pi", action)
	}
}

// A differ against an existing library mirror file with a parseable target is the
// "manual edit or the software rewrote its fields" case, and plan shows the keys.
func TestVerifyCauseLocalEdit(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFileAt(t, filepath.Join(magicHome, "packs", "harness", "claude", "settings.json"), `{"foo":"bar"}`)
	writeFileAt(t, filepath.Join(userHome, ".claude", "settings.json"), `{"foo":"dirty"}`)

	out, err := runMagic(t, magicHome, cmdVerify(), "--home", userHome, "--only", "claude", "--json")
	if err == nil {
		t.Fatalf("verify passed on a dirty target:\n%s", out)
	}
	rep := decodeVerifyReport(t, out)
	reason, action := rep.causeOf(t, "local-edit")
	if !strings.Contains(reason, "软件升级后重写了字段") {
		t.Fatalf("reason = %q, want the manual-edit / upgrade explanation", reason)
	}
	if action != "magic plan --only claude" {
		t.Fatalf("action = %q, want plan --only claude", action)
	}
}

// A target the provider projection writes also carries the provider switch hint:
// the active provider's endpoint is what decides its content.
func TestVerifyCauseProviderProjection(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeProviderFixture(t, magicHome)
	runApply(t, magicHome, userHome, "claude")

	settings := filepath.Join(userHome, ".claude", "settings.json")
	root := readJSONFile(t, settings)
	env := root["env"].(map[string]any)
	delete(env, "ANTHROPIC_AUTH_TOKEN")
	writeJSONFile(t, settings, root)

	out, err := runMagic(t, magicHome, cmdVerify(), "--home", userHome, "--only", "claude", "--json")
	if err == nil {
		t.Fatalf("verify passed on a dirty provider projection:\n%s", out)
	}
	rep := decodeVerifyReport(t, out)
	reason, action := rep.causeOf(t, "provider-projection")
	if !strings.Contains(reason, "example") {
		t.Fatalf("reason = %q, want the active provider named", reason)
	}
	if action != "magic provider use example --harness claude" {
		t.Fatalf("action = %q, want provider use example --harness claude", action)
	}
}

// The terminal output prints the reason and the next step per drifted row.
func TestVerifyHumanOutputShowsCauseAndAction(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFileAt(t, filepath.Join(magicHome, "packs", "harness", "pi", "AGENTS.md"), "# Pi\n")

	out, err := runMagic(t, magicHome, cmdVerify(), "--home", userHome, "--only", "pi")
	if err == nil {
		t.Fatalf("verify passed without the target:\n%s", out)
	}
	for _, want := range []string{"原因:", "下一步: magic apply --only pi"} {
		if !strings.Contains(out, want) {
			t.Fatalf("verify output lacks %q:\n%s", want, out)
		}
	}
}
