package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"magic/internal/core"
	"magic/internal/mcp"
)

func cmdMCP() *cobra.Command {
	c := &cobra.Command{
		Use:   "mcp",
		Short: "MCP：库 packs/mcp/mcp.yaml → 各 harness 原生 MCP 配置",
		Long: `Canonical MCP definition lives in $MAGIC_HOME/packs/mcp/mcp.yaml.
` + "`magic apply`" + ` translates it into each harness's native file/format
(Claude/Codex/Grok/Pi/OpenCode/Gemini/Copilot/WorkBuddy/Trae/mcpuser).

Keys are written as plaintext values (D8); unsupported servers are skipped
with an "n/a" log line (D10).`,
	}
	c.AddCommand(cmdMCPList())
	c.AddCommand(cmdMCPStatus())
	c.AddCommand(cmdMCPInit())
	return c
}

func cmdMCPList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List canonical MCP servers in packs/mcp/mcp.yaml",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			path := filepath.Join(core.NewPaths(mh).PacksMCP, "mcp.yaml")
			f, err := mcp.Load(path)
			if err != nil {
				return err
			}
			if len(f.MCP) == 0 {
				fmt.Printf("(no MCP servers — %s)\n", path)
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tTRANSPORT\tENABLED\tTARGET")
			for _, n := range f.Names() {
				s := f.MCP[n]
				target := s.Command
				if s.IsRemote() {
					target = s.URL
				}
				fmt.Fprintf(w, "%s\t%s\t%v\t%s\n", n, s.TransportKind(), s.IsEnabled(), target)
			}
			w.Flush()
			return nil
		},
	}
}

func cmdMCPStatus() *cobra.Command {
	var home string
	c := &cobra.Command{
		Use:   "status",
		Short: "Show which canonical MCP servers are present per harness",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			uh, err := core.UserHome(home)
			if err != nil {
				return err
			}
			path := filepath.Join(core.NewPaths(mh).PacksMCP, "mcp.yaml")
			f, err := mcp.Load(path)
			if err != nil {
				return err
			}
			names := f.Names()
			if len(names) == 0 {
				fmt.Printf("(no MCP servers — %s)\n", path)
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "HARNESS\tFILE\tEXISTS\tPRESENT")
			for _, t := range mcp.Targets() {
				present, exists, err := mcp.Inspect(t, uh, names)
				if err != nil {
					return err
				}
				p := 0
				for _, ok := range present {
					if ok {
						p++
					}
				}
				file := t.Path(uh)
				if !exists {
					file += " (missing)"
				}
				fmt.Fprintf(w, "%s\t%s\t%v\t%d/%d\n", t.Harness, file, exists, p, len(names))
			}
			w.Flush()
			return nil
		},
	}
	c.Flags().StringVar(&home, "home", "", "user home（fixture）")
	return c
}

func cmdMCPInit() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Write an example packs/mcp/mcp.yaml (does not overwrite)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			dir := core.NewPaths(mh).PacksMCP
			path := filepath.Join(dir, "mcp.yaml")
			if _, err := os.Stat(path); err == nil {
				fmt.Printf("exists: %s\n", path)
				return nil
			}
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte(exampleMCPYAML), 0o600); err != nil {
				return err
			}
			fmt.Printf("wrote %s\n", path)
			return nil
		},
	}
}

const exampleMCPYAML = `# packs/mcp/mcp.yaml — canonical MCP servers, projected by "magic apply".
#
# transport: stdio | http | sse
#   stdio  → command / args / env / cwd
#   http|sse → url / headers
# enabled: default true
#
# Keys/headers are written to harness configs as plaintext values (decision D8).

mcp: {}
#  context7:
#    transport: stdio
#    command: npx
#    args: ["-y", "@upstash/context7-mcp@latest"]
#    env:
#      API_KEY: "sk-..."
#    enabled: true
#
#  linear:
#    transport: http
#    url: https://mcp.linear.app/mcp
#    headers:
#      Authorization: "Bearer ..."
`
