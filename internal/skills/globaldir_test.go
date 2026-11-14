package skills

import (
	"path/filepath"
	"reflect"
	"testing"
)

// The .agents/skills group in the table is eight ids (D11, verbatim). A member's
// SharedWith lists the other *tools* alphabetically and never includes itself,
// which is what tells a user "my other tools read this directory too".
// `kimi-code-cli` is an alias of `kimi-code`, so the group collapses to six.
func TestGlobalDirInfoSharedWith(t *testing.T) {
	home := t.TempDir()
	info, err := GlobalDirInfo("cline", home)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".agents", "skills"); info.Path != want {
		t.Errorf("Path = %q, want %q", info.Path, want)
	}
	want := []string{"dexto", "kimi-code", "loaf", "sarvam-code", "warp", "zed"}
	if !reflect.DeepEqual(info.SharedWith, want) {
		t.Errorf("SharedWith = %v, want %v", info.SharedWith, want)
	}
	if !info.Primary {
		t.Error("cline's adapter SkillsDir is .agents/skills: Primary = false")
	}
}

// An alias is the same harness under a second name, not a second tool reading the
// directory: it must not show up as shared, and the canonical id must appear once.
func TestGlobalDirInfoSharedWithCollapsesAliases(t *testing.T) {
	home := t.TempDir()
	cases := []struct {
		agent string
		want  []string
	}{
		{"claude", nil},      // claude-code is the same tool
		{"claude-code", nil}, // and the other way round
		{"workbuddy", nil},   // D15: workbuddy = CodeBuddy
		{"codebuddy", nil},   // D15: same product, same dir
		// kimi-code-cli is not another tool either: both ids report the same six.
		{"kimi-code", []string{"cline", "dexto", "loaf", "sarvam-code", "warp", "zed"}},
		{"kimi-code-cli", []string{"cline", "dexto", "loaf", "sarvam-code", "warp", "zed"}},
	}
	for _, c := range cases {
		info, err := GlobalDirInfo(c.agent, home)
		if err != nil {
			t.Fatalf("%s: %v", c.agent, err)
		}
		if !reflect.DeepEqual(info.SharedWith, c.want) {
			t.Errorf("%s: SharedWith = %v, want %v", c.agent, info.SharedWith, c.want)
		}
		for _, id := range info.SharedWith {
			if id == canonicalAgent(c.agent) {
				t.Errorf("%s: SharedWith lists its own canonical id %q", c.agent, id)
			}
			if id != canonicalAgent(id) {
				t.Errorf("%s: SharedWith carries a non-canonical alias %q", c.agent, id)
			}
		}
	}
}

// Primary asks whether the directory is the tool's own main path: the adapter's
// SkillsDir compared with the table entry. pi/grok/codex own their path and are
// primary; a shared entry with no adapter of its own is not.
func TestGlobalDirInfoPrimary(t *testing.T) {
	home := t.TempDir()
	cases := []struct {
		agent string
		want  bool
	}{
		{"cline", true},          // .agents/skills, table entry == adapter
		{"kimi-code", true},      // .agents/skills, table entry == adapter
		{"zed", true},            // .agents/skills, table entry == adapter
		{"pi", true},             // .pi/agent/skills, own path
		{"grok", true},           // .grok/skills, own path
		{"codex", true},          // .codex/skills, own path
		{"workbuddy", false},     // adapter declares no skills dir
		{"kimi-code-cli", false}, // no adapter of its own
		{"loaf", false},          // no adapter of its own
	}
	for _, c := range cases {
		info, err := GlobalDirInfo(c.agent, home)
		if err != nil {
			t.Fatalf("%s: %v", c.agent, err)
		}
		if info.Primary != c.want {
			t.Errorf("%s: Primary = %v, want %v (path %s)", c.agent, info.Primary, c.want, info.Path)
		}
	}
}

// CompatReads is documentation evidence only: it neither moves Primary nor
// applies to harnesses with no recorded extra directory.
func TestGlobalDirInfoCompatReads(t *testing.T) {
	home := t.TempDir()
	cases := map[string][]string{
		"pi":    {".agents/skills"},
		"grok":  {".agents/skills", ".claude/skills", ".cursor/skills"},
		"codex": {".agents/skills"},
		"cline": nil, // the shared directory *is* its path, not an extra one
	}
	for agent, want := range cases {
		info, err := GlobalDirInfo(agent, home)
		if err != nil {
			t.Fatalf("%s: %v", agent, err)
		}
		if !reflect.DeepEqual(info.CompatReads, want) {
			t.Errorf("%s: CompatReads = %v, want %v", agent, info.CompatReads, want)
		}
		if !info.Primary {
			t.Errorf("%s: CompatReads must not affect Primary", agent)
		}
	}
}

func TestGlobalDirInfoUnknownAgent(t *testing.T) {
	if _, err := GlobalDirInfo("windsurf", t.TempDir()); err == nil {
		t.Fatal("windsurf is not in the table (D13)")
	}
}

// The upstream table wins over the adapter: claude's dir is the D11 snapshot
// entry, so the adapter can never move it.
func TestGlobalAgentDirTableFirst(t *testing.T) {
	home := t.TempDir()
	dir, err := globalAgentDir("claude", home)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".claude", "skills"); dir != want {
		t.Errorf("globalAgentDir(claude) = %q, want the table entry %q", dir, want)
	}
}

// hermes and mimocode are in DefaultAgents but not in the upstream snapshot, so
// their dir comes from the adapter that Magic owns.
func TestGlobalAgentDirAdapterFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HERMES_HOME", "")
	t.Setenv("MIMOCODE_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	cases := map[string]string{
		"hermes":   filepath.Join(home, ".hermes", "skills"),
		"mimocode": filepath.Join(home, ".config", "mimocode", "skills"),
	}
	for agent, want := range cases {
		dir, err := globalAgentDir(agent, home)
		if err != nil {
			t.Fatalf("%s: %v", agent, err)
		}
		if dir != want {
			t.Errorf("%s = %q, want %q", agent, dir, want)
		}
	}
}

// An id in neither the table nor the adapter registry stays an error.
func TestGlobalAgentDirUnknown(t *testing.T) {
	if _, err := globalAgentDir("windsurf", t.TempDir()); err == nil {
		t.Fatal("windsurf has no table entry and no adapter (D13)")
	}
}

// The directory --shared writes is the one the table points the whole
// .agents/skills group at, so one copy serves all eight ids.
func TestSharedTargetDirMatchesAgentsGroup(t *testing.T) {
	home := t.TempDir()
	shared := SharedTargetDir(home)
	if want := filepath.Join(home, ".agents", "skills"); shared != want {
		t.Fatalf("SharedTargetDir = %q, want %q", shared, want)
	}
	group := []string{"cline", "dexto", "kimi-code", "kimi-code-cli", "loaf", "sarvam-code", "warp", "zed"}
	for _, id := range group {
		dir, err := GlobalTargetDir(id, home)
		if err != nil {
			t.Fatal(err)
		}
		if dir != shared {
			t.Errorf("%s resolves to %q, shared layer is %q", id, dir, shared)
		}
	}
}
