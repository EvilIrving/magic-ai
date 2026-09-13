package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"magic/internal/adapter"
	"magic/internal/agents"
	"magic/internal/catalog"
	"magic/internal/core"
	"magic/internal/mcp"
	"magic/internal/registry"
	magsync "magic/internal/sync"
)

var (
	flagHome      string
	flagMagicHome string
	flagDryRun    bool
	flagOnly      string
)

func main() {
	root := &cobra.Command{
		Use:   "magic",
		Short: "Magic — AI harness 配置管理器（配置库）",
		Long: `Magic 管理 AI coding harness 的用户级配置，真相源是本地配置库（默认 ~/.magic-ai）。

环境变量:
  MAGIC_HOME / MAGIC_VAULT  配置库路径（默认 ~/.magic-ai）
  MAGIC_PASSPHRASE          仅 sync push/pull：含密钥文件 ↔ 云端 .enc；日常命令不需要
  MAGIC_R2_*                Cloudflare R2 同步凭证

"agents" 命令 = 配置库 packs/agents 里的 sub-agent（不是独立 harness）。
Skill 真相源 = $MAGIC_HOME/packs/skills（默认 ~/.magic-ai）。
用 magic skills 管理同步（非 ~/.agents/skills）。`,
	}

	root.PersistentFlags().StringVar(&flagMagicHome, "magic-home", "", "配置库路径（默认 MAGIC_HOME / MAGIC_VAULT / ~/.magic-ai）")

	root.AddCommand(cmdVersion())
	root.AddCommand(cmdInit())
	root.AddCommand(cmdGuide())
	root.AddCommand(cmdAdapters())
	root.AddCommand(cmdStatus())
	root.AddCommand(cmdApply())
	root.AddCommand(cmdSync())
	root.AddCommand(cmdCatalog())
	root.AddCommand(cmdMCP())
	root.AddCommand(cmdCopy())
	root.AddCommand(cmdSecrets())
	root.AddCommand(cmdRegistry())
	root.AddCommand(cmdAgents())
	root.AddCommand(cmdSkills())
	root.AddCommand(cmdProvider())
	root.AddCommand(cmdLogin())
	root.AddCommand(cmdAdd())
	root.AddCommand(cmdUseAlias())
	root.AddCommand(cmdTestAlias())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func resolveMagicHome() (string, error) {
	if flagMagicHome != "" {
		return filepath.Clean(flagMagicHome), nil
	}
	return core.ResolveHome()
}

func cmdVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print Magic version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(core.Version)
		},
	}
}

func cmdInit() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "在 MAGIC_HOME 创建配置库骨架",
		RunE: func(cmd *cobra.Command, args []string) error {
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			if err := core.InitSkeleton(mh); err != nil {
				return err
			}
			fmt.Printf("initialized 配置库 at %s\n", mh)
			return nil
		},
	}
}

func cmdGuide() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:   "guide",
		Short: "Print the getting-started sequence",
		RunE: func(cmd *cobra.Command, args []string) error {
			steps := []struct {
				Cmd  string `json:"command"`
				What string `json:"what"`
			}{
				{"magic init", "Create the config library at MAGIC_HOME (default ~/.magic-ai)"},
				{"magic provider add --list", "List builtin providers you can import"},
				{"magic provider add deepseek", "Copy a builtin provider into providers.yaml (user wins)"},
				{"magic login deepseek", "Enter a key, fetch /models, pick models, write providers.yaml + secret"},
				{"magic secrets set DEEPSEEK_API_KEY --value '…'", "Store the API key locally in plaintext (0600); .enc is cloud-only"},
				{"magic apply", "Project skills / MCP / providers into each installed harness"},
				{"magic status", "Show which harnesses are installed here"},
				{"magic sync push", "Back up the config library (needs MAGIC_R2_* + MAGIC_PASSPHRASE)"},
			}
			if asJSON {
				out, err := json.MarshalIndent(map[string]any{"steps": steps}, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(out))
				return nil
			}
			fmt.Println("Magic 快速上手")
			fmt.Println()
			for i, s := range steps {
				fmt.Printf("%d. %-45s %s\n", i+1, s.Cmd, s.What)
			}
			fmt.Println()
			fmt.Println("换机器：magic sync pull + magic apply。配置库是唯一真相源，harness 家目录只是投影目标。")
			return nil
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "print the guide as JSON")
	return c
}

