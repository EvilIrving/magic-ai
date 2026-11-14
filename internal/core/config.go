package core

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LibrarySchemaVersion is the structure version of magic.yaml this magic reads
// and writes. A library without schema_version (0) was written before the field
// existed and needs `magic migrate` (architecture I7). Bump it together with the
// migration step in migrate.go, never on its own.
const LibrarySchemaVersion = 1

// MagicConfig is magic.yaml.
type MagicConfig struct {
	Version       string `yaml:"version"`
	SchemaVersion int    `yaml:"schema_version"`
	Adapters      struct {
		Enabled []string `yaml:"enabled"`
	} `yaml:"adapters"`
}

// SchemaState is how magic.yaml's schema_version compares to
// LibrarySchemaVersion. The three states decide whether a command may write.
type SchemaState int

const (
	// SchemaCurrent: the library is at the version this magic writes.
	SchemaCurrent SchemaState = iota
	// SchemaLegacy: schema_version is absent (0) or older than this magic.
	// Writing commands refuse until `magic migrate` upgrades the library, so an
	// old library is never silently rewritten by a newer layout.
	SchemaLegacy
	// SchemaNewer: a newer magic wrote the library. This one does not know the
	// fields that magic added, so rewriting the file would drop them: every
	// writing command refuses and points at the upgrade.
	SchemaNewer
)

// Schema reports the structure state of the library. A nil config is legacy: it
// carries no version to trust.
func (c *MagicConfig) Schema() SchemaState {
	if c == nil {
		return SchemaLegacy
	}
	switch {
	case c.SchemaVersion < LibrarySchemaVersion:
		return SchemaLegacy
	case c.SchemaVersion > LibrarySchemaVersion:
		return SchemaNewer
	}
	return SchemaCurrent
}

// SchemaProblem returns nil when this magic can read and write the library, and
// otherwise the one sentence naming the command that changes that.
func (c *MagicConfig) SchemaProblem() error {
	version := 0
	if c != nil {
		version = c.SchemaVersion
	}
	switch c.Schema() {
	case SchemaLegacy:
		return fmt.Errorf("库结构版本 %d，先运行 magic migrate", version)
	case SchemaNewer:
		return fmt.Errorf("库结构版本 %d 比 magic 支持的 %d 新，拒绝写入（请升级 magic）", version, LibrarySchemaVersion)
	}
	return nil
}

// CheckLibrarySchema loads magic.yaml and reports its structure state together
// with the refusal that state implies. A missing magic.yaml is current: the
// library has no versioned file yet and magic init creates one. A parse error is
// returned as is.
func CheckLibrarySchema(path string) (SchemaState, error) {
	cfg, err := LoadMagicConfig(path)
	if err != nil {
		return SchemaCurrent, err
	}
	return cfg.Schema(), cfg.SchemaProblem()
}

func LoadMagicConfig(path string) (*MagicConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			c := &MagicConfig{Version: "1.0", SchemaVersion: LibrarySchemaVersion}
			c.Adapters.Enabled = []string{"pi", "grok", "claude", "codex", "mcpuser", "opencode", "gemini", "copilot", "workbuddy", "trae", "cursor", "antigravity-cli", "kimi-code", "qoder", "qoder-cn", "cline", "kiro", "droid", "zcode", "qwen-code", "iflow-cli", "kilo", "zed"}
			return c, nil
		}
		return nil, err
	}
	var c MagicConfig
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *MagicConfig) IsEnabled(id string) bool {
	if c == nil || len(c.Adapters.Enabled) == 0 {
		return true
	}
	for _, e := range c.Adapters.Enabled {
		if e == id {
			return true
		}
	}
	return false
}
