package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"magic/internal/core"
)

func cmdMigrate() *cobra.Command {
	var dryRun bool
	c := &cobra.Command{
		Use:   "migrate",
		Short: "把配置库结构升级到当前 schema 版本",
		Long: `把 magic.yaml 的结构版本升级到 magic 支持的版本（当前 schema ` + fmt.Sprint(core.LibrarySchemaVersion) + `）。

只改 schema_version，不重写文件：用户自己的字段、字段顺序与注释都保留。
写盘前先把 magic.yaml 复制进 backups/<stamp>/，可以用 magic rollback --stamp <stamp> 回放。
--dry-run 只列出会改什么，不备份、不写盘。
已经是最新版本时输出「已是最新（schema N）」，退出码 0，不写盘。
库由更新版 magic 写过（schema_version 大于本程序支持的版本）时拒绝迁移，避免旧程序改坏新库。`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			res, err := core.MigrateLibrary(core.MigrateOptions{MagicHome: mh, DryRun: dryRun})
			if err != nil {
				return err
			}
			if res.AlreadyCurrent {
				fmt.Printf("已是最新（schema %d）\n", res.TargetVersion)
				return nil
			}
			if res.BackupDir != "" {
				fmt.Printf("backup → %s\n", res.BackupDir)
			}
			if res.DryRun {
				fmt.Printf("库结构版本 %d → %d（dry-run，未写盘）\n", res.SchemaVersion, res.TargetVersion)
			} else {
				fmt.Printf("已迁移：库结构版本 %d → %d\n", res.SchemaVersion, res.TargetVersion)
			}
			for _, m := range res.Changes {
				fmt.Printf("  %s: %s → %s\n", m.Field, m.From, m.To)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&dryRun, "dry-run", false, "只列出要改的字段，不写盘")
	return c
}
