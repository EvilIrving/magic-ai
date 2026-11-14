package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// runMagic executes one command against temp homes and returns its stdout.
// --home and --only go through cobra because the flags bind to the package
// globals when the command is built, which would overwrite anything set before
// the command exists.
func runMagic(t *testing.T, magicHome string, c *cobra.Command, args ...string) (string, error) {
	t.Helper()
	oldMagicHome, oldHome, oldOnly, oldDryRun := flagMagicHome, flagHome, flagOnly, flagDryRun
	t.Cleanup(func() { flagMagicHome, flagHome, flagOnly, flagDryRun = oldMagicHome, oldHome, oldOnly, oldDryRun })
	flagMagicHome = magicHome

	c.SilenceErrors = true
	c.SilenceUsage = true
	c.SetArgs(args)
	return captureStdout(t, func() error { return c.Execute() })
}

// captureStdout runs fn with os.Stdout redirected and returns what it printed.
func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	runErr := fn()
	w.Close()
	os.Stdout = old
	return <-done, runErr
}

func runVerify(t *testing.T, magicHome, userHome, only string) error {
	t.Helper()
	_, err := runVerifyOutput(t, magicHome, userHome, verifyArgs(userHome, only, false)...)
	return err
}

func runVerifyOutput(t *testing.T, magicHome, userHome string, args ...string) (string, error) {
	t.Helper()
	return runMagic(t, magicHome, cmdVerify(), args...)
}

func verifyArgs(userHome, only string, asJSON bool) []string {
	args := []string{"--home", userHome}
	if only != "" {
		args = append(args, "--only", only)
	}
	if asJSON {
		args = append(args, "--json")
	}
	return args
}

// runApply runs `magic apply` for one adapter against temp homes.
func runApply(t *testing.T, magicHome, userHome, only string) string {
	t.Helper()
	args := []string{"--home", userHome}
	if only != "" {
		args = append(args, "--only", only)
	}
	out, err := runMagic(t, magicHome, cmdApply(), args...)
	if err != nil {
		t.Fatalf("apply %s: %v\n%s", only, err, out)
	}
	return out
}

func writeFileAt(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// verify must not pass when it checked nothing: an empty library proves nothing.
func TestVerifyFailsWhenNothingIsChecked(t *testing.T) {
	err := runVerify(t, t.TempDir(), t.TempDir(), "")
	if err == nil {
		t.Fatal("verify passed without checking any file")
	}
	if !strings.Contains(err.Error(), "没有检查任何文件") {
		t.Fatalf("error = %v, want the empty-check failure", err)
	}
}

// A clean adapter passes only when every selected adapter was actually checked.
func TestVerifyPassesForAnInSyncAdapter(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFileAt(t, filepath.Join(magicHome, "packs", "harness", "pi", "AGENTS.md"), "# Pi\n")
	writeFileAt(t, filepath.Join(userHome, ".pi", "agent", "AGENTS.md"), "# Pi\n")

	if err := runVerify(t, magicHome, userHome, "pi"); err != nil {
		t.Fatalf("verify = %v, want no drift", err)
	}
}

// A skipped adapter is not a verified adapter, even when nothing drifted.
func TestVerifyFailsOnSkippedAdapters(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFileAt(t, filepath.Join(magicHome, "packs", "harness", "pi", "AGENTS.md"), "# Pi\n")
	writeFileAt(t, filepath.Join(userHome, ".pi", "agent", "AGENTS.md"), "# Pi\n")

	err := runVerify(t, magicHome, userHome, "")
	if err == nil {
		t.Fatal("verify passed while adapters were skipped")
	}
	if !strings.Contains(err.Error(), "被跳过") {
		t.Fatalf("error = %v, want the skipped-adapter failure", err)
	}
}

func TestVerifyFailsOnDrift(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFileAt(t, filepath.Join(magicHome, "packs", "harness", "pi", "AGENTS.md"), "# Pi\n")
	writeFileAt(t, filepath.Join(userHome, ".pi", "agent", "AGENTS.md"), "# dirty\n")

	err := runVerify(t, magicHome, userHome, "pi")
	if err == nil || !strings.Contains(err.Error(), "漂移") {
		t.Fatalf("error = %v, want a drift report", err)
	}
}

// captureStderr runs fn with os.Stderr redirected and returns what it printed.
func captureStderr(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	runErr := fn()
	w.Close()
	os.Stderr = old
	return <-done, runErr
}
