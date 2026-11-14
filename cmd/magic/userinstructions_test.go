package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// `magic plan` lists the shared instruction file under each harness's own path,
// relative to the user home, so a user can see where plans/AGENTS.md lands before
// apply writes it.
func TestPlanListsSharedInstructionFiles(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFileAt(t, filepath.Join(magicHome, "packs", "AGENTS.md"), "# shared\n")
	// The projection goes to installed harnesses, so the plan only lists them.
	for _, dir := range []string{".codex", ".claude"} {
		if err := os.MkdirAll(filepath.Join(userHome, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	out, err := runMagic(t, magicHome, cmdPlan(), "--home", userHome, "--json")
	if err != nil {
		t.Fatalf("plan: %v\n%s", err, out)
	}
	for _, rel := range []string{".codex/AGENTS.md", ".claude/CLAUDE.md"} {
		if !strings.Contains(out, rel) {
			t.Errorf("plan JSON does not list %s:\n%s", rel, out)
		}
	}

	text, err := runMagic(t, magicHome, cmdPlan(), "--home", userHome, "--only", "codex")
	if err != nil {
		t.Fatalf("plan codex: %v\n%s", err, text)
	}
	if !strings.Contains(text, ".codex/AGENTS.md") {
		t.Errorf("plan text does not show the target path:\n%s", text)
	}
}

// apply writes the shared source to the harness path, verify calls the result
// clean, a hand edit is reported as drift, and rollback of the create removes the
// file again.
func TestSharedInstructionFileApplyVerifyRollback(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	const body = "# shared rules\n"
	writeFileAt(t, filepath.Join(magicHome, "packs", "AGENTS.md"), body)
	target := filepath.Join(userHome, ".codex", "AGENTS.md")

	runApply(t, magicHome, userHome, "codex")
	if got := readFileAt(t, target); got != body {
		t.Fatalf("%s = %q, want the shared content", target, got)
	}
	if err := runVerify(t, magicHome, userHome, "codex"); err != nil {
		t.Fatalf("verify after apply: %v", err)
	}

	writeFileAt(t, target, "# edited by hand\n")
	if err := runVerify(t, magicHome, userHome, "codex"); err == nil {
		t.Fatal("verify passed on a hand-edited shared instruction file")
	} else if !strings.Contains(err.Error(), "漂移") {
		t.Fatalf("verify error = %v, want a drift report", err)
	}

	// Undo apply's own write: the file did not exist before it, so rollback removes
	// it. The edit above makes the target the user's, so ask for --force.
	if _, err := runMagic(t, magicHome, cmdRollback(), "--home", userHome, "--force"); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("%s survived rollback (err = %v)", target, statErr)
	}
}