func cmdAdapters() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:   "adapters",
		Short: "列出 harness 适配器",
		RunE: func(cmd *cobra.Command, args []string) error {
			if asJSON {
				out := []map[string]any{}
				for _, a := range registry.List() {
					c := a.Caps()
					out = append(out, map[string]any{
						"id": a.ID(), "name": a.Name(), "implemented": a.Implemented(),
						"skills": c.Skills, "agents": c.Agents, "mcp": c.MCP, "hooks": c.Hooks, "models": c.Models, "themes": c.Themes,
					})
				}
				b, err := json.MarshalIndent(out, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(b))
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tIMPLEMENTED\tSKILLS\tSUB-AGENT\tMCP\tHOOKS")
			for _, a := range registry.List() {
				c := a.Caps()
				fmt.Fprintf(w, "%s\t%s\t%v\t%v\t%v\t%v\t%v\n",
					a.ID(), a.Name(), a.Implemented(), c.Skills, c.Agents, c.MCP, c.Hooks)
			}
			w.Flush()
			return nil
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "print as JSON")
	return c
}

func cmdStatus() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:   "status",
		Short: "显示配置库与已安装 harness 状态",
		RunE: func(cmd *cobra.Command, args []string) error {
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			uh, err := core.UserHome(flagHome)
			if err != nil {
				return err
			}
			initialized := false
			if st, err := os.Stat(mh); err == nil && st.IsDir() {
				initialized = true
			}
			type adapterRow struct {
				ID          string `json:"id"`
				Implemented bool   `json:"implemented"`
				Mirror      bool   `json:"mirror"`
			}
			var rows []adapterRow
			for _, a := range registry.List() {
				if !a.Detect(uh) {
					continue
				}
				mirror := false
				if st, err := os.Stat(filepath.Join(mh, a.LibraryMirrorRel())); err == nil && st.IsDir() {
					mirror = true
				}
				rows = append(rows, adapterRow{ID: string(a.ID()), Implemented: a.Implemented(), Mirror: mirror})
			}
			if asJSON {
				b, err := json.MarshalIndent(map[string]any{
					"version":     core.Version,
					"magic_home":  mh,
					"initialized": initialized,
					"user_home":   uh,
					"adapters":    rows,
				}, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(b))
				return nil
			}
			fmt.Printf("Magic %s\n", core.Version)
			fmt.Printf("配置库 (MAGIC_HOME): %s\n", mh)
			if !initialized {
				fmt.Println("  (未初始化 — 运行: magic init)")
			} else {
				fmt.Println("  initialized: yes")
			}
			fmt.Printf("user home: %s\n\n", uh)
			// D12: probe every adapter, but only show the ones installed on this machine.
			fmt.Println("已安装 harness（完整列表见 magic adapters）:")
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ADAPTER\tIMPLEMENTED\tMIRROR")
			for _, r := range rows {
				mirror := "no"
				if r.Mirror {
					mirror = "yes"
				}
				fmt.Fprintf(w, "%s\t%v\t%s\n", r.ID, r.Implemented, mirror)
			}
			if len(rows) == 0 {
				fmt.Println("  (none detected)")
			} else {
				w.Flush()
			}
			return nil
		},
	}
	c.Flags().StringVar(&flagHome, "home", "", "user home to scan (fixture)")
	c.Flags().BoolVar(&asJSON, "json", false, "print as JSON")
	return c
}

