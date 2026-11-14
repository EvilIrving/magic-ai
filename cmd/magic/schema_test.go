package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// legacyMagicYAMLFixture writes a magic.yaml from before schema_version existed:
// the library apply, plan and verify must refuse to touch until migration.
func legacyMagicYAMLFixture(t *testing.T, magicHome string, ids ...string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("# Magic config library (配置库)\nversion: \"1.0\"\nadapters:\n  enabled:\n")
	for _, id := range ids {
		b.WriteString("    - " + id + "\n")
	}
	writeFileAt(t, filepath.Join(magicHome, "magic.yaml"), b.String())
}

// schemaGatedCommands returns the three commands that refuse a library this
// magic cannot read, as the flag sets main builds.
func schemaGatedCommands() []struct {
	name string
	cmd  *cobra.Command
} {
	return []struct {
		name string
		cmd  *cobra.Command
	}{
		{"apply", cmdApply()},
		{"plan", cmdPlan()},
		{"verify", cmdVerify()},
	}
}

// The three commands that compute or write a projection refuse a schema-0
// library with the command that upgrades it, and apply writes nothing first.
func TestCommandsRefuseLegacySchema(t *testing.T) {
	commands := schemaGatedCommands()
	for _, tc := range commands {
		t.Run(tc.name, func(t *testing.T) {
			magicHome, userHome := t.TempDir(), t.TempDir()
			legacyMagicYAMLFixture(t, magicHome, "pi")
			writeFileAt(t, filepath.Join(magicHome, "packs", "harness", "pi", "AGENTS.md"), "# Pi\n")

			out, err := runMagic(t, magicHome, tc.cmd, "--home", userHome)
			if err == nil {
				t.Fatalf("%s ran on a schema-0 library:\n%s", tc.name, out)
			}
			if !strings.Contains(err.Error(), "库结构版本 0，先运行 magic migrate") {
				t.Fatalf("%s error = %v, want the migrate instruction", tc.name, err)
			}
			if _, statErr := os.Stat(filepath.Join(magicHome, "backups")); !os.IsNotExist(statErr) {
				t.Fatalf("%s created backups/ before refusing (err = %v)", tc.name, statErr)
			}
			if _, statErr := os.Stat(filepath.Join(userHome, ".pi", "agent", "AGENTS.md")); !os.IsNotExist(statErr) {
				t.Fatalf("%s wrote a target before refusing (err = %v)", tc.name, statErr)
			}
		})
	}
}

// A library written by a newer magic is refused the same way: the older program
// would drop fields it does not know.
func TestCommandsRefuseNewerSchema(t *testing.T) {
	commands := schemaGatedCommands()
	for _, tc := range commands {
		t.Run(tc.name, func(t *testing.T) {
			magicHome, userHome := t.TempDir(), t.TempDir()
			writeFileAt(t, filepath.Join(magicHome, "magic.yaml"), "version: \"1.0\"\nschema_version: 99\n")
			writeFileAt(t, filepath.Join(magicHome, "packs", "harness", "pi", "AGENTS.md"), "# Pi\n")

			out, err := runMagic(t, magicHome, tc.cmd, "--home", userHome)
			if err == nil {
				t.Fatalf("%s ran on a schema-99 library:\n%s", tc.name, out)
			}
			for _, want := range []string{"库结构版本 99", "拒绝写入"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("%s error = %v, want %q", tc.name, err, want)
				}
			}
		})
	}
}

