package core

import (
	"os"

	"gopkg.in/yaml.v3"
)

// MagicConfig is magic.yaml.
type MagicConfig struct {
	Version  string `yaml:"version"`
	Adapters struct {
		Enabled []string `yaml:"enabled"`
	} `yaml:"adapters"`
}

func LoadMagicConfig(path string) (*MagicConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			c := &MagicConfig{Version: "1.0"}
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