func cmdApply() *cobra.Command {
	c := &cobra.Command{
		Use:   "apply",
		Short: "从配置库恢复镜像 + 投影 global packs/skills 与 sub-agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			uh, err := core.UserHome(flagHome)
			if err != nil {
				return err
			}
			opts := core.ApplyOptions{MagicHome: mh, UserHome: uh, DryRun: flagDryRun, Only: flagOnly}
			var backupDir string
			if !flagDryRun {
				backupDir, err = core.NewBackupDir(mh)
				if err != nil {
					return err
				}
				fmt.Printf("backup → %s\n", backupDir)
			}
			cfg, _ := core.LoadMagicConfig(core.NewPaths(mh).MagicYAML)
			list := registry.List()
			var enabled []adapter.Adapter
			for _, a := range list {
				if flagOnly != "" && string(a.ID()) != flagOnly {
					continue
				}
				if flagOnly == "" && cfg != nil && !cfg.IsEnabled(string(a.ID())) {
					continue
				}
				if !a.Implemented() {
					fmt.Printf("[%s] skip (not implemented)\n", a.ID())
					continue
				}
				enabled = append(enabled, a)
				logs, err := core.ApplyAdapter(a, opts, backupDir)
				if err != nil {
					return err
				}
				for _, l := range logs {
					fmt.Println(l)
				}
			}
			skillLogs, err := core.ApplySkills(enabled, opts)
			if err != nil {
				return err
			}
			fmt.Println("\n## skills projection")
			if len(skillLogs) == 0 {
				fmt.Println("(no skills entries)")
			}
			for _, l := range skillLogs {
				fmt.Println(l)
			}
			agentLogs, err := core.ApplyAgents(enabled, opts)
			if err != nil {
				return err
			}
			fmt.Println("\n## sub-agent projection")
			if len(agentLogs) == 0 {
				fmt.Println("(no packs/agents entries)")
			}
			for _, l := range agentLogs {
				fmt.Println(l)
			}
			provLogs, err := core.ApplyProviders(enabled, opts)
			if err != nil {
				return err
			}
			fmt.Println("\n## providers projection")
			if len(provLogs) == 0 {
				fmt.Println("(no provider projection)")
			}
			for _, l := range provLogs {
				fmt.Println(l)
			}
			mcpLogs, err := mcp.Apply(mcp.Options{
				MagicHome: mh,
				UserHome:  uh,
				DryRun:    flagDryRun,
				Only:      flagOnly,
				BackupDir: backupDir,
				Adapters:  enabled,
			})
			if err != nil {
				return err
			}
			fmt.Println("\n## mcp projection")
			if len(mcpLogs) == 0 {
				fmt.Println("(no mcp projection)")
			}
			for _, l := range mcpLogs {
				fmt.Println(l)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&flagDryRun, "dry-run", false, "仅预览")
	c.Flags().StringVar(&flagOnly, "only", "", "仅此 adapter id")
	c.Flags().StringVar(&flagHome, "home", "", "user home（fixture）")
	return c
}

func cmdSync() *cobra.Command {
	c := &cobra.Command{
		Use:   "sync",
		Short: "经 Cloudflare R2 同步配置库",
	}
	c.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "显示 R2 同步状态",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, ok, msg := magsync.FromEnv()
			if !ok {
				fmt.Fprintln(os.Stderr, msg)
				os.Exit(2)
			}
			s, err := magsync.Status(context.Background(), cfg)
			if err != nil {
				return err
			}
			fmt.Println(s)
			return nil
		},
	})
	c.AddCommand(&cobra.Command{
		Use:   "push",
		Short: "上传配置库到 R2（含密钥文件加密成 .enc）",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, ok, msg := magsync.FromEnv()
			if !ok {
				fmt.Fprintln(os.Stderr, msg)
				os.Exit(2)
			}
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			if err := magsync.Push(context.Background(), cfg, mh, os.Getenv("MAGIC_PASSPHRASE")); err != nil {
				return err
			}
			fmt.Println("push ok")
			return nil
		},
	})
	c.AddCommand(&cobra.Command{
		Use:   "pull",
		Short: "从 R2 下载配置库（.enc 解密为本地明文）",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, ok, msg := magsync.FromEnv()
			if !ok {
				fmt.Fprintln(os.Stderr, msg)
				os.Exit(2)
			}
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			if err := magsync.Pull(context.Background(), cfg, mh, os.Getenv("MAGIC_PASSPHRASE")); err != nil {
				return err
			}
			fmt.Println("pull ok")
			return nil
		},
	})
	return c
}

func cmdCatalog() *cobra.Command {
	c := &cobra.Command{
		Use:   "catalog",
		Short: "列出 providers.yaml 中的 provider / pick",
		RunE: func(cmd *cobra.Command, args []string) error {
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			f, err := catalog.Load(core.NewPaths(mh).ProvidersYAML)
			if err != nil {
				return err
			}
			fmt.Print(catalog.FormatList(f))
			return nil
		},
	}
	c.AddCommand(cmdCatalogPick())
	return c
}

func cmdAgents() *cobra.Command {
	return &cobra.Command{
		Use:   "agents",
		Short: "列出配置库 sub-agent 与各适配器支持情况",
		RunE: func(cmd *cobra.Command, args []string) error {
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			lib := core.NewPaths(mh).PacksAgents
			names, err := agents.ListLibrary(lib)
			if err != nil {
				return err
			}
			fmt.Printf("packs/agents (%s):\n", lib)
			if len(names) == 0 {
				fmt.Println("  (empty — add <name>.md)")
			}
			for _, n := range names {
				fmt.Printf("  - %s\n", n)
			}
			fmt.Println("\nSub-agent adapters → dir:")
			uh, _ := core.UserHome("")
			for _, a := range registry.List() {
				if !a.Implemented() || !a.Caps().Agents {
					continue
				}
				fmt.Printf("  %s → %s\n", a.ID(), a.AgentsDir(uh))
			}
			return nil
		},
	}
}
