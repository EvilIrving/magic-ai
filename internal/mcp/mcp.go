// Package mcp projects the canonical MCP definition in
// $MAGIC_HOME/packs/mcp/mcp.yaml into each harness's native MCP config.
// Design: docs/mcp-design.md; decisions: docs/decisions.md D18.
package mcp

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Transport values in the canonical file.
const (
	TransportStdio = "stdio"
	TransportHTTP  = "http"
	TransportSSE   = "sse"
)

// Server is one canonical MCP server.
type Server struct {
	Transport string            `yaml:"transport,omitempty"` // stdio | http | sse (default stdio)
	Command   string            `yaml:"command,omitempty"`
	Args      []string          `yaml:"args,omitempty"`
	Env       map[string]string `yaml:"env,omitempty"`
	Cwd       string            `yaml:"cwd,omitempty"`
	URL       string            `yaml:"url,omitempty"`
	Headers   map[string]string `yaml:"headers,omitempty"`
	Enabled   *bool             `yaml:"enabled,omitempty"`
}

// File is the packs/mcp/mcp.yaml root.
type File struct {
	MCP map[string]Server `yaml:"mcp"`
}

// Load reads the canonical MCP file. Missing file → empty file, nil error.
func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &File{MCP: map[string]Server{}}, nil
		}
		return nil, err
	}
	var f File
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if f.MCP == nil {
		f.MCP = map[string]Server{}
	}
	return &f, nil
}

// Save writes the canonical file.
func Save(path string, f *File) error {
	if f == nil {
		f = &File{MCP: map[string]Server{}}
	}
	data, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// IsEnabled reports whether the server is enabled. nil → true.
func (s Server) IsEnabled() bool { return s.Enabled == nil || *s.Enabled }

// Transport returns the normalized transport (default stdio when a command is set).
func (s Server) TransportKind() string {
	t := strings.ToLower(strings.TrimSpace(s.Transport))
	switch t {
	case TransportHTTP, "streamable-http", "streamablehttp":
		return TransportHTTP
	case TransportSSE:
		return TransportSSE
	case TransportStdio:
		return TransportStdio
	}
	if strings.TrimSpace(s.URL) != "" {
		return TransportHTTP
	}
	return TransportStdio
}

// IsRemote reports whether the server is a remote (url-based) server.
func (s Server) IsRemote() bool {
	k := s.TransportKind()
	return (k == TransportHTTP || k == TransportSSE) && strings.TrimSpace(s.URL) != ""
}

// Names returns all server names, sorted.
func (f *File) Names() []string {
	if f == nil {
		return nil
	}
	names := make([]string, 0, len(f.MCP))
	for n := range f.MCP {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// EnabledNames returns enabled server names, sorted.
func (f *File) EnabledNames() []string {
	var out []string
	for _, n := range f.Names() {
		if f.MCP[n].IsEnabled() {
			out = append(out, n)
		}
	}
	return out
}
