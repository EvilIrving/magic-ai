package policy

import (
	"testing"

	"magic/internal/adapter"
)

func TestMatchAny(t *testing.T) {
	cases := []struct {
		rel      string
		patterns []string
		want     bool
	}{
		{"settings.json", []string{"settings.json"}, true},
		{"skills/foo/SKILL.md", []string{"skills/**"}, true},
		{"skills", []string{"skills/**"}, true},
		{"sessions/a.json", []string{"sessions/**"}, true},
		{"themes/dark.json", []string{"themes/**"}, true},
		{"statusline.sh", []string{"statusline.sh", "statusline.*"}, true},
		{"statusline.ps1", []string{"statusline.*"}, true},
		{"statusline-command.sh", []string{"statusline-command.sh", "statusline.*"}, true},
		{"other.sh", []string{"statusline.*"}, false},
		{"my-extra.toml", []string{"my-extra.toml"}, true},
		{"other.toml", []string{"settings.json", "skills/**"}, false},
		{"bin/x", []string{"bin/**"}, true},
		{"npm/pkg", []string{"npm/**"}, true},
	}
	for _, c := range cases {
		got := MatchAny(c.rel, c.patterns)
		if got != c.want {
			t.Errorf("MatchAny(%q, %v)=%v want %v", c.rel, c.patterns, got, c.want)
		}
	}
}

func TestClassifyPrecedence(t *testing.T) {
	builtin := adapter.CapturePolicy{
		Allow: []string{"settings.json", "skills/**", "themes/**"},
		Deny:  []string{"sessions/**", "bin/**"},
	}

	t.Run("builtin allow", func(t *testing.T) {
		d, r := Classify("settings.json", builtin, nil)
		if d != "allow" || r != "builtin.allow" {
			t.Fatalf("got %s %s", d, r)
		}
	})
	t.Run("builtin deny", func(t *testing.T) {
		d, r := Classify("sessions/x.json", builtin, nil)
		if d != "deny" || r != "builtin.deny" {
			t.Fatalf("got %s %s", d, r)
		}
	})
	t.Run("unknown default deny", func(t *testing.T) {
		d, r := Classify("my-extra.toml", builtin, nil)
		if d != "unknown" {
			t.Fatalf("got %s %s", d, r)
		}
	})
	t.Run("user.include beats unknown", func(t *testing.T) {
		ov := &OverlayHarness{Include: []string{"my-extra.toml"}}
		d, r := Classify("my-extra.toml", builtin, ov)
		if d != "allow" || r != "user.include" {
			t.Fatalf("got %s %s", d, r)
		}
	})
	t.Run("user.exclude beats builtin.allow", func(t *testing.T) {
		ov := &OverlayHarness{Exclude: []string{"themes/**"}}
		d, r := Classify("themes/dark.json", builtin, ov)
		if d != "deny" || r != "user.exclude" {
			t.Fatalf("got %s %s", d, r)
		}
	})
	t.Run("user.include beats builtin.deny", func(t *testing.T) {
		ov := &OverlayHarness{Include: []string{"sessions/keep.json"}}
		d, r := Classify("sessions/keep.json", builtin, ov)
		if d != "allow" || r != "user.include" {
			t.Fatalf("got %s %s", d, r)
		}
	})
	t.Run("user.exclude beats user.include", func(t *testing.T) {
		ov := &OverlayHarness{
			Include: []string{"settings.json"},
			Exclude: []string{"settings.json"},
		}
		d, r := Classify("settings.json", builtin, ov)
		if d != "deny" || r != "user.exclude" {
			t.Fatalf("got %s %s", d, r)
		}
	})
}

func TestOverlayApproveDeny(t *testing.T) {
	o := &Overlay{Harnesses: map[string]OverlayHarness{}}
	o.Approve("pi", "my-extra.toml")
	h := o.Harnesses["pi"]
	if len(h.Include) != 1 || h.Include[0] != "my-extra.toml" {
		t.Fatalf("include=%v", h.Include)
	}
	o.DenyPath("pi", "my-extra.toml")
	h = o.Harnesses["pi"]
	if containsString(h.Include, "my-extra.toml") {
		t.Fatal("should remove from include")
	}
	if !containsString(h.Exclude, "my-extra.toml") {
		t.Fatal("should add to exclude")
	}
}
