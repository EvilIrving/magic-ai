package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"magic/internal/adapter"
	"magic/internal/core"
	"magic/internal/registry"
)

// enabledAdapters returns the adapters apply would consider: the ones enabled in
// magic.yaml, narrowed by --only when given.
func enabledAdapters(magicHome, only string) []adapter.Adapter {
	cfg, _ := core.LoadMagicConfig(core.NewPaths(magicHome).MagicYAML)
	var out []adapter.Adapter
	for _, a := range registry.List() {
		if only != "" && string(a.ID()) != only {
			continue
		}
		if only == "" && cfg != nil && !cfg.IsEnabled(string(a.ID())) {
			continue
		}
		out = append(out, a)
	}
	return out
}

func cmdPlan() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:   "plan",
		Short: "只读：apply 会写哪些文件、改成什么",
		// A schema refusal is a normal outcome, not a usage error.
		SilenceUsage: true,
		Long: `只读地计算 apply 的结果：每个目标文件是新建、更新还是不变，
JSON 配置文件列出被覆盖与被保留的键，config.toml 列出行的增删。

按 apply 的写入顺序叠加四层：镜像文件、claude partial、provider 投影、MCP 投影；
后置层在前一层的输出上继续合并，所以 after 就是 apply 之后磁盘上的内容。
provider 层就是该 harness 的 SwitchWriter.Files 算出的字节，与 apply 同一份实现。

不写任何目标文件。入库、apply、sync 都不受影响。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			if err := requireLibrarySchema(mh); err != nil {
				return err
			}
			uh, err := core.UserHome(flagHome)
			if err != nil {
				return err
			}
			opts := core.ApplyOptions{MagicHome: mh, UserHome: uh, Only: flagOnly}
			p, err := core.BuildPlan(enabledAdapters(mh, flagOnly), opts)
			if err != nil {
				return err
			}
			if asJSON {
				out, err := p.JSON()
				if err != nil {
					return err
				}
				fmt.Print(string(out))
				return nil
			}
			printPlan(p)
			return nil
		},
	}
	c.Flags().StringVar(&flagHome, "home", "", "user home（fixture）")
	c.Flags().StringVar(&flagOnly, "only", "", "只处理一个 adapter")
	c.Flags().BoolVar(&asJSON, "json", false, "print as JSON")
	return c
}

func printPlan(p core.Plan) {
	fmt.Printf("配置库: %s\n", p.MagicHome)
	fmt.Printf("目标主目录: %s\n\n", p.UserHome)
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ADAPTER\tACTION\tREL\tMODE\tBACKUP\t说明")
	for _, ap := range p.Adapters {
		if ap.Skipped != "" {
			fmt.Fprintf(w, "%s\t-\t-\t-\t-\t%s\n", ap.Adapter, ap.Skipped)
			continue
		}
		for _, c := range ap.Changes {
			backup := "-"
			if c.Backup {
				backup = "yes"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", c.Adapter, c.Action, c.Rel, c.Mode, backup, c.Reason)
			for _, k := range c.Keys {
				fmt.Fprintf(w, "\t\t\t\t\t%s %s\n", k.Op, k.Key)
			}
			if c.Lines != nil {
				fmt.Fprintf(w, "\t\t\t\t\t行 +%d -%d\n", c.Lines.Added, c.Lines.Removed)
			}
		}
	}
	w.Flush()
	fmt.Printf("\n%d 个文件：%d 新建，%d 更新，%d 删除，%d 不变\n",
		p.Summary.Files, p.Summary.Create, p.Summary.Update, p.Summary.Delete, p.Summary.Unchanged)
}
