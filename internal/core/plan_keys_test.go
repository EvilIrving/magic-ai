package core

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"magic/internal/adapter"
	"magic/internal/adapters/claude"
)

// The key diff is based on the disk: a target whose final bytes already matched
// is rewritten by nothing, so it reports no key change. Otherwise apply → verify
// would show "overwrite env.*" for keys apply itself wrote a moment ago.
func TestPlanUnchangedTargetReportsNoKeys(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeProviderLibrary(t, magicHome, "sk-live-abcdef123456")
	writeFile(t, filepath.Join(magicHome, "packs", "harness", "claude", "settings.json"),
		`{"env":{"ANTHROPIC_BASE_URL":"https://library.example"}}`)
	writeFile(t, filepath.Join(magicHome, "packs", "mcp", "mcp.yaml"), `mcp:
  docs:
    command: npx
`)
	opts := ApplyOptions{MagicHome: magicHome, UserHome: userHome, Only: "claude"}
	applyLayers(t, claude.New(), opts)

	p, err := BuildPlan([]adapter.Adapter{claude.New()}, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Adapters) != 1 || len(p.Adapters[0].Changes) == 0 {
		t.Fatalf("plan = %+v, want the applied targets", p.Adapters)
	}
	for _, c := range p.Adapters[0].Changes {
		if c.Action != ActionUnchanged {
			t.Fatalf("%s: action = %s, want unchanged", c.Rel, c.Action)
		}
		if len(c.Keys) != 0 {
			t.Fatalf("%s: unchanged target reports key changes: %+v", c.Rel, c.Keys)
		}
	}
}

// A target a layer removes is a delete: no content afterwards, and its disk
// content is what a backup has to hold. No harness apply projects today reaches
// it (codex is `provider use` only), so the change is checked directly.
func TestPlanChangeReportsDelete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "models.json")
	writeFile(t, path, `{"models":[{"slug":"example-model"}]}`)
	f := &planFile{
		path:   path,
		rel:    "models.json",
		mode:   0o600,
		disk:   []byte(`{"models":[{"slug":"example-model"}]}`),
		exists: true,
		remove: true,
		layers: []string{"provider 投影 models.json"},
	}
	c := f.change("codex")
	if c.Action != ActionDelete {
		t.Fatalf("action = %s, want delete", c.Action)
	}
	if c.After != "" {
		t.Fatalf("after = %s, want no fingerprint for a removed file", c.After)
	}
	if c.Before != FingerprintOf(f.disk) || !c.Backup {
		t.Fatalf("change = %+v, want the disk content backed up", c)
	}
	if len(c.Keys) != 0 || c.Lines != nil {
		t.Fatalf("change = %+v, want no keys/lines for a removed file", c)
	}
}

// Plan predicts every projection apply performs, so the old "uncovered
// projection" escape hatch must not come back: nothing in the tree may name it,
// in code or in the docs a reader checks.
func TestNoUncoveredProjectionLeft(t *testing.T) {
	needles := []string{"Unpred" + "icted", "未覆" + "盖"}
	root := filepath.Join("..", "..")
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		switch filepath.Ext(path) {
		case ".go", ".md", ".json":
		default:
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, needle := range needles {
			if strings.Contains(string(data), needle) {
				t.Errorf("%s still mentions %q: plan predicts this projection now", path, needle)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
