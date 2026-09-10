package core

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExtractClaudePartial(t *testing.T) {
	in := []byte(`{
  "mcpServers": {"fs": {"command": "npx"}},
  "theme": "dark",
  "projects": {"/huge/path": {"history": [1,2,3]}},
  "numStartups": 999,
  "model": "sonnet"
}`)
	out, err := ExtractClaudePartial(in)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["mcpServers"]; !ok {
		t.Fatal("mcpServers missing")
	}
	if _, ok := m["theme"]; !ok {
		t.Fatal("theme missing")
	}
	if _, ok := m["projects"]; ok {
		t.Fatal("projects should not be extracted")
	}
	if strings.Contains(string(out), "history") {
		t.Fatal("runtime state leaked")
	}
}
