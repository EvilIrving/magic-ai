package agents

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyProjectsMarkdown(t *testing.T) {
	tmp := t.TempDir()
	lib := filepath.Join(tmp, "packs", "agents")
	_ = os.MkdirAll(lib, 0o755)
	_ = os.WriteFile(filepath.Join(lib, "reviewer.md"), []byte("# reviewer\n"), 0o644)
	h := filepath.Join(tmp, ".claude", "agents")
	logs, err := ApplyProjects(lib, []string{h}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Fatalf("logs=%v", logs)
	}
	data, err := os.ReadFile(filepath.Join(h, "reviewer.md"))
	if err != nil || string(data) != "# reviewer\n" {
		t.Fatalf("%s %v", data, err)
	}
}
