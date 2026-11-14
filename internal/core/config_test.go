package core

import (
	"path/filepath"
	"strings"
	"testing"
)

// The three schema states are what every writing command branches on: current
// writes, a schema-0 library needs migrate, a newer library must not be written.
func TestMagicConfigSchemaStates(t *testing.T) {
	cases := []struct {
		name    string
		cfg     *MagicConfig
		want    SchemaState
		problem string
	}{
		{
			name: "current",
			cfg:  &MagicConfig{Version: "1.0", SchemaVersion: LibrarySchemaVersion},
			want: SchemaCurrent,
		},
		{
			name:    "legacy without schema_version",
			cfg:     &MagicConfig{Version: "1.0"},
			want:    SchemaLegacy,
			problem: "库结构版本 0，先运行 magic migrate",
		},
		{
			name:    "newer than this magic",
			cfg:     &MagicConfig{SchemaVersion: LibrarySchemaVersion + 1},
			want:    SchemaNewer,
			problem: "比 magic 支持的 1 新，拒绝写入",
		},
		{
			name:    "nil config has no version to trust",
			cfg:     nil,
			want:    SchemaLegacy,
			problem: "库结构版本 0，先运行 magic migrate",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.cfg.Schema(); got != tc.want {
				t.Fatalf("Schema() = %v, want %v", got, tc.want)
			}
			err := tc.cfg.SchemaProblem()
			if tc.problem == "" {
				if err != nil {
					t.Fatalf("SchemaProblem() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.problem) {
				t.Fatalf("SchemaProblem() = %v, want %q", err, tc.problem)
			}
		})
	}
}

// magic init writes the current schema version, so a fresh library never needs a
// migration.
func TestInitSkeletonWritesSchemaVersion(t *testing.T) {
	root := t.TempDir()
	if err := InitSkeleton(root); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadMagicConfig(NewPaths(root).MagicYAML)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SchemaVersion != LibrarySchemaVersion || cfg.Schema() != SchemaCurrent {
		t.Fatalf("init wrote schema_version %d (state %v), want current", cfg.SchemaVersion, cfg.Schema())
	}
}

// A missing magic.yaml is current: there is no versioned file to upgrade, and
// apply on a fresh MAGIC_HOME keeps working with the built-in default.
func TestLoadMagicConfigMissingFileIsCurrentSchema(t *testing.T) {
	cfg, err := LoadMagicConfig(filepath.Join(t.TempDir(), "magic.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Schema() != SchemaCurrent {
		t.Fatalf("missing magic.yaml schema = %v, want current", cfg.Schema())
	}
}
