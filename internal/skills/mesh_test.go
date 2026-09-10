package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSkill(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# s\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSyncGlobalCreatesManagedLinks(t *testing.T) {
	tmp := t.TempDir()
	magic := filepath.Join(tmp, ".magic-ai")
	home := filepath.Join(tmp, "home")
	writeSkill(t, filepath.Join(magic, "packs", "skills", "alpha"))

	r, err := Sync(Options{
		MagicHome: magic,
		UserHome:  home,
		Scope:     ScopeGlobal,
		Agents:    []string{"claude", "pi"},
	})
	if err != nil {
		t.Fatalf("sync: %v\n%s", err, strings.Join(r.Lines, "\n"))
	}
	claude := filepath.Join(home, ".claude", "skills", "alpha")
	pi := filepath.Join(home, ".pi", "agent", "skills", "alpha")
	for _, p := range []string{claude, pi} {
		fi, err := os.Lstat(p)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("%s not symlink", p)
		}
	}
	if r.Stats.Created < 2 {
		t.Fatalf("created=%d want >=2", r.Stats.Created)
	}
}

func TestStatusReportsConflictRealDir(t *testing.T) {
	tmp := t.TempDir()
	magic := filepath.Join(tmp, ".magic-ai")
	home := filepath.Join(tmp, "home")
	writeSkill(t, filepath.Join(magic, "packs", "skills", "alpha"))
	real := filepath.Join(home, ".claude", "skills", "alpha")
	_ = os.MkdirAll(real, 0o755)
	_ = os.WriteFile(filepath.Join(real, "keep.txt"), []byte("x"), 0o644)

	r, err := Status(Options{
		MagicHome: magic,
		UserHome:  home,
		Scope:     ScopeGlobal,
		Agents:    []string{"claude"},
	})
	if err != nil && r.Stats.Issues == 0 {
		t.Fatalf("unexpected err: %v", err)
	}
	out := strings.Join(r.Lines, "\n")
	if !strings.Contains(out, "conflict") {
		t.Fatalf("expected conflict in status:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(real, "keep.txt")); err != nil {
		t.Fatal("real dir damaged")
	}
}

func TestSyncNeverDeletesRealDir(t *testing.T) {
	tmp := t.TempDir()
	magic := filepath.Join(tmp, ".magic-ai")
	home := filepath.Join(tmp, "home")
	writeSkill(t, filepath.Join(magic, "packs", "skills", "alpha"))
	real := filepath.Join(home, ".claude", "skills", "alpha")
	_ = os.MkdirAll(real, 0o755)
	_ = os.WriteFile(filepath.Join(real, "keep.txt"), []byte("x"), 0o644)

	r, err := Sync(Options{
		MagicHome: magic,
		UserHome:  home,
		Scope:     ScopeGlobal,
		Agents:    []string{"claude"},
		Force:     true,
	})
	if r == nil {
		t.Fatal(err)
	}
	if r.Stats.Conflicts < 1 {
		t.Fatalf("want conflict, stats=%+v lines=\n%s", r.Stats, strings.Join(r.Lines, "\n"))
	}
	if _, err := os.Stat(filepath.Join(real, "keep.txt")); err != nil {
		t.Fatal("real dir damaged")
	}
}

func TestProjectSyncLayout(t *testing.T) {
	tmp := t.TempDir()
	magic := filepath.Join(tmp, ".magic-ai")
	_ = os.MkdirAll(filepath.Join(magic, "packs", "skills"), 0o755)
	proj := filepath.Join(tmp, "proj")
	writeSkill(t, filepath.Join(proj, ".agents", "skills", "beta"))

	r, err := Sync(Options{
		MagicHome:   magic,
		UserHome:    filepath.Join(tmp, "home"),
		Scope:       ScopeProject,
		ProjectRoot: proj,
		Agents:      []string{"claude", "cursor", "pi"},
	})
	if err != nil {
		t.Fatalf("%v\n%s", err, strings.Join(r.Lines, "\n"))
	}
	for _, agent := range []string{"claude", "cursor", "pi"} {
		p := filepath.Join(proj, "."+agent, "skills", "beta")
		if _, err := os.Lstat(p); err != nil {
			t.Fatalf("%s: %v", p, err)
		}
	}
}

func TestIncludeGlobalProjectOverride(t *testing.T) {
	tmp := t.TempDir()
	magic := filepath.Join(tmp, ".magic-ai")
	writeSkill(t, filepath.Join(magic, "packs", "skills", "shared"))
	writeSkill(t, filepath.Join(magic, "packs", "skills", "only-global"))
	proj := filepath.Join(tmp, "proj")
	writeSkill(t, filepath.Join(proj, ".agents", "skills", "shared")) // override

	r, err := List(Options{
		MagicHome:     magic,
		Scope:         ScopeProject,
		ProjectRoot:   proj,
		IncludeGlobal: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	out := strings.Join(r.Lines, "\n")
	if !strings.Contains(out, "only-global (inherited global)") {
		t.Fatalf("missing inherited:\n%s", out)
	}
	if strings.Contains(out, "shared (inherited") {
		t.Fatalf("shared should be project-local:\n%s", out)
	}
}

func TestImportAndDoctor(t *testing.T) {
	tmp := t.TempDir()
	magic := filepath.Join(tmp, ".magic-ai")
	home := filepath.Join(tmp, "home")
	ext := filepath.Join(tmp, "ext-skill")
	writeSkill(t, ext)

	r, err := ImportPath(Options{
		MagicHome: magic,
		UserHome:  home,
		Scope:     ScopeGlobal,
		Agents:    []string{"grok"},
		SyncAfter: true,
	}, ext)
	if err != nil {
		t.Fatalf("%v\n%s", err, strings.Join(r.Lines, "\n"))
	}
	link := filepath.Join(magic, "packs", "skills", "ext-skill")
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("import link missing: %v", err)
	}
	dr, err := Doctor(Options{
		MagicHome: magic,
		UserHome:  home,
		Scope:     ScopeGlobal,
		Agents:    []string{"grok"},
	})
	if err != nil {
		t.Fatalf("doctor: %v\n%s", err, strings.Join(dr.Lines, "\n"))
	}
}

func TestCleanStaleManaged(t *testing.T) {
	tmp := t.TempDir()
	magic := filepath.Join(tmp, ".magic-ai")
	home := filepath.Join(tmp, "home")
	_ = os.MkdirAll(filepath.Join(magic, "packs", "skills"), 0o755)
	targetDir := filepath.Join(home, ".claude", "skills")
	_ = os.MkdirAll(targetDir, 0o755)
	gone := filepath.Join(magic, "packs", "skills", "gone")
	link := filepath.Join(targetDir, "gone")
	if err := os.Symlink(gone, link); err != nil {
		t.Fatal(err)
	}
	r, err := Clean(Options{
		MagicHome: magic,
		UserHome:  home,
		Scope:     ScopeGlobal,
		Agents:    []string{"claude"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("stale link should be removed; stats=%+v", r.Stats)
	}
}
