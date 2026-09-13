package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"magic/internal/catalog"
	"magic/internal/core"
	"magic/internal/provider"
	"magic/internal/secrets"
)

func cmdProvider() *cobra.Command {
	c := &cobra.Command{
		Use:   "provider",
		Short: "Switch harnesses between official and third-party providers (library → dialect)",
		Long: `Product loop: official ↔ third-party (relay).

Library (source of truth):
  providers.yaml (providers + endpoints + pick + active)
  secrets/providers.json (local plaintext; .enc is cloud-only)
  magic.yaml adapter enablement

Semantics:
  magic provider use official|--harness …
  magic provider use <providerName>   # from providers.yaml
  magic provider use company          # alias for providers.active / MAGIC_ACTIVE_PROVIDER

Dialect writers (SwitchWriter): Claude, Codex, Grok, Pi, OpenCode, WorkBuddy implemented; gemini/copilot/trae apply-only.`,
	}
	c.PersistentFlags().String("harness", "", "claude|codex|grok|pi|… (default: all implemented for list/status; required for use/test/backup when ambiguous)")
	c.PersistentFlags().String("home", "", "user home (fixture)")

	c.AddCommand(cmdProviderList())
	c.AddCommand(cmdProviderStatus())
	c.AddCommand(cmdProviderUse())
	c.AddCommand(cmdProviderEnv())
	c.AddCommand(cmdProviderTest())
	c.AddCommand(cmdProviderBackup())
	c.AddCommand(cmdProviderSeed())
	return c
}

func providerCtx(cmd *cobra.Command) (provider.Context, error) {
	mh, err := resolveMagicHome()
	if err != nil {
		return provider.Context{}, err
	}
	homeFlag := ""
	for p := cmd; p != nil; p = p.Parent() {
		if f := p.PersistentFlags().Lookup("home"); f != nil {
			homeFlag = f.Value.String()
			if homeFlag != "" {
				break
			}
		}
	}
	uh, err := core.UserHome(homeFlag)
	if err != nil {
		return provider.Context{}, err
	}
	return provider.Context{MagicHome: mh, UserHome: uh}, nil
}

func harnessFlag(cmd *cobra.Command) string {
	for p := cmd; p != nil; p = p.Parent() {
		if f := p.PersistentFlags().Lookup("harness"); f != nil {
			v := f.Value.String()
			if v != "" {
				return provider.NormalizeHarness(v)
			}
		}
		if f := p.Flags().Lookup("harness"); f != nil {
			v := f.Value.String()
			if v != "" {
				return provider.NormalizeHarness(v)
			}
		}
	}
	return ""
}

func cmdProviderList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List library providers and harness switch targets",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := providerCtx(cmd)
			if err != nil {
				return err
			}
			_ = provider.EnsureSeeded(ctx.MagicHome)
			f, err := ctx.File()
			if err != nil {
				return err
			}
			fmt.Println("## Library (providers.yaml)")
			fmt.Print(catalog.FormatList(f))
			fmt.Printf("\nactive (company alias): %s\n", catalog.ActiveProviderName(f))
			h := harnessFlag(cmd)
			fmt.Println("\n## Harness writers")
			for _, w := range provider.Writers() {
				if h != "" && w.Harness() != h {
					continue
				}
				impl := "stub"
				if w.Implemented() {
					impl = "ok"
				}
				fmt.Printf("[%s] %s\n", impl, w.Harness())
				if w.Implemented() {
					fmt.Print(w.List(ctx))
				}
			}
			return nil
		},
	}
}

func cmdProviderStatus() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show official vs third-party state per harness",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := providerCtx(cmd)
			if err != nil {
				return err
			}
			h := harnessFlag(cmd)
			for _, w := range provider.Writers() {
				if h != "" && w.Harness() != h {
					continue
				}
				if !w.Implemented() && h == "" {
					continue
				}
				fmt.Print(w.Status(ctx))
				fmt.Println()
			}
			return nil
		},
	}
}

func cmdProviderUse() *cobra.Command {
	return &cobra.Command{
		Use:   "use <official|company|providerName>",
		Short: "Switch harness to official or a providers.yaml provider (company=active)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := providerCtx(cmd)
			if err != nil {
				return err
			}
			_ = provider.EnsureSeeded(ctx.MagicHome)
			h := harnessFlag(cmd)
			target := args[0]
			if h == "" {
				// apply to all implemented writers
				var failed bool
				for _, w := range provider.Writers() {
					if !w.Implemented() {
						continue
					}
					msg, err := provider.Use(ctx, w.Harness(), target)
					if err != nil {
						fmt.Fprintf(os.Stderr, "[%s] %v\n", w.Harness(), err)
						failed = true
						continue
					}
					fmt.Printf("## %s\n%s\n", w.Harness(), msg)
				}
				if failed {
					return fmt.Errorf("one or more harness switches failed")
				}
				return nil
			}
			msg, err := provider.Use(ctx, h, target)
			if err != nil {
				return err
			}
			fmt.Print(msg)
			return nil
		},
	}
}

