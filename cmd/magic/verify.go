package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"magic/internal/adapter"
	"magic/internal/catalog"
	"magic/internal/core"
	"magic/internal/mcp"
	"magic/internal/provider"
	"magic/internal/registry"
	"magic/internal/secrets"
)

// driftCause is why one target drifted, and the command that resolves it. A row
// can carry more than one: a provider-projected file that also has a library
// mirror is both a provider question and a manual-edit question.
type driftCause struct {
	// ID is the machine-readable cause: never-applied | local-edit |
	// provider-projection | content-differs.
	ID     string `json:"id"`
	Reason string `json:"reason"`
	Action string `json:"action"`
}

// driftRow is one file whose content differs from what the library would write.
type driftRow struct {
	Adapter string       `json:"adapter"`
	Rel     string       `json:"rel"`
	Path    string       `json:"path"`
	Kind    string       `json:"kind"` // missing | differ | delete
	Before  string       `json:"before,omitempty"`
	After   string       `json:"after"`
	Causes  []driftCause `json:"causes,omitempty"`
}

func cmdVerify() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:   "verify",
		Short: "只读：检查各 harness 的配置是否与配置库一致",
		// A drift report is a normal outcome, not a usage error.
		SilenceUsage: true,
		Long: `对比配置库与每个目标文件，报告缺失、内容不同与应当删除的文件。

每个漂移项带一个原因和一个下一步：
  missing  目标文件不存在 → magic apply --only <id>
  differ   目标有库内同名镜像文件 → 可能是手工改动，也可能是软件升级后重写了字段 → magic plan --only <id>
  differ   文件由 provider 投影写出 → magic provider use <name> --harness <id>

发现漂移时退出码为 1，可以直接放进 CI：apply 之后 verify 应当没有漂移。
被跳过的 adapter（未实现、库内没有镜像目录、没有镜像根）也会让退出码为 1：
它没有被检查，verify 不能声称它一致。没有检查任何文件也同样是失败，
并说明是哪一种原因：库里没有可投影的内容、provider 的协议对不上、enabled 排除了 adapter。
只读，不写任何目标文件。`,
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
			adapters := enabledAdapters(mh, flagOnly)
			p, err := core.BuildPlan(adapters, opts)
			if err != nil {
				return err
			}
			activeProvider := activeProviderName(mh)

			var rows []driftRow
			checked := 0
			var skipped []string
			for _, ap := range p.Adapters {
				if ap.Skipped != "" {
					skipped = append(skipped, fmt.Sprintf("%s: %s", ap.Adapter, ap.Skipped))
					continue
				}
				for _, c := range ap.Changes {
					checked++
					if c.Action == core.ActionUnchanged {
						continue
					}
					kind := "differ"
					switch c.Action {
					case core.ActionCreate:
						kind = "missing"
					case core.ActionDelete:
						kind = "delete"
					}
					rows = append(rows, driftRow{
						Adapter: string(c.Adapter),
						Rel:     c.Rel,
						Path:    c.Path,
						Kind:    kind,
						Before:  string(c.Before),
						After:   string(c.After),
						Causes:  driftCauses(c, mh, activeProvider),
					})
				}
			}

			if asJSON {
				out, err := json.MarshalIndent(map[string]any{
					"magic_home": p.MagicHome,
					"user_home":  p.UserHome,
					"checked":    checked,
					"drift":      rows,
					"skipped":    skipped,
				}, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(out))
			} else {
				fmt.Printf("配置库: %s\n", p.MagicHome)
				fmt.Printf("目标主目录: %s\n\n", p.UserHome)
				w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
				fmt.Fprintln(w, "ADAPTER\t状态\tREL\t路径")
				for _, r := range rows {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Adapter, r.Kind, r.Rel, r.Path)
				}
				w.Flush()
				for _, s := range skipped {
					fmt.Printf("跳过：%s\n", s)
				}
				printDriftCauses(rows)
				fmt.Printf("\n检查 %d 个文件，%d 处漂移，%d 个 adapter 被跳过\n", checked, len(rows), len(skipped))
			}
			if len(rows) > 0 {
				return fmt.Errorf("发现 %d 处漂移（运行 magic apply 修正，magic plan 查看细节）", len(rows))
			}
			if checked == 0 {
				return verifyNothingChecked(p, mh, uh, flagOnly, adapters)
			}
			if len(skipped) > 0 {
				return fmt.Errorf("%d 个 adapter 被跳过，verify 不能判定它们的状态", len(skipped))
			}
			return nil
		},
	}
	c.Flags().StringVar(&flagHome, "home", "", "user home（fixture）")
	c.Flags().StringVar(&flagOnly, "only", "", "只检查一个 adapter")
	c.Flags().BoolVar(&asJSON, "json", false, "print as JSON")
	return c
}