// doctor reports a schema-0 library as a WARN with the migrate action, and still
// exits non-zero: apply, plan and verify would refuse it.
func TestDoctorWarnsAndFailsOnLegacySchema(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	legacyMagicYAMLFixture(t, magicHome, "claude")
	doctorFixtureProviders(t, magicHome, anthropicEndpoint)
	if err := os.MkdirAll(filepath.Join(userHome, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := runDoctorCmd(t, magicHome, userHome, "claude", false)
	if err == nil {
		t.Fatalf("doctor passed a schema-0 library:\n%s", out)
	}
	if !strings.Contains(err.Error(), "库结构版本 0，先运行 magic migrate") {
		t.Fatalf("doctor error = %v, want the migrate instruction", err)
	}
	if !strings.Contains(out, "库结构版本 0，先运行 magic migrate") {
		t.Fatalf("doctor did not print the schema WARN:\n%s", out)
	}
	if !strings.Contains(out, "magic migrate") {
		t.Fatalf("doctor did not name the migrate action:\n%s", out)
	}
}

// A newer library is a FAIL in doctor, with the upgrade as the next step.
func TestDoctorFailsOnNewerSchema(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	writeFileAt(t, filepath.Join(magicHome, "magic.yaml"), "version: \"1.0\"\nschema_version: 99\n")
	doctorFixtureProviders(t, magicHome, anthropicEndpoint)
	if err := os.MkdirAll(filepath.Join(userHome, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := runDoctorCmd(t, magicHome, userHome, "claude", false)
	if err == nil {
		t.Fatalf("doctor passed a schema-99 library:\n%s", out)
	}
	for _, want := range []string{"FAIL", "库结构版本 99 比 magic 支持的 1 新"} {
		if !strings.Contains(out, want) {
			t.Fatalf("doctor output lacks %q:\n%s", want, out)
		}
	}
}

// A current library is an OK schema finding: the section is not noise.
func TestDoctorSchemaOKOnCurrentLibrary(t *testing.T) {
	magicHome, userHome := t.TempDir(), t.TempDir()
	doctorFixtureMagicYAML(t, magicHome, "zcode")
	doctorFixtureProviders(t, magicHome, anthropicEndpoint)
	if err := os.MkdirAll(filepath.Join(userHome, ".zcode"), 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := runDoctorCmd(t, magicHome, userHome, "zcode", false)
	if err != nil {
		t.Fatalf("doctor = %v:\n%s", err, out)
	}
	if !strings.Contains(out, "OK    库结构版本 1") {
		t.Fatalf("doctor did not report the current schema:\n%s", out)
	}
}

// runLikeCLI runs c the way main does: cobra prints the error to stderr and,
// unless the command silences usage, the whole flag list after it. runMagic
// silences both, so it cannot catch a schema refusal that buries the migrate
// instruction under the usage block.
func runLikeCLI(t *testing.T, magicHome string, c *cobra.Command, args ...string) (string, error) {
	t.Helper()
	oldMagicHome, oldHome, oldOnly, oldDryRun := flagMagicHome, flagHome, flagOnly, flagDryRun
	t.Cleanup(func() { flagMagicHome, flagHome, flagOnly, flagDryRun = oldMagicHome, oldHome, oldOnly, oldDryRun })
	flagMagicHome = magicHome
	c.SetArgs(args)
	return captureStderr(t, func() error { return c.Execute() })
}

// A schema refusal is one line. The usage block is for wrong flags, not for a
// library this magic cannot read.
func TestSchemaRefusalDoesNotPrintUsage(t *testing.T) {
	for _, tc := range schemaGatedCommands() {
		t.Run(tc.name, func(t *testing.T) {
			magicHome, userHome := t.TempDir(), t.TempDir()
			legacyMagicYAMLFixture(t, magicHome, "pi")

			stderr, err := runLikeCLI(t, magicHome, tc.cmd, "--home", userHome)
			if err == nil {
				t.Fatalf("%s ran on a schema-0 library:\n%s", tc.name, stderr)
			}
			if strings.Contains(stderr, "Usage:") {
				t.Fatalf("%s printed the usage block on a schema refusal:\n%s", tc.name, stderr)
			}
			if !strings.Contains(stderr, "库结构版本 0，先运行 magic migrate") {
				t.Fatalf("%s did not print the migrate instruction:\n%s", tc.name, stderr)
			}
		})
	}
}
