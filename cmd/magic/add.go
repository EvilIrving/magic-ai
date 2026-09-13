package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"magic/internal/core"
	"magic/internal/registry"
)

func cmdAdd() *cobra.Command {
	c := &cobra.Command{
		Use:   "add",
		Short: "把外部文件收进配置库",
	}
	c.AddCommand(cmdAddPath())
	return c
}

func cmdAddPath() *cobra.Command {
	var as string
	c := &cobra.Command{
		Use:   "path <file>",
		Short: "复制 harness 自定义文件到 packs/harness/<id>/（不移动原文件）",
		Long: `Copy a harness-local file into the config library.

  magic add path ~/.pi/agent/extensions/catalog-allowlist.ts
    → $MAGIC_HOME/packs/harness/pi/extensions/catalog-allowlist.ts

  magic add path ./footer.ts --as pi/extensions/short-path-footer.ts

apply 再按文件写回该 harness。这是用户自己加定制的入口，不是内置清单。`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			uh, err := core.UserHome(flagHome)
			if err != nil {
				return err
			}
			res, err := core.AddPath(core.AddPathOptions{
				MagicHome: mh,
				UserHome:  uh,
				Source:    args[0],
				As:        as,
				Adapters:  registry.List(),
			})
			if err != nil {
				return err
			}
			fmt.Printf("added %s → packs/harness/%s/%s\n", args[0], res.Harness, res.Rel)
			return nil
		},
	}
	c.Flags().StringVar(&as, "as", "", "harness/relative path, e.g. pi/extensions/foo.ts")
	c.Flags().StringVar(&flagHome, "home", "", "user home (fixture)")
	return c
}