// driftCauses classifies one drift by what a user can do about it. It reads only
// the plan's change and the filesystem: no drift is left without a next step.
func driftCauses(c core.Change, magicHome, activeProvider string) []driftCause {
	id := string(c.Adapter)
	switch c.Action {
	case core.ActionCreate:
		return []driftCause{{
			ID:     "never-applied",
			Reason: "目标文件不存在：从未 apply，或被删除",
			Action: fmt.Sprintf("magic apply --only %s", id),
		}}
	case core.ActionDelete:
		// A projection removing a Magic-managed file; the plan names the layer.
		return nil
	}

	var out []driftCause
	if libraryMirrorExists(magicHome, id, c.Rel) {
		reason := "内容与配置库不一致：可能是手工改动"
		if jsonParseable(c.Path) {
			reason = "内容与配置库不一致：可能是手工改动，也可能是软件升级后重写了字段"
		}
		out = append(out, driftCause{
			ID:     "local-edit",
			Reason: reason,
			Action: fmt.Sprintf("magic plan --only %s", id),
		})
	}
	if strings.Contains(c.Reason, core.ProviderLayerLabel) {
		name := activeProvider
		if name == "" {
			name = "company"
		}
		out = append(out, driftCause{
			ID:     "provider-projection",
			Reason: fmt.Sprintf("该文件由 provider 投影写出：内容是 active provider %q 的 endpoint 投影结果", name),
			Action: fmt.Sprintf("magic provider use %s --harness %s", name, id),
		})
	}
	if len(out) == 0 {
		out = append(out, driftCause{
			ID:     "content-differs",
			Reason: "目标内容与配置库投影的结果不一致",
			Action: fmt.Sprintf("magic plan --only %s", id),
		})
	}
	return out
}

// libraryMirrorExists reports whether the library holds a mirror file the target
// is projected from. A provider-only or MCP-only target has none.
func libraryMirrorExists(magicHome, adapterID, rel string) bool {
	a, ok := registry.ByID(adapterID)
	if !ok || a.LibraryMirrorRel() == "" {
		return false
	}
	st, err := os.Stat(filepath.Join(magicHome, a.LibraryMirrorRel(), filepath.FromSlash(rel)))
	return err == nil && !st.IsDir()
}

// jsonParseable reports whether the target holds JSON (jsonc included), which is
// what "the software rewrote its own fields" applies to.
func jsonParseable(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var v any
	return json.Unmarshal(secrets.StripJSONC(data), &v) == nil
}

// activeProviderName is the provider providers.yaml names, for the hint that names
// the provider in a command.
func activeProviderName(magicHome string) string {
	f, err := provider.LoadCatalog(magicHome)
	if err != nil {
		return ""
	}
	return catalog.ActiveProviderName(f)
}

// printDriftCauses prints one block per drifted row: the reason, then the command
// that changes it. Rows without a cause (a projection removing a file) print none.
func printDriftCauses(rows []driftRow) {
	if len(rows) == 0 {
		return
	}
	fmt.Println()
	for _, r := range rows {
		if len(r.Causes) == 0 {
			continue
		}
		fmt.Printf("%s  %s  (%s)\n", r.Adapter, r.Rel, r.Kind)
		for _, c := range r.Causes {
			fmt.Printf("    原因: %s\n", c.Reason)
			fmt.Printf("    下一步: %s\n", c.Action)
		}
	}
}

