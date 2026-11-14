package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedSkillLibrary writes skills with SKILL.md into $MAGIC_HOME/packs/skills.
func seedSkillLibrary(t *testing.T, magicHome string, names ...string) {
	t.Helper()
	for _, name := range names {
		dir := filepath.Join(magicHome, "packs", "skills", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# "+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// --shared and a project scope are two different layers; giving both is a usage
// error rather than a silent pick.
func TestSkillsSyncSharedRejectsProjectScope(t *testing.T) {
	out, err := runMagic(t, t.TempDir(), cmdSkills(),
		"sync", "--shared", "--scope", "project", "--home", t.TempDir())
	if err == nil {
		t.Fatalf("sync --shared --scope project succeeded, want refusal:\n%s", out)
	}
	if !strings.Contains(err.Error(), "--shared") {
		t.Errorf("error = %v, want it to name --shared", err)
	}
}

// End to end on temporary homes: --shared writes one link per library skill into
// ~/.agents/skills and no tool's own path, and a user's own entry there survives.
func TestSkillsSyncSharedWritesOnlySharedLayer(t *testing.T) {
	magicHome, home := t.TempDir(), t.TempDir()
	seedSkillLibrary(t, magicHome, "alpha", "beta")
	shared := filepath.Join(home, ".agents", "skills")
	if err := os.MkdirAll(filepath.Join(shared, "user-own"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shared, "user-own", "SKILL.md"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runMagic(t, magicHome, cmdSkills(), "sync", "--shared", "--home", home)
	if err != nil {
		t.Fatalf("sync --shared: %v\n%s", err, out)
	}
	if !strings.Contains(out, "shared layer: "+shared) {
		t.Errorf("output does not name the shared layer:\n%s", out)
	}
	for _, name := range []string{"alpha", "beta"} {
		if _, err := os.Lstat(filepath.Join(shared, name)); err != nil {
			t.Errorf("shared layer missing %s: %v\n%s", name, err, out)
		}
	}
	// The only skills directory under the temp home is the shared layer.
	var dirs []string
	err = filepath.WalkDir(home, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "skills" {
			dirs = append(dirs, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) != 1 || dirs[0] != shared {
		t.Errorf("skills dirs = %v, want only %s", dirs, shared)
	}
	if _, err := os.Stat(filepath.Join(shared, "user-own", "SKILL.md")); err != nil {
		t.Errorf("user entry in the shared layer was touched: %v", err)
	}
}

// --shared names one directory; --target names a harness's own primary path, so
// combining them is refused instead of silently ignoring the target.
func TestSkillsSyncSharedRejectsTarget(t *testing.T) {
	out, err := runMagic(t, t.TempDir(), cmdSkills(),
		"sync", "--shared", "--home", t.TempDir(), "--target", "codex")
	if err == nil {
		t.Fatalf("sync --shared --target codex succeeded, want refusal:\n%s", out)
	}
	if !strings.Contains(err.Error(), "--target") {
		t.Errorf("error = %v, want it to name --target", err)
	}
}

// --dry-run prints the plan for the shared layer without creating it.
func TestSkillsSyncSharedDryRun(t *testing.T) {
	magicHome, home := t.TempDir(), t.TempDir()
	seedSkillLibrary(t, magicHome, "alpha")

	out, err := runMagic(t, magicHome, cmdSkills(), "sync", "--shared", "--dry-run", "--home", home)
	if err != nil {
		t.Fatalf("sync --shared --dry-run: %v\n%s", err, out)
	}
	if _, err := os.Lstat(filepath.Join(home, ".agents", "skills", "alpha")); !os.IsNotExist(err) {
		t.Errorf("dry-run created a shared link (err %v)", err)
	}
	if !strings.Contains(out, "[dry-run] would link:") {
		t.Errorf("dry-run plan missing:\n%s", out)
	}
}

// The status row shows which shared directory backs an agent and whether it is
// that agent's own path, next to the link counts.
func TestSkillsStatusShowsSharedColumn(t *testing.T) {
	magicHome, home := t.TempDir(), t.TempDir()
	seedSkillLibrary(t, magicHome, "alpha")
	if out, err := runMagic(t, magicHome, cmdSkills(), "sync", "--home", home, "--target", "cline"); err != nil {
		t.Fatalf("sync: %v\n%s", err, out)
	}

	out, err := runMagic(t, magicHome, cmdSkills(), "status", "--home", home, "--target", "cline")
	if err != nil {
		t.Fatalf("status: %v\n%s", err, out)
	}
	if !strings.Contains(out, "shared +6, primary yes") {
		t.Errorf("status row missing the shared column:\n%s", out)
	}
	if !strings.Contains(out, "linked 1") {
		t.Errorf("status row lost its counts:\n%s", out)
	}
}

// The default target list names hermes and mimocode, which the upstream skills
// path table does not list. Their dir comes from their adapter, so status reports
// the row instead of failing the whole command.
func TestSkillsStatusReportsAgentsMissingFromUpstreamTable(t *testing.T) {
	magicHome, home := t.TempDir(), t.TempDir()
	seedSkillLibrary(t, magicHome, "alpha")

	out, err := runMagic(t, magicHome, cmdSkills(), "status", "--home", home)
	if err != nil {
		t.Fatalf("status: %v\n%s", err, out)
	}
	if strings.Contains(out, "unknown agent") {
		t.Errorf("status treats a table-less adapter as an error:\n%s", out)
	}
	for _, agent := range []string{"hermes", "mimocode"} {
		if !strings.Contains(out, "- "+agent+": no upstream entry") {
			t.Errorf("%s row missing the no-upstream-entry marker:\n%s", agent, out)
		}
	}
}

// --json on the default target list: an alias is the same tool, not another one,
// so claude shares with nobody and cline's list carries kimi-code once.
func TestSkillsStatusJSONDefaultAgents(t *testing.T) {
	magicHome, home := t.TempDir(), t.TempDir()
	seedSkillLibrary(t, magicHome, "alpha")

	type row struct {
		Agent           string   `json:"agent"`
		SharedWith      []string `json:"shared_with"`
		NoUpstreamEntry bool     `json:"no_upstream_entry"`
	}
	// One row per directory, so ask for the two agents whose dirs differ.
	out, err := runMagic(t, magicHome, cmdSkills(), "status", "--json", "--home", home, "--target", "claude,cline")
	if err != nil {
		t.Fatalf("status --json: %v\n%s", err, out)
	}
	var rows []row
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("decode: %v\n%s", err, out)
	}
	byAgent := map[string]row{}
	for _, r := range rows {
		byAgent[r.Agent] = r
	}
	if got := byAgent["claude"]; len(got.SharedWith) != 0 {
		t.Errorf("claude shared_with = %v, want [] (claude-code is the same tool)", got.SharedWith)
	}
	if got, ok := byAgent["cline"]; !ok || len(got.SharedWith) != 6 {
		t.Errorf("cline shared_with = %v, want the 6 other tools", got.SharedWith)
	} else {
		count := 0
		for _, id := range got.SharedWith {
			if id == "kimi-code" {
				count++
			}
			if id == "kimi-code-cli" {
				t.Errorf("cline shared_with carries the kimi-code alias: %v", got.SharedWith)
			}
		}
		if count != 1 {
			t.Errorf("cline shared_with lists kimi-code %d times, want 1: %v", count, got.SharedWith)
		}
	}

	// The default list carries hermes/mimocode, which have no table row.
	out, err = runMagic(t, magicHome, cmdSkills(), "status", "--json", "--home", home)
	if err != nil {
		t.Fatalf("status --json (default targets): %v\n%s", err, out)
	}
	rows = nil
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("decode: %v\n%s", err, out)
	}
	seen := map[string]bool{}
	for _, r := range rows {
		if r.NoUpstreamEntry {
			seen[r.Agent] = true
		}
	}
	for _, agent := range []string{"hermes", "mimocode"} {
		if !seen[agent] {
			t.Errorf("%s no_upstream_entry = false, want true:\n%s", agent, out)
		}
	}
}

// A harness that only reads the shared directory as a compatibility path says so.
func TestSkillsStatusShowsCompatReads(t *testing.T) {
	magicHome, home := t.TempDir(), t.TempDir()
	seedSkillLibrary(t, magicHome, "alpha")

	out, err := runMagic(t, magicHome, cmdSkills(), "status", "--home", home, "--target", "pi")
	if err != nil {
		t.Fatalf("status: %v\n%s", err, out)
	}
	if !strings.Contains(out, "compat reads: .agents/skills") {
		t.Errorf("status row missing compat reads:\n%s", out)
	}
	if !strings.Contains(out, "shared -, primary yes") {
		t.Errorf("pi owns its path but shares nothing:\n%s", out)
	}
}

// --json carries the same sharing facts; project scope omits them instead of
// claiming primary=false.
func TestSkillsStatusJSONSharedWith(t *testing.T) {
	magicHome, home := t.TempDir(), t.TempDir()
	seedSkillLibrary(t, magicHome, "alpha")
	if out, err := runMagic(t, magicHome, cmdSkills(), "sync", "--home", home, "--target", "cline"); err != nil {
		t.Fatalf("sync: %v\n%s", err, out)
	}

	type row struct {
		Scope      string   `json:"scope"`
		Agent      string   `json:"agent"`
		Dir        string   `json:"dir"`
		SharedWith []string `json:"shared_with"`
		Primary    *bool    `json:"primary"`
		Linked     int      `json:"linked"`
	}
	out, err := runMagic(t, magicHome, cmdSkills(), "status", "--json", "--home", home, "--target", "cline")
	if err != nil {
		t.Fatalf("status --json: %v\n%s", err, out)
	}
	var rows []row
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("decode: %v\n%s", err, out)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1:\n%s", len(rows), out)
	}
	got := rows[0]
	if want := filepath.Join(home, ".agents", "skills"); got.Dir != want {
		t.Errorf("dir = %q, want %q", got.Dir, want)
	}
	// .agents/skills has eight ids, one of them an alias of another, so six tools.
	if len(got.SharedWith) != 6 {
		t.Errorf("shared_with = %v, want the 6 other tools", got.SharedWith)
	}
	for _, id := range got.SharedWith {
		if id == "kimi-code-cli" {
			t.Errorf("shared_with carries the kimi-code alias: %v", got.SharedWith)
		}
	}
	if got.Primary == nil || !*got.Primary {
		t.Errorf("primary = %v, want true", got.Primary)
	}
	if got.Linked != 1 {
		t.Errorf("linked = %d, want 1", got.Linked)
	}
}
