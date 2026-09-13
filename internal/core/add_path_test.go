package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"magic/internal/adapter"
	"magic/internal/adapters/claude"
	"magic/internal/adapters/pi"
)

func TestAddPathAutoDetectPiExtension(t *testing.T) {
	magicHome := t.TempDir()
	userHome := t.TempDir()
	src := filepath.Join(userHome, ".pi", "agent", "extensions", "catalog-allowlist.ts")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("export default function () {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := AddPath(AddPathOptions{
		MagicHome: magicHome,
		UserHome:  userHome,
		Source:    src,
		Adapters:  []adapter.Adapter{pi.New(), claude.New()},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Harness != "pi" || res.Rel != "extensions/catalog-allowlist.ts" {
		t.Fatalf("got %+v", res)
	}
	got, err := os.ReadFile(res.Dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "export default function () {}" {
		t.Fatalf("copied content: %q", got)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("source must stay: %v", err)
	}
}

func TestAddPathAsOverride(t *testing.T) {
	magicHome := t.TempDir()
	src := filepath.Join(t.TempDir(), "footer.ts")
	if err := os.WriteFile(src, []byte("// footer"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := AddPath(AddPathOptions{
		MagicHome: magicHome,
		Source:    src,
		As:        "pi/extensions/short-path-footer.ts",
		Adapters:  []adapter.Adapter{pi.New()},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(magicHome, "packs", "harness", "pi", "extensions", "short-path-footer.ts")
	if res.Dest != want {
		t.Fatalf("dest %s want %s", res.Dest, want)
	}
}

func TestAddPathUnknownWithoutAs(t *testing.T) {
	src := filepath.Join(t.TempDir(), "nope.ts")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := AddPath(AddPathOptions{
		MagicHome: t.TempDir(),
		UserHome:  t.TempDir(),
		Source:    src,
		Adapters:  []adapter.Adapter{pi.New()},
	})
	if err == nil || !strings.Contains(err.Error(), "--as") {
		t.Fatalf("want --as hint, got %v", err)
	}
}

func TestAddPathRejectsDirectory(t *testing.T) {
	dir := t.TempDir()
	_, err := AddPath(AddPathOptions{
		MagicHome: t.TempDir(),
		Source:    dir,
		As:        "pi/extensions",
		Adapters:  []adapter.Adapter{pi.New()},
	})
	if err == nil || !strings.Contains(err.Error(), "directory") {
		t.Fatalf("got %v", err)
	}
}