// verifyNothingChecked explains why verify checked no file. It names the one
// cause a user can act on, in the order that decides what to change first: the
// adapter --only named that magic.yaml leaves disabled, a provider the software
// cannot take, a library with nothing to project for that adapter, or an enabled
// list that leaves the library's content out. "all adapters were skipped" was the
// old wording, and it blamed the adapters for problems that live in magic.yaml,
// providers.yaml or the library.
func verifyNothingChecked(p core.Plan, magicHome, userHome, only string, adapters []adapter.Adapter) error {
	opts := core.ApplyOptions{MagicHome: magicHome, UserHome: userHome, Only: only}
	// --only bypasses the enabled list, so a disabled adapter is skipped without
	// the adapter list showing it: name the list instead of the adapters.
	if _, ok := registry.ByID(only); only != "" && ok && !adapterEnabled(magicHome, only) {
		return fmt.Errorf("没有检查任何文件：%s 未启用（magic.yaml 的 adapters.enabled），下一步: magic status 看已安装的软件", only)
	}
	for _, a := range adapters {
		if gap, ok := core.ProviderProjectionGap(a, opts); ok && gap.ProtocolMismatch {
			return fmt.Errorf("没有检查任何文件：provider %s 没有 %s 支持的协议（下一步: magic doctor）", gap.Provider, a.Name())
		}
	}
	// --only asks about one adapter, so say what the library lacks for it rather
	// than describing the library as a whole.
	if a, ok := registry.ByID(only); only != "" && ok && a.Implemented() && planSkippedFor(p, only) {
		return fmt.Errorf("没有检查任何文件：配置库里没有可以投影给 %s 的内容（下一步: magic add path <file> 把 harness 文件收进库）", only)
	}
	if libraryHasNoProjectableContent(magicHome) {
		return fmt.Errorf("没有检查任何文件：配置库里没有可投影的内容（下一步: magic init 建库，magic add path <file> 把 harness 文件收进库）")
	}
	if disabled := disabledAdapterCount(magicHome, only); disabled > 0 {
		return fmt.Errorf("没有检查任何文件：%d 个 adapter 未启用（magic.yaml 的 adapters.enabled 没有它们，下一步: magic status 看已安装的软件）", disabled)
	}
	return fmt.Errorf("没有检查任何文件：所有 adapter 都被跳过，verify 不能据此判定一致")
}

// adapterEnabled reports whether magic.yaml's adapters.enabled names id. A
// missing magic.yaml uses LoadMagicConfig's built-in list, the same default
// apply reads.
func adapterEnabled(magicHome, id string) bool {
	cfg, err := core.LoadMagicConfig(core.NewPaths(magicHome).MagicYAML)
	return err == nil && cfg.IsEnabled(id)
}

// planSkippedFor reports whether the plan writes nothing for id: every layer
// (mirror, provider, MCP) was empty or did not apply, so the library has nothing
// apply could put in front of it.
func planSkippedFor(p core.Plan, id string) bool {
	for _, ap := range p.Adapters {
		if string(ap.Adapter) == id {
			return ap.Skipped != ""
		}
	}
	return false
}

// libraryHasNoProjectableContent reports that the library holds nothing apply
// could write for any harness: no file under packs/harness/<id>, no MCP server,
// no provider. It looks at every implemented adapter, not just the enabled ones,
// so an enabled list that leaves the content out is reported as that, not as an
// empty library.
func libraryHasNoProjectableContent(magicHome string) bool {
	for _, a := range registry.List() {
		if !a.Implemented() {
			continue
		}
		rel := a.LibraryMirrorRel()
		if rel == "" {
			continue
		}
		entries, err := os.ReadDir(filepath.Join(magicHome, filepath.FromSlash(rel)))
		if err == nil && len(entries) > 0 {
			return false
		}
	}
	if mf, err := mcp.Load(filepath.Join(magicHome, filepath.FromSlash(mcp.CanonicalRelPath))); err == nil && len(mf.MCP) > 0 {
		return false
	}
	if f, err := catalog.Load(core.NewPaths(magicHome).ProvidersYAML); err == nil && len(f.Providers) > 0 {
		return false
	}
	return true
}

// disabledAdapterCount counts the adapters magic.yaml's enabled list leaves out.
// --only bypasses that list, so it counts none.
func disabledAdapterCount(magicHome, only string) int {
	if only != "" {
		return 0
	}
	cfg, err := core.LoadMagicConfig(core.NewPaths(magicHome).MagicYAML)
	if err != nil {
		return 0
	}
	n := 0
	for _, a := range registry.List() {
		if !cfg.IsEnabled(string(a.ID())) {
			n++
		}
	}
	return n
}
