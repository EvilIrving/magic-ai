package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"magic/internal/core"
	"magic/internal/provider"
)

func cmdUseAlias() *cobra.Command {
	var harness, home string
	c := &cobra.Command{
		Use:   "use <official|company|providerName>",
		Short: "Alias for magic provider use",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			uh, err := core.UserHome(home)
			if err != nil {
				return err
			}
			_ = provider.EnsureSeeded(mh)
			ctx := provider.Context{MagicHome: mh, UserHome: uh}
			target := args[0]
			h := provider.NormalizeHarness(harness)
			if h == "" {
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
	c.Flags().StringVar(&harness, "harness", "", "claude|codex|grok|pi")
	c.Flags().StringVar(&home, "home", "", "user home (fixture)")
	return c
}

func cmdTestAlias() *cobra.Command {
	var home string
	c := &cobra.Command{
		Use:   "test [provider|company]",
		Short: "Alias for magic provider test",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			uh, err := core.UserHome(home)
			if err != nil {
				return err
			}
			_ = provider.EnsureSeeded(mh)
			ctx := provider.Context{MagicHome: mh, UserHome: uh}
			target := "company"
			if len(args) > 0 {
				target = args[0]
			}
			msg, err := provider.TestProvider(ctx, target)
			if err != nil {
				return err
			}
			fmt.Println(msg)
			return nil
		},
	}
	c.Flags().StringVar(&home, "home", "", "user home")
	return c
}
