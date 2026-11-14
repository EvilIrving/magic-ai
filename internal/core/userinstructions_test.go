package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"magic/internal/adapter"
	"magic/internal/adapters/claude"
	"magic/internal/adapters/codex"
	"magic/internal/adapters/droid"
	"magic/internal/adapters/grok"
	"magic/internal/adapters/opencode"
	"magic/internal/adapters/pi"
)

// sharedHome is a user home that has every harness with a verified user-level
// instruction file installed, plus the XDG root OpenCode resolves under it.
func sharedHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	for _, dir := range []string{".codex", ".claude", ".factory", ".pi/agent", ".grok", ".config/opencode"} {
		if err := os.MkdirAll(filepath.Join(home, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return home
}

// sharedTargets is the verified user-level instruction file of each harness, the
// path the projection must land on. A new entry here is a claim that the path was
// checked against the harness's own docs or source.
func sharedTargets(t *testing.T, userHome string) []struct {
	adapter adapter.Adapter
	target  string
} {
	t.Helper()
	return []struct {
		adapter adapter.Adapter
		target  string
	}{
		{codex.New(), filepath.Join(userHome, ".codex", "AGENTS.md")},
		{claude.New(), filepath.Join(userHome, ".claude", "CLAUDE.md")},
		{droid.New(), filepath.Join(userHome, ".factory", "AGENTS.md")},
		{grok.New(), filepath.Join(userHome, ".grok", "AGENTS.md")},
		{opencode.New(), filepath.Join(userHome, ".config", "opencode", "AGENTS.md")},
		{pi.New(), filepath.Join(userHome, ".pi", "agent", "AGENTS.md")},
	}
}

// A library that holds only packs/AGENTS.md puts the same bytes in every
// harness's user-level instruction file.
func TestApplyProjectsSharedInstructionsToEveryVerifiedHome(t *testing.T) {
	magicHome, userHome := t.TempDir(), sharedHome(t)
	const body = "# Shared rules\n\n- Prefer the library.\n"
	writeFile(t, filepath.Join(magicHome, "packs", "AGENTS.md"), body)

	for _, c := range sharedTargets(t, userHome) {
		if _, err := ApplyAdapter(c.adapter, ApplyOptions{MagicHome: magicHome, UserHome: userHome}, ""); err != nil {
			t.Fatalf("%s: %v", c.adapter.ID(), err)
		}
		if got := readFile(t, c.target); got != body {
			t.Errorf("%s target %s = %q, want the shared content", c.adapter.ID(), c.target, got)
		}
	}
}

// The per-harness file is the user's own decision for that harness: it wins over
// the shared source, at the file name that harness actually reads (Claude Code
// reads CLAUDE.md, not AGENTS.md).
func TestPerHarnessInstructionsWinOverShared(t *testing.T) {
	cases := []struct {
		a        adapter.Adapter
		libName  string
		homeFile []string
	}{
		{codex.New(), "AGENTS.md", []string{".codex", "AGENTS.md"}},
		{claude.New(), "CLAUDE.md", []string{".claude", "CLAUDE.md"}},
	}
	for _, c := range cases {
		t.Run(string(c.a.ID()), func(t *testing.T) {
			magicHome, userHome := t.TempDir(), sharedHome(t)
			writeFile(t, filepath.Join(magicHome, "packs", "AGENTS.md"), "# shared\n")
			harnessBody := "# " + string(c.a.ID()) + " only\n"
			writeFile(t, filepath.Join(magicHome, c.a.LibraryMirrorRel(), c.libName), harnessBody)

			if _, err := ApplyAdapter(c.a, ApplyOptions{MagicHome: magicHome, UserHome: userHome}, ""); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(append([]string{userHome}, c.homeFile...)...)
			if got := readFile(t, target); got != harnessBody {
				t.Fatalf("%s = %q, want the per-harness content", target, got)
			}
		})
	}
}

// A per-harness file that holds only a comment is still the user's file for that
// harness: ApplyAdapter writes it. Only the shared source is treated as an
// unwritten placeholder.
func TestPerHarnessCommentOnlyFileIsStillWritten(t *testing.T) {
	magicHome, userHome := t.TempDir(), sharedHome(t)
	const harnessBody = "<!-- my own notes -->\n"
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "pi", "AGENTS.md"), harnessBody)
	writeFile(t, filepath.Join(magicHome, "packs", "AGENTS.md"), "# shared\n")

	if _, err := ApplyAdapter(pi.New(), ApplyOptions{MagicHome: magicHome, UserHome: userHome}, ""); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(userHome, ".pi", "agent", "AGENTS.md")
	if got := readFile(t, target); got != harnessBody {
		t.Fatalf("%s = %q, want the per-harness content", target, got)
	}
}

