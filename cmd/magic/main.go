package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"magic/internal/agents"
	"magic/internal/catalog"
	"magic/internal/core"
	"magic/internal/policy"
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
  MAGIC_PASSPHRASE          加解密 secrets/（capture/apply reinject + provider keys for copy）
  MAGIC_R2_*                Cloudflare R2 同步凭证

"agents" 命令 = 配置库 packs/agents 里的 sub-agent（不是独立 harness）。
Skill 真相源 = $MAGIC_HOME/packs/skills（默认 ~/.magic-ai）。
用 magic skills 管理同步（非 ~/.agents/skills）。`,
	}

	root.PersistentFlags().StringVar(&flagMagicHome, "magic-home", "", "配置库路径（默认 MAGIC_HOME / MAGIC_VAULT / ~/.magic-ai）")

	root.AddCommand(cmdVersion())
	root.AddCommand(cmdInit())
	root.AddCommand(cmdAdapters())
	root.AddCommand(cmdStatus())
	root.AddCommand(cmdCapture())
	root.AddCommand(cmdApply())
	root.AddCommand(cmdSync())
	root.AddCommand(cmdCatalog())
	root.AddCommand(cmdCopy())
	root.AddCommand(cmdSecrets())
	root.AddCommand(cmdRegistry())
	root.AddCommand(cmdAgents())
	root.AddCommand(cmdSkills())
	root.AddCommand(cmdProvider())
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

func cmdAdapters() *cobra.Command {
	return &cobra.Command{
		Use:   "adapters",
		Short: "列出 harness 适配器",
		Run: func(cmd *cobra.Command, args []string) {
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tIMPLEMENTED\tSKILLS\tAGENTS\tMCP")
			for _, a := range registry.List() {
				c := a.Caps()
				fmt.Fprintf(w, "%s\t%s\t%v\t%v\t%v\t%v\n",
					a.ID(), a.Name(), a.Implemented(), c.Skills, c.Agents, c.MCP)
			}
			w.Flush()
		},
	}
}

func cmdStatus() *cobra.Command {
	c := &cobra.Command{
		Use:   "status",
		Short: "显示配置库与适配器探测状态",
		RunE: func(cmd *cobra.Command, args []string) error {
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			uh, err := core.UserHome(flagHome)
			if err != nil {
				return err
			}
			fmt.Printf("Magic %s\n", core.Version)
			fmt.Printf("配置库 (MAGIC_HOME): %s\n", mh)
			if st, err := os.Stat(mh); err != nil || !st.IsDir() {
				fmt.Println("  (未初始化 — 运行: magic init)")
			} else {
				fmt.Println("  initialized: yes")
			}
			fmt.Printf("user home: %s\n\n", uh)
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ADAPTER\tDETECTED\tIMPLEMENTED\tMIRROR")
			for _, a := range registry.List() {
				mirror := filepath.Join(mh, a.LibraryMirrorRel())
				hasMirror := "no"
				if st, err := os.Stat(mirror); err == nil && st.IsDir() {
					hasMirror = "yes"
				}
				fmt.Fprintf(w, "%s\t%v\t%v\t%s\n", a.ID(), a.Detect(uh), a.Implemented(), hasMirror)
			}
			w.Flush()
			return nil
		},
	}
	c.Flags().StringVar(&flagHome, "home", "", "user home to scan (fixture)")
	return c
}

func cmdCapture() *cobra.Command {
	c := &cobra.Command{
		Use:   "capture",
		Short: "扫描 harness 根 → 分类 → 将 allow 写入配置库",
		RunE:  runCapture,
	}
	c.Flags().BoolVar(&flagDryRun, "dry-run", false, "仅分类，不复制")
	c.Flags().StringVar(&flagOnly, "only", "", "仅此 adapter id")
	c.Flags().StringVar(&flagHome, "home", "", "user home（fixture）")

	approve := &cobra.Command{
		Use:   "approve <adapter> <rel>",
		Short: "将路径加入 capture.overlay.yaml include",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return editOverlay(args[0], args[1], true)
		},
	}
	deny := &cobra.Command{
		Use:   "deny <adapter> <rel>",
		Short: "将路径加入 capture.overlay.yaml exclude",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return editOverlay(args[0], args[1], false)
		},
	}
	c.AddCommand(approve, deny)
	return c
}

func editOverlay(adapterID, rel string, approve bool) error {
	mh, err := resolveMagicHome()
	if err != nil {
		return err
	}
	path := core.NewPaths(mh).Overlay
	ov, err := policy.LoadOverlay(path)
	if err != nil {
		return err
	}
	if approve {
		ov.Approve(adapterID, rel)
		fmt.Printf("approved %s/%s → user.include\n", adapterID, rel)
	} else {
		ov.DenyPath(adapterID, rel)
		fmt.Printf("denied %s/%s → user.exclude\n", adapterID, rel)
	}
	return policy.SaveOverlay(path, ov)
}

func runCapture(cmd *cobra.Command, args []string) error {
	mh, err := resolveMagicHome()
	if err != nil {
		return err
	}
	uh, err := core.UserHome(flagHome)
	if err != nil {
		return err
	}
	if !flagDryRun {
		if err := core.InitSkeleton(mh); err != nil {
			return err
		}
	}
	ov, err := policy.LoadOverlay(core.NewPaths(mh).Overlay)
	if err != nil {
		return err
	}
	cfg, _ := core.LoadMagicConfig(core.NewPaths(mh).MagicYAML)
	opts := core.CaptureOptions{MagicHome: mh, UserHome: uh, DryRun: flagDryRun, Only: flagOnly}
	for _, a := range registry.List() {
		if !a.Implemented() {
			continue
		}
		if flagOnly != "" && string(a.ID()) != flagOnly {
			continue
		}
		if cfg != nil && !cfg.IsEnabled(string(a.ID())) && flagOnly == "" {
			continue
		}
		report, copied, err := core.CaptureAdapter(a, ov, opts)
		if err != nil {
			return err
		}
		fmt.Printf("\n## adapter %s\n", a.ID())
		fmt.Print(core.FormatReport(report))
		if !flagDryRun {
			fmt.Printf("copied %d files → %s\n", len(copied), filepath.Join(mh, a.LibraryMirrorRel()))
			for _, p := range copied {
				fmt.Printf("  %s\n", p)
			}
		}
	}
	return nil
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
			list := registry.List()
			for _, a := range list {
				if flagOnly != "" && string(a.ID()) != flagOnly {
					continue
				}
				if !a.Implemented() {
					fmt.Printf("[%s] skip (not implemented)\n", a.ID())
					continue
				}
				logs, err := core.ApplyAdapter(a, opts, backupDir)
				if err != nil {
					return err
				}
				for _, l := range logs {
					fmt.Println(l)
				}
			}
			skillLogs, err := core.ApplySkills(list, opts)
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
			agentLogs, err := core.ApplyAgents(list, opts)
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
			provLogs, err := core.ApplyProviders(list, opts)
			if err != nil {
				return err
			}
			fmt.Println("\n## providers projection (Claude / Pi)")
			if len(provLogs) == 0 {
				fmt.Println("(no provider projection)")
			}
			for _, l := range provLogs {
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
		Short: "上传配置库到 R2",
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
			if err := magsync.Push(context.Background(), cfg, mh); err != nil {
				return err
			}
			fmt.Println("push ok")
			return nil
		},
	})
	c.AddCommand(&cobra.Command{
		Use:   "pull",
		Short: "从 R2 下载配置库",
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
			if err := magsync.Pull(context.Background(), cfg, mh); err != nil {
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
		Short: "列出配置库 sub-agent 与各适配器 Caps.Agents 支持情况",
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
			fmt.Println("\nCaps.Agents adapters → AgentsDir:")
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
