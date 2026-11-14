package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"magic/internal/core"
	"magic/internal/registry"
)

func cmdRollback() *cobra.Command {
	var list, dryRun, force bool
	var to, onlyAdapter string
	c := &cobra.Command{
		Use:   "rollback",
		Short: "把目标文件恢复到某次 apply 之前",
		Long: `从 backups/<时间戳>/ 回滚一次 apply。

有 manifest.json 的备份是完整回滚：apply 覆盖过的文件恢复原内容，apply 新建的文件被删除。
没有 manifest.json 的旧备份只能恢复文件，无法删除新建的文件。

数据安全：apply 之后被用户改过的目标默认不动，逐条报告跳过（内容指纹不等于 apply 写入的内容）。
带 --force 跳过内容校验，按 manifest 执行；目标在 user home 之外、备份路径越界的条目无论是否 --force 都不执行。
旧 manifest 没有内容指纹（无法判断），同样需要 --force。

默认恢复到最新一次 apply 之前的备份；--to 指定时间戳；--adapter 只处理一家。
带 --dry-run 只显示会动哪些文件。备份里没有对应 adapter 的目录会被跳过并列出。`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			uh, err := core.UserHome(flagHome)
			if err != nil {
				return err
			}
			if list {
				sets, err := core.ListBackups(mh)
				if err != nil {
					return err
				}
				providers, err := core.ListProviderBackupDirs(mh)
				if err != nil {
					return err
				}
				if len(sets) == 0 && len(providers) == 0 {
					fmt.Printf("没有备份：%s\n", mh+"/backups")
					return nil
				}
				if len(sets) == 0 {
					fmt.Printf("没有 apply 备份（只有 provider 备份）：%s\n", mh+"/backups")
				} else {
					w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
					fmt.Fprintln(w, "时间戳\t文件数\tADAPTERS")
					for _, s := range sets {
						fmt.Fprintf(w, "%s\t%d\t%s\n", s.Stamp, s.Files, joinComma(s.Adapters))
					}
					w.Flush()
				}
				if len(providers) > 0 {
					fmt.Printf("provider 备份，不参与回滚：%s\n", joinComma(providers))
				}
				return nil
			}

			res, err := core.Rollback(registry.List(), core.RollbackOptions{
				MagicHome: mh,
				UserHome:  uh,
				Stamp:     to,
				Adapter:   onlyAdapter,
				DryRun:    dryRun,
				Force:     force,
			})
			if err != nil {
				return err
			}
			restoreVerb, deleteVerb := "恢复", "删除"
			if dryRun {
				restoreVerb, deleteVerb = "将恢复", "将删除"
			}
			deletes := 0
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ADAPTER\t动作\tREL\t路径\t字节")
			for _, r := range res.Restored {
				action, size := restoreVerb, fmt.Sprintf("%d", r.Bytes)
				if r.Delete {
					action, size, deletes = deleteVerb, "-", deletes+1
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", r.Adapter, action, r.Rel, r.Path, size)
			}
			w.Flush()
			for _, s := range res.Refused {
				fmt.Printf("拒绝：%s\n", s)
			}
			for _, s := range res.Skipped {
				fmt.Printf("跳过：%s\n", s)
			}
			if len(res.Skipped) > 0 {
				fmt.Printf("\n跳过 %d 个（见上），加 --force 强制按 manifest 执行\n", len(res.Skipped))
			}
			if len(res.Refused) > 0 {
				fmt.Printf("\n拒绝 %d 个（见上），--force 也不执行\n", len(res.Refused))
			}
			fmt.Printf("\n%s %d 个文件，%s %d 个（备份 %s）\n", restoreVerb, len(res.Restored)-deletes, deleteVerb, deletes, res.Stamp)
			return nil
		},
	}
	c.Flags().BoolVar(&list, "list", false, "列出可用的备份")
	c.Flags().StringVar(&to, "to", "", "备份时间戳（默认最新一次）")
	c.Flags().StringVar(&onlyAdapter, "adapter", "", "只恢复一个 adapter")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "只显示，不写文件")
	c.Flags().BoolVar(&force, "force", false, "跳过内容校验，按 manifest 强制执行（路径越界仍然拒绝）")
	c.Flags().StringVar(&flagHome, "home", "", "user home（fixture）")
	return c
}

func joinComma(items []string) string {
	out := ""
	for i, s := range items {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}