// The init placeholder holds no instruction, so apply writes no target file. One
// line of the user's own content turns the projection on: the placeholder rule
// must not become a silent way to never project.
func TestSharedInstructionsPlaceholderWritesNothing(t *testing.T) {
	magicHome, userHome := t.TempDir(), sharedHome(t)
	writeFile(t, filepath.Join(magicHome, "packs", "AGENTS.md"), sharedInstructionsPlaceholder)

	for _, c := range sharedTargets(t, userHome) {
		if _, err := ApplyAdapter(c.adapter, ApplyOptions{MagicHome: magicHome, UserHome: userHome}, ""); err != nil {
			t.Fatalf("%s: %v", c.adapter.ID(), err)
		}
		if _, err := os.Stat(c.target); !os.IsNotExist(err) {
			t.Errorf("%s target %s was written for a placeholder source (err = %v)", c.adapter.ID(), c.target, err)
		}
	}

	const withContent = sharedInstructionsPlaceholder + "\n# 共享规则\n"
	writeFile(t, filepath.Join(magicHome, "packs", "AGENTS.md"), withContent)
	if _, err := ApplyAdapter(codex.New(), ApplyOptions{MagicHome: magicHome, UserHome: userHome}, ""); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(userHome, ".codex", "AGENTS.md")); got != withContent {
		t.Fatalf("target = %q, want the source with one content line", got)
	}
}

// hasInstructionText decides whether the shared source is written at all, so the
// rule it encodes is pinned: blank lines and HTML comments are not instructions,
// a markdown heading is, and an inline comment does not hide the text around it.
func TestHasInstructionText(t *testing.T) {
	cases := map[string]bool{
		"":                                   false,
		"\n\n  \n\t\n":                       false,
		"<!-- just a note -->\n":             false,
		"<!--\nmulti\nline\n-->\n":           false,
		"<!--\nunterminated\n":               false,
		"\n<!-- x -->\n\n<!-- y -->\n":       false,
		"# 项目约定\n":                           true,
		"<!-- note -->\nrun the tests\n":     true,
		"<!-- a --> text <!-- b -->\n":       true,
		"  <!-- indented note -->  \n":       false,
		"a line <!-- an inline note -->\n":   true,
		"<!--\nnote\n-->\n## Build\n```sh\n": true,
	}
	for src, want := range cases {
		if got := hasInstructionText([]byte(src)); got != want {
			t.Errorf("hasInstructionText(%q) = %v, want %v", src, got, want)
		}
	}
}

// A harness that is not installed gets nothing: writing its config file would
// invent an installation and create a directory the user never had. --only names
// the harness explicitly and bypasses the gate.
func TestSharedInstructionsSkipUninstalledHarness(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(userHome, ".config"))
	writeFile(t, filepath.Join(magicHome, "packs", "AGENTS.md"), "# shared\n")

	if _, err := ApplyAdapter(codex.New(), ApplyOptions{MagicHome: magicHome, UserHome: userHome}, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(userHome, ".codex")); !os.IsNotExist(err) {
		t.Fatalf("apply created ~/.codex for an uninstalled harness (err = %v)", err)
	}

	opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome, Only: "codex"}
	if _, err := ApplyAdapter(codex.New(), opts, ""); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(userHome, ".codex", "AGENTS.md")); got != "# shared\n" {
		t.Fatalf("--only target = %q, want the shared content", got)
	}
}

