package core_test

import (
	"os"
	"path/filepath"
	"testing"

	"magic/internal/adapters/pi"
	"magic/internal/core"
	"magic/internal/policy"
)

func TestCaptureTriageUnknownThenApprove(t *testing.T) {
	tmp := t.TempDir()
	userHome := filepath.Join(tmp, "home")
	magicHome := filepath.Join(tmp, "lib")
	piRoot := filepath.Join(userHome, ".pi", "agent")
	_ = os.MkdirAll(filepath.Join(piRoot, "skills", "x"), 0o755)
	_ = os.WriteFile(filepath.Join(piRoot, "settings.json"), []byte(`{"a":1}`), 0o644)
	_ = os.WriteFile(filepath.Join(piRoot, "my-extra.toml"), []byte("x=1\n"), 0o644)
	_ = os.MkdirAll(filepath.Join(piRoot, "sessions"), 0o755)
	_ = os.WriteFile(filepath.Join(piRoot, "sessions", "s.json"), []byte("{}"), 0o644)

	if err := core.InitSkeleton(magicHome); err != nil {
		t.Fatal(err)
	}
	a := pi.New()
	ov, err := policy.LoadOverlay(core.NewPaths(magicHome).Overlay)
	if err != nil {
		t.Fatal(err)
	}
	report, _, err := core.CaptureAdapter(a, ov, core.CaptureOptions{
		MagicHome: magicHome, UserHome: userHome, DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Allow) < 1 {
		t.Fatalf("expected allow settings, got %+v", report.Allow)
	}
	foundUnknown := false
	for _, u := range report.Unknown {
		if u.Rel == "my-extra.toml" {
			foundUnknown = true
		}
	}
	if !foundUnknown {
		t.Fatalf("expected my-extra.toml unknown: %+v", report.Unknown)
	}
	foundDeny := false
	for _, d := range report.Deny {
		if d.Rel == "sessions/s.json" {
			foundDeny = true
		}
	}
	if !foundDeny {
		t.Fatalf("expected sessions deny: %+v", report.Deny)
	}

	// approve and capture for real
	ov.Approve("pi", "my-extra.toml")
	if err := policy.SaveOverlay(core.NewPaths(magicHome).Overlay, ov); err != nil {
		t.Fatal(err)
	}
	ov2, _ := policy.LoadOverlay(core.NewPaths(magicHome).Overlay)
	report2, copied, err := core.CaptureAdapter(a, ov2, core.CaptureOptions{
		MagicHome: magicHome, UserHome: userHome, DryRun: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	allowedExtra := false
	for _, v := range report2.Allow {
		if v.Rel == "my-extra.toml" && v.Reason == "user.include" {
			allowedExtra = true
		}
	}
	if !allowedExtra {
		t.Fatalf("after approve, my-extra should allow: %+v", report2.Allow)
	}
	dst := filepath.Join(magicHome, "packs", "harness", "pi", "my-extra.toml")
	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("expected copied file %s among %v: %v", dst, copied, err)
	}
}