func cmdProviderEnv() *cobra.Command {
	c := &cobra.Command{
		Use:   "env",
		Short: "Manage provider secrets (wraps magic secrets)",
	}
	var valueFlag string
	set := &cobra.Command{
		Use:   "set <key_env>",
		Short: "Set key in local plaintext secrets/providers.json",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			val := valueFlag
			if val == "" {
				return fmt.Errorf("pass --value (or use: magic secrets set %s --value …)", args[0])
			}
			if err := provider.SetSecret(core.NewPaths(mh).Secrets, args[0], val); err != nil {
				return err
			}
			fmt.Printf("set %s in secrets/%s\n", args[0], secrets.ProvidersFileName)
			return nil
		},
	}
	set.Flags().StringVar(&valueFlag, "value", "", "secret value")
	c.AddCommand(set)
	return c
}

func cmdProviderTest() *cobra.Command {
	return &cobra.Command{
		Use:   "test [provider|company|openai|deepseek|grok]",
		Short: "HTTP-test a provider gateway (harness-agnostic; uses secrets)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := providerCtx(cmd)
			if err != nil {
				return err
			}
			_ = provider.EnsureSeeded(ctx.MagicHome)
			target := "company"
			if len(args) > 0 {
				target = args[0]
			}
			// Soft aliases: openai/grok without a library name → company (active).
			switch strings.ToLower(target) {
			case "openai", "grok":
				target = "company"
			}
			h := harnessFlag(cmd)
			msg, err := provider.TestProvider(ctx, target)
			if err != nil {
				return err
			}
			fmt.Println(msg)
			// Optional Pi responses ping when harness=pi
			if h == provider.HarnessPi {
				f, _ := ctx.File()
				rt, err := provider.ResolveUseTarget(f, target)
				if err == nil && rt.Mode == provider.ModeProvider {
					ep, err := provider.PreferredEndpoint(rt.Provider, provider.HarnessPi)
					if err == nil {
						key, err := provider.ResolveSecret(ctx.SecretsDir(), rt.Provider.KeyEnv)
						if err == nil {
							model := ""
							if len(rt.Provider.Pick) > 0 {
								model = rt.Provider.Pick[0]
							}
							ping, err := provider.CheapResponsesPing(ctx.HTTP, ep.BaseURL, key, model)
							if err != nil {
								fmt.Fprintf(os.Stderr, "responses ping: %v\n", err)
							} else {
								fmt.Println(ping)
							}
						}
					}
				}
			}
			return nil
		},
	}
}

func cmdProviderBackup() *cobra.Command {
	c := &cobra.Command{
		Use:   "backup",
		Short: "List / restore / clear harness config backups under MAGIC_HOME/backups/<harness>",
	}
	c.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List backups",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := providerCtx(cmd)
			if err != nil {
				return err
			}
			h := harnessFlag(cmd)
			if h == "" {
				return fmt.Errorf("--harness required for backup list")
			}
			files, err := provider.ListBackups(ctx.MagicHome, h)
			if err != nil {
				return err
			}
			if len(files) == 0 {
				fmt.Println("No backups.")
				return nil
			}
			for _, f := range files {
				fmt.Println(f)
			}
			return nil
		},
	})
	c.AddCommand(&cobra.Command{
		Use:   "restore",
		Short: "Restore latest backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := providerCtx(cmd)
			if err != nil {
				return err
			}
			h := harnessFlag(cmd)
			if h == "" {
				return fmt.Errorf("--harness required")
			}
			w := provider.Writer(h)
			if w == nil || !w.Implemented() {
				return provider.ErrNotImplemented(h)
			}
			dest := w.ConfigPath(ctx.UserHome)
			from, err := provider.RestoreLatest(ctx.MagicHome, h, dest, w.BackupPrefix(), w.BackupExt())
			if err != nil {
				return err
			}
			fmt.Printf("Restored: %s → %s\n", from, dest)
			return nil
		},
	})
	c.AddCommand(&cobra.Command{
		Use:   "clear",
		Short: "Clear backups",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := providerCtx(cmd)
			if err != nil {
				return err
			}
			h := harnessFlag(cmd)
			if h == "" {
				return fmt.Errorf("--harness required")
			}
			n, err := provider.ClearBackups(ctx.MagicHome, h)
			if err != nil {
				return err
			}
			fmt.Printf("Removed %d backup(s).\n", n)
			return nil
		},
	})
	return c
}

func cmdProviderSeed() *cobra.Command {
	return &cobra.Command{
		Use:   "seed",
		Short: "Seed default provider profiles into providers.yaml (idempotent)",
		RunE: func(cmd *cobra.Command, args []string) error {
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			added, err := provider.SeedDefaults(mh)
			if err != nil {
				return err
			}
			if len(added) == 0 {
				fmt.Println("providers.yaml already has default profiles (or active set)")
				return nil
			}
			fmt.Printf("seeded: %s\n", strings.Join(added, ", "))
			return nil
		},
	}
}