// The plan lists the shared write under the path relative to the user home, and
// what it predicts is what apply leaves on disk: the second plan says unchanged.
func TestPlanNamesSharedInstructionFileByUserHomeRel(t *testing.T) {
	magicHome, userHome := t.TempDir(), sharedHome(t)
	const body = "# shared\n"
	writeFile(t, filepath.Join(magicHome, "packs", "AGENTS.md"), body)
	opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome}

	p, err := BuildPlan([]adapter.Adapter{codex.New(), claude.New()}, opts)
	if err != nil {
		t.Fatal(err)
	}
	codexChange := findChange(t, p, "codex", ".codex/AGENTS.md")
	if codexChange.Action != ActionCreate {
		t.Fatalf("codex action = %q, want create", codexChange.Action)
	}
	if codexChange.After != FingerprintOf([]byte(body)) {
		t.Fatalf("codex after = %q, want the shared fingerprint", codexChange.After)
	}
	if !strings.Contains(codexChange.Reason, SharedInstructionsLayerLabel) {
		t.Fatalf("codex reason = %q, want the shared layer label", codexChange.Reason)
	}
	if c := findChange(t, p, "claude", ".claude/CLAUDE.md"); c.Action != ActionCreate {
		t.Fatalf("claude action = %q, want create", c.Action)
	}

	for _, a := range []adapter.Adapter{codex.New(), claude.New()} {
		if _, err := ApplyAdapter(a, opts, ""); err != nil {
			t.Fatal(err)
		}
	}
	p, err = BuildPlan([]adapter.Adapter{codex.New(), claude.New()}, opts)
	if err != nil {
		t.Fatal(err)
	}
	if c := findChange(t, p, "codex", ".codex/AGENTS.md"); c.Action != ActionUnchanged {
		t.Fatalf("codex action after apply = %q, want unchanged", c.Action)
	}
	if c := findChange(t, p, "claude", ".claude/CLAUDE.md"); c.Action != ActionUnchanged {
		t.Fatalf("claude action after apply = %q, want unchanged", c.Action)
	}
}

// An existing file with different content is backed up and overwritten like every
// other apply write, and rollback puts it back. A file apply created is removed.
func TestRollbackUndoesSharedInstructionFile(t *testing.T) {
	magicHome, userHome := t.TempDir(), sharedHome(t)
	writeFile(t, filepath.Join(magicHome, "packs", "AGENTS.md"), "# shared\n")

	// create case: the file did not exist before apply. The stamp is explicit so
	// the two sets below cannot collide inside one second.
	backupDir := filepath.Join(magicHome, "backups", "20260101T000000Z")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatal(err)
	}
	opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome, BackupDir: backupDir}
	if _, err := ApplyAdapter(codex.New(), opts, backupDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(backupDir, "codex", ".codex", "AGENTS.md")); err == nil {
		t.Fatal("a create entry copied the target it did not find")
	}
	res, err := Rollback(nil, RollbackOptions{MagicHome: magicHome, UserHome: userHome})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Restored) != 1 || !res.Restored[0].Delete || res.Restored[0].Rel != ".codex/AGENTS.md" {
		t.Fatalf("restored = %+v, want one delete of .codex/AGENTS.md", res.Restored)
	}
	if _, err := os.Stat(filepath.Join(userHome, ".codex", "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("rollback left the created file (err = %v)", err)
	}

	// update case: the user's own file is restored from the backup set.
	target := filepath.Join(userHome, ".codex", "AGENTS.md")
	writeFile(t, target, "user's own rules\n")
	backupDir = filepath.Join(magicHome, "backups", "20260101T000001Z")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatal(err)
	}
	opts = ApplyOptions{MagicHome: magicHome, UserHome: userHome, BackupDir: backupDir}
	if _, err := ApplyAdapter(codex.New(), opts, backupDir); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, target); got != "# shared\n" {
		t.Fatalf("target after apply = %q, want the shared content", got)
	}
	if got := readFile(t, filepath.Join(backupDir, "codex", ".codex", "AGENTS.md")); got != "user's own rules\n" {
		t.Fatalf("backup copy = %q, want the pre-apply content", got)
	}
	res, err = Rollback(nil, RollbackOptions{MagicHome: magicHome, UserHome: userHome})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Restored) != 1 || res.Restored[0].Delete {
		t.Fatalf("restored = %+v, want one restore", res.Restored)
	}
	if got := readFile(t, target); got != "user's own rules\n" {
		t.Fatalf("target after rollback = %q, want the user's content", got)
	}
}

// A drift on the shared file is the same drift as any other apply write: plan
// reports it as an update against the disk content.
func TestPlanSeesDriftOnSharedInstructionFile(t *testing.T) {
	magicHome, userHome := t.TempDir(), sharedHome(t)
	writeFile(t, filepath.Join(magicHome, "packs", "AGENTS.md"), "# shared\n")
	target := filepath.Join(userHome, ".codex", "AGENTS.md")
	writeFile(t, target, "# edited by hand\n")
	opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome}

	p, err := BuildPlan([]adapter.Adapter{codex.New()}, opts)
	if err != nil {
		t.Fatal(err)
	}
	c := findChange(t, p, "codex", ".codex/AGENTS.md")
	if c.Action != ActionUpdate {
		t.Fatalf("action = %q, want update", c.Action)
	}
	if c.Before != FingerprintOf([]byte("# edited by hand\n")) {
		t.Fatalf("before = %q, want the disk fingerprint", c.Before)
	}
	if !c.Backup {
		t.Fatal("an update of an existing file must be backed up")
	}
}
