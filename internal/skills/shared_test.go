package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedSkills writes skills with SKILL.md into the global library and returns the
// library path.
func seedSkills(t *testing.T, magicHome string, names ...string) string {
	t.Helper()
	central := CentralDir(magicHome)
	for _, name := range names {
		dir := filepath.Join(central, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# "+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return central
}

// assertInstalled accepts both install modes EnsureLink can pick: a link to the
// source, or (when symlinks are unavailable, e.g. Windows without developer
// mode) a copy carrying the skill content.
func assertInstalled(t *testing.T, entry, source string) {
	t.Helper()
	fi, err := os.Lstat(entry)
	if err != nil {
		t.Fatalf("%s not installed: %v", entry, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		dest, err := LinkDestination(entry)
		if err != nil {
			t.Fatalf("%s: %v", entry, err)
		}
		want, _ := LexicalAbs(source, "")
		if dest != want {
			t.Errorf("%s -> %s, want %s", entry, dest, want)
		}
		return
	}
	if _, err := os.Stat(filepath.Join(entry, "SKILL.md")); err != nil {
		t.Errorf("%s is neither a link to %s nor a copy of it", entry, source)
	}
}

// --shared writes the shared layer and nothing else: the agent primary paths are
// plain `magic skills sync`'s job, so not a single one may appear here.
func TestSyncSharedWritesOnlySharedLayer(t *testing.T) {
	magicHome, home := t.TempDir(), t.TempDir()
	central := seedSkills(t, magicHome, "alpha", "beta")

	r, err := SyncShared(Options{
		MagicHome: magicHome,
		UserHome:  home,
		Scope:     ScopeGlobal,
	})
	if err != nil {
		t.Fatalf("SyncShared: %v\n%s", err, strings.Join(r.Lines, "\n"))
	}
	out := strings.Join(r.Lines, "\n")
	shared := SharedTargetDir(home)
	if !strings.Contains(out, "shared layer: "+shared) {
		t.Errorf("output must name the shared layer, got:\n%s", out)
	}
	for _, name := range []string{"alpha", "beta"} {
		assertInstalled(t, filepath.Join(shared, name), filepath.Join(central, name))
	}
	// One directory, one link per skill -- that is the whole run.
	if r.Stats.Created != 2 {
		t.Errorf("Created = %d, want 2 (one shared layer, no primary paths)\n%s", r.Stats.Created, out)
	}
	for _, rel := range []string{".claude", ".codex", ".pi/agent", ".cursor", ".copilot", ".trae", ".qoder", ".config/opencode", ".grok"} {
		if _, err := os.Lstat(filepath.Join(home, rel, "skills")); !os.IsNotExist(err) {
			t.Errorf("--shared wrote another tool's primary path %s (err %v)", rel, err)
		}
	}
}

// --shared names one directory; --target names an agent's own primary dir, so the
// two cannot be combined (and must say so instead of ignoring the target).
func TestSyncSharedRejectsTarget(t *testing.T) {
	magicHome, home := t.TempDir(), t.TempDir()
	seedSkills(t, magicHome, "alpha")
	_, err := SyncShared(Options{
		MagicHome: magicHome,
		UserHome:  home,
		Scope:     ScopeGlobal,
		Agents:    []string{"codex"},
	})
	if err == nil {
		t.Fatal("SyncShared with Agents succeeded, want a --target refusal")
	}
	if !strings.Contains(err.Error(), "--target") {
		t.Errorf("error = %v, want it to name --target", err)
	}
}

// The shared directory belongs to other tools: entries Magic did not create are
// reported one by one and left alone, and nothing is pruned.
func TestSyncSharedKeepsUserEntriesInSharedLayer(t *testing.T) {
	magicHome, home := t.TempDir(), t.TempDir()
	seedSkills(t, magicHome, "alpha", "beta")
	shared := SharedTargetDir(home)
	if err := os.MkdirAll(shared, 0o755); err != nil {
		t.Fatal(err)
	}
	// A skill of the user's own that the library does not have.
	userOwn := filepath.Join(shared, "user-own")
	if err := os.MkdirAll(userOwn, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userOwn, "SKILL.md"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A real directory whose name collides with a library skill.
	collide := filepath.Join(shared, "alpha")
	if err := os.MkdirAll(collide, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(collide, "SKILL.md"), []byte("user copy"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A symlink of the user's own whose name collides with another library skill.
	elsewhere := filepath.Join(t.TempDir(), "other-skill")
	if err := os.MkdirAll(elsewhere, 0o755); err != nil {
		t.Fatal(err)
	}
	foreignLink := filepath.Join(shared, "beta")
	if err := os.Symlink(elsewhere, foreignLink); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	r, err := SyncShared(Options{
		MagicHome: magicHome,
		UserHome:  home,
		Scope:     ScopeGlobal,
	})
	// Conflicts are real findings, so the command reports them like sync does.
	if err == nil {
		t.Error("expected the conflicts to be reported (non-zero exit)")
	}
	if r.Stats.Conflicts != 2 {
		t.Errorf("Conflicts = %d, want 2\n%s", r.Stats.Conflicts, strings.Join(r.Lines, "\n"))
	}
	out := strings.Join(r.Lines, "\n")
	if !strings.Contains(out, "shared: keep real file/dir "+collide) {
		t.Errorf("colliding real dir not reported:\n%s", out)
	}
	if !strings.Contains(out, "shared: keep existing link "+foreignLink) {
		t.Errorf("colliding external link not reported:\n%s", out)
	}

	// Nothing was overwritten or removed.
	if b, err := os.ReadFile(filepath.Join(collide, "SKILL.md")); err != nil || string(b) != "user copy" {
		t.Errorf("user dir rewritten: %q (err %v)", b, err)
	}
	if dest, err := LinkDestination(foreignLink); err != nil || dest != elsewhere {
		t.Errorf("foreign link rewritten: %s (err %v)", dest, err)
	}
	if b, err := os.ReadFile(filepath.Join(userOwn, "SKILL.md")); err != nil || string(b) != "mine" {
		t.Errorf("unrelated user entry touched: %q (err %v)", b, err)
	}
}

// --dry-run reports the same plan without touching the shared directory.
func TestSyncSharedDryRunWritesNothing(t *testing.T) {
	magicHome, home := t.TempDir(), t.TempDir()
	seedSkills(t, magicHome, "alpha")

	r, err := SyncShared(Options{
		MagicHome: magicHome,
		UserHome:  home,
		Scope:     ScopeGlobal,
		DryRun:    true,
	})
	if err != nil {
		t.Fatalf("SyncShared dry-run: %v\n%s", err, strings.Join(r.Lines, "\n"))
	}
	shared := SharedTargetDir(home)
	if _, err := os.Lstat(filepath.Join(shared, "alpha")); !os.IsNotExist(err) {
		t.Errorf("dry-run created %s (err %v)", filepath.Join(shared, "alpha"), err)
	}
	if !strings.Contains(strings.Join(r.Lines, "\n"), "[dry-run] would link: "+filepath.Join(shared, "alpha")) {
		t.Errorf("dry-run plan missing:\n%s", strings.Join(r.Lines, "\n"))
	}
}

// The shared layer is a global-scope project; combining it with a project scope
// has no meaning.
func TestSyncSharedRejectsNonGlobalScope(t *testing.T) {
	for _, scope := range []Scope{ScopeProject, ScopeAll} {
		_, err := SyncShared(Options{MagicHome: t.TempDir(), UserHome: t.TempDir(), Scope: scope})
		if err == nil || !strings.Contains(err.Error(), "--shared") {
			t.Errorf("scope %s: err = %v, want a --shared refusal", scope, err)
		}
	}
}
