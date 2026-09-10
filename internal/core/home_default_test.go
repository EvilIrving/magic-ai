package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveHomeDefaultMagicAI(t *testing.T) {
	os.Unsetenv("MAGIC_HOME")
	os.Unsetenv("MAGIC_VAULT")
	got, err := ResolveHome()
	if err != nil {
		t.Fatal(err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".magic-ai")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
