package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"magic/internal/adapter"
	"magic/internal/backup"
	"magic/internal/catalog"
	"magic/internal/core"
	"magic/internal/provider"
	"magic/internal/registry"
	"magic/internal/secrets"
)

// doctor statuses. FAIL is the only one that makes the exit code non-zero: a WARN
// is something to know about, not something that stops a working setup.
const (
	doctorOK   = "OK"
	doctorWarn = "WARN"
	doctorFail = "FAIL"
)

// doctorResult is one finding: what was checked, what it found, and the command
// that changes it.
type doctorResult struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Action  string `json:"action,omitempty"`
}

// doctorSection groups the findings of one check, so the terminal prints the
// seven checks in order while --json flattens them into one list.
type doctorSection struct {
	ID      string
	Title   string
	Results []doctorResult
}

// doctorReport is everything doctor found for one library and one user home.
// Results is the flat --json view; sections keeps the same findings grouped for
// the terminal and is not serialized.
type doctorReport struct {
	MagicHome string         `json:"magic_home"`
	UserHome  string         `json:"user_home"`
	Results   []doctorResult `json:"results"`
	OK        int            `json:"ok"`
	Warn      int            `json:"warn"`
	Fail      int            `json:"fail"`
	sections  []doctorSection
}

func cmdDoctor() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:   "doctor",
		Short: "只读：诊断「配置库 → 各软件」链路上会卡住的状态",
		Long: `按顺序检查八件事，每项给 OK / WARN / FAIL、一句人话和一个可执行的下一步：

  1 库结构版本  magic.yaml 的 schema_version 与本程序是否匹配（旧库 WARN + magic migrate；更新的库 FAIL）
  2 配置库        MAGIC_HOME 是否存在、magic.yaml 是否可解析、是否启用 adapter
  3 密钥          providers.yaml 的每个 key_env 是否在 secrets/providers.json 里有值
  4 协议匹配      active provider 会被选中的 endpoint，该软件是否吃得下这个协议
  5 未实现的软件  已安装但 provider 写入仍是 stub 的软件，附 stub 原因
  6 漂移          与 magic plan 同一份计算，统计非 unchanged 的目标文件
  7 目标可写性    这些目标文件的父目录是否可写
  8 备份可用性    backups/ 有没有 manifest，最近一次备份是什么时候

只读：不写任何文件，不改任何目标。有 FAIL 时退出码为 1；
库结构版本为 0（旧库）或比程序新时退出码也为 1，提示 magic migrate 或升级 magic。
--only <id> 只查一个软件；1 / 2 / 3 / 8 是库级检查，不受 --only 影响。`,
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
			if flagOnly != "" {
				if _, ok := registry.ByID(flagOnly); !ok {
					return fmt.Errorf("unknown adapter %q（magic adapters 列出全部）", flagOnly)
				}
			}
			rep := runDoctor(mh, uh, flagOnly)
			// The schema state decides the exit code even when it is only a WARN:
			// apply, plan and verify refuse a library this magic cannot read, and
			// doctor has to say so instead of looking green.
			_, schemaErr := core.CheckLibrarySchema(core.NewPaths(mh).MagicYAML)
			if asJSON {
				out, err := json.MarshalIndent(rep, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(out))
			} else {
				printDoctor(rep)
			}
			if rep.Fail > 0 {
				return fmt.Errorf("doctor: %d 项 FAIL（见上）", rep.Fail)
			}
			return schemaErr
		},
	}
	c.Flags().StringVar(&flagHome, "home", "", "user home（fixture）")
	c.Flags().StringVar(&flagOnly, "only", "", "只查一个软件（adapter id）")
	c.Flags().BoolVar(&asJSON, "json", false, "print as JSON")
	return c
}

// runDoctor runs the eight checks in product order. The plan is computed once
// and shared by the drift and writability checks, so both see the same target
// files.
func runDoctor(magicHome, userHome, only string) doctorReport {
	rep := doctorReport{MagicHome: magicHome, UserHome: userHome}
	plan, planErr := core.BuildPlan(enabledAdapters(magicHome, only), core.ApplyOptions{
		MagicHome: magicHome,
		UserHome:  userHome,
		Only:      only,
	})
	rep.sections = []doctorSection{
		{ID: "schema", Title: "1 库结构版本", Results: checkSchema(magicHome)},
		{ID: "library", Title: "2 配置库", Results: checkLibrary(magicHome)},
		{ID: "secrets", Title: "3 密钥", Results: checkSecrets(magicHome)},
		{ID: "protocol", Title: "4 协议匹配", Results: checkProtocol(magicHome, userHome, only)},
		{ID: "stubs", Title: "5 未实现的软件", Results: checkStubs(userHome, only)},
		{ID: "drift", Title: "6 漂移", Results: checkDrift(plan, planErr)},
		{ID: "writable", Title: "7 目标可写性", Results: checkWritable(plan, planErr)},
		{ID: "backups", Title: "8 备份可用性", Results: checkBackups(magicHome)},
	}
	for _, s := range rep.sections {
		for _, r := range s.Results {
			rep.Results = append(rep.Results, r)
			switch r.Status {
			case doctorOK:
				rep.OK++
			case doctorWarn:
				rep.Warn++
			case doctorFail:
				rep.Fail++
			}
		}
	}
	return rep
}

// checkSchema reads magic.yaml's schema_version. A library written before the
// field existed is a WARN with the migrate command: it still reads, and doctor
// is where the user finds out before apply refuses. A library written by a newer
// magic is a FAIL: this magic does not know the fields that magic added, so
// rewriting the file would drop them.
func checkSchema(magicHome string) []doctorResult {
	p := core.NewPaths(magicHome)
	if _, err := os.Stat(p.MagicYAML); os.IsNotExist(err) {
		return []doctorResult{{
			ID: "schema", Status: doctorWarn,
			Message: fmt.Sprintf("magic.yaml 不存在，库结构版本按内置默认 schema %d", core.LibrarySchemaVersion),
			Action:  "magic init",
		}}
	}
	cfg, err := core.LoadMagicConfig(p.MagicYAML)
	if err != nil {
		return []doctorResult{{
			ID: "schema", Status: doctorFail,
			Message: "magic.yaml 无法解析，判断不了库结构版本",
			Action:  "修正 magic.yaml 的 YAML（见「2 配置库」）",
		}}
	}
	switch cfg.Schema() {
	case core.SchemaLegacy:
		return []doctorResult{{
			ID: "schema", Status: doctorWarn,
			Message: fmt.Sprintf("库结构版本 %d，先运行 magic migrate", cfg.SchemaVersion),
			Action:  "magic migrate（只改 schema_version，其它字段与注释保留）",
		}}
	case core.SchemaNewer:
		return []doctorResult{{
			ID: "schema", Status: doctorFail,
			Message: fmt.Sprintf("库结构版本 %d 比 magic 支持的 %d 新，拒绝写入", cfg.SchemaVersion, core.LibrarySchemaVersion),
			Action:  "升级 magic 后再用这个配置库（旧版写它会丢掉新字段）",
		}}
	}
	return []doctorResult{{
		ID: "schema", Status: doctorOK,
		Message: fmt.Sprintf("库结构版本 %d（magic 支持的版本）", cfg.SchemaVersion),
	}}
}

// checkLibrary verifies the library itself: the directory, magic.yaml, and that
// at least one adapter is enabled.
func checkLibrary(magicHome string) []doctorResult {
	p := core.NewPaths(magicHome)
	st, err := os.Stat(magicHome)
	if err != nil {
		msg := "配置库不存在: " + magicHome
		if !os.IsNotExist(err) {
			msg = fmt.Sprintf("配置库无法读取: %s（%v）", magicHome, err)
		}
		return []doctorResult{{ID: "library.home", Status: doctorFail, Message: msg, Action: "magic init"}}
	}
	if !st.IsDir() {
		return []doctorResult{{
			ID: "library.home", Status: doctorFail,
			Message: "MAGIC_HOME 不是目录: " + magicHome,
			Action:  "换一个目录，或删掉同名文件后 magic init",
		}}
	}
	out := []doctorResult{{ID: "library.home", Status: doctorOK, Message: "配置库存在: " + magicHome}}

	// A missing magic.yaml is a WARN: LoadMagicConfig falls back to a built-in
	// default that enables every adapter. A parse error is a FAIL: apply would
	// silently use that same default instead of the user's file.
	cfg, cfgErr := core.LoadMagicConfig(p.MagicYAML)
	if _, statErr := os.Stat(p.MagicYAML); os.IsNotExist(statErr) {
		return append(out, doctorResult{
			ID: "library.config", Status: doctorWarn,
			Message: "magic.yaml 不存在，将使用内置默认（启用全部 adapter）",
			Action:  "magic init",
		})
	}
	if cfgErr != nil {
		return append(out, doctorResult{
			ID: "library.config", Status: doctorFail,
			Message: "magic.yaml 无法解析: " + cfgErr.Error(),
			Action:  "修正 YAML（magic init 只补缺失文件，不覆盖已有 magic.yaml）",
		})
	}
	if len(cfg.Adapters.Enabled) == 0 {
		return append(out, doctorResult{
			ID: "library.config", Status: doctorWarn,
			Message: "magic.yaml 没有启用任何 adapter，apply 会跳过全部软件",
			Action:  "在 magic.yaml 的 adapters.enabled 里加上要投影的软件 id（magic adapters 列出全部）",
		})
	}
	return append(out, doctorResult{
		ID: "library.config", Status: doctorOK,
		Message: fmt.Sprintf("magic.yaml 可解析，启用 %d 个 adapter", len(cfg.Adapters.Enabled)),
	})
}

// checkSecrets verifies every provider in the user's providers.yaml has a value
// for its key_env in the local plaintext store.
//
// It reads providers.yaml directly rather than provider.LoadCatalog: the merged
// catalog also holds the embedded builtin template, whose key_env names the user
// never configured and should not be reported as missing.
func checkSecrets(magicHome string) []doctorResult {
	p := core.NewPaths(magicHome)
	f, err := catalog.Load(p.ProvidersYAML)
	if err != nil {
		return []doctorResult{{
			ID: "secrets", Status: doctorFail,
			Message: "providers.yaml 无法解析: " + err.Error(),
			Action:  "修正 providers.yaml（magic init 生成的示例在 providers.yaml 里注释着格式）",
		}}
	}
	if len(f.Providers) == 0 {
		return []doctorResult{{
			ID: "secrets", Status: doctorFail,
			Message: "providers.yaml 里没有任何 provider，apply 没有可投影的 provider",
			Action:  "magic provider add --list 然后 magic provider add <name>",
		}}
	}
	store, err := secrets.LoadProviderSecrets(p.Secrets)
	if err != nil {
		return []doctorResult{{
			ID: "secrets", Status: doctorFail,
			Message: "secrets/providers.json 无法解析: " + err.Error(),
			Action:  "修正或删除 $MAGIC_HOME/secrets/providers.json 后 magic secrets set <KEY>",
		}}
	}
	names := make([]string, 0, len(f.Providers))
	for name := range f.Providers {
		names = append(names, name)
	}
	sort.Strings(names)
	var missing, actions []string
	for _, name := range names {
		keyEnv := strings.TrimSpace(f.Providers[name].KeyEnv)
		if keyEnv == "" {
			keyEnv = name
		}
		if strings.TrimSpace(store[keyEnv]) != "" {
			continue
		}
		missing = append(missing, fmt.Sprintf("%s（provider %s 的 key_env）", keyEnv, name))
		actions = append(actions, fmt.Sprintf("magic secrets set %s --value '…'", keyEnv))
	}
	if len(missing) > 0 {
		return []doctorResult{{
			ID: "secrets", Status: doctorFail,
			Message: "缺少密钥: " + strings.Join(missing, "；"),
			Action:  strings.Join(actions, "；"),
		}}
	}
	return []doctorResult{{
		ID: "secrets", Status: doctorOK,
		Message: fmt.Sprintf("%d 个 provider 的 key_env 在 secrets/providers.json 里都有值", len(names)),
	}}
}

// checkProtocol is the check that answers "will this software reject the provider
// with a 400": which endpoint of the active provider is selected for each
// installed software, and whether that software's dialect supports its protocol.
func checkProtocol(magicHome, userHome, only string) []doctorResult {
	f, err := provider.LoadCatalog(magicHome)
	if err != nil {
		return []doctorResult{{
			ID: "protocol", Status: doctorFail,
			Message: "providers.yaml 无法解析: " + err.Error(),
			Action:  "修正 providers.yaml",
		}}
	}
	active := catalog.ActiveProviderName(f)
	var out []doctorResult
	for _, a := range doctorAdapters(userHome, only) {
		id := string(a.ID())
		w := provider.Writer(id)
		if w == nil || !w.Implemented() {
			// No dialect writer, or a stub: the stub check reports those.
			continue
		}
		r := doctorResult{ID: "protocol." + id}
		if active == "" {
			r.Status = doctorFail
			r.Message = fmt.Sprintf("%s：没有 active provider（providers.yaml 的 active / MAGIC_ACTIVE_PROVIDER）", a.Name())
			r.Action = "在 providers.yaml 写 active: <provider>，或 export MAGIC_ACTIVE_PROVIDER=<provider>"
			out = append(out, r)
			continue
		}
		p, ok := f.Providers[active]
		if !ok {
			r.Status = doctorFail
			r.Message = fmt.Sprintf("%s：active provider %q 不在 providers.yaml 里", a.Name(), active)
			r.Action = "magic provider add --list 导入它，或修正 active"
			out = append(out, r)
			continue
		}
		ep, err := provider.PreferredEndpoint(p, id)
		if err != nil {
			r.Status = doctorFail
			r.Message = fmt.Sprintf("%s：provider %s 没有可用 endpoint（%v）", a.Name(), active, err)
			r.Action = fmt.Sprintf("在 providers.yaml 的 %s.endpoints 里加一条（base_url + protocol）", active)
			out = append(out, r)
			continue
		}
		want := provider.SupportedProtocols(id)
		if len(want) > 0 && !protocolIn(ep.Protocol, want) {
			r.Status = doctorFail
			r.Message = fmt.Sprintf("%s 不支持 %s，需要给这家 provider 加一个 %s 的 endpoint", a.Name(), ep.Protocol, want[0])
			r.Action = fmt.Sprintf("在 providers.yaml 的 %s.endpoints 里加一条 protocol: %s 的 endpoint", active, want[0])
			out = append(out, r)
			continue
		}
		r.Status = doctorOK
		r.Message = fmt.Sprintf("%s → endpoint %q（%s）", a.Name(), ep.ID, ep.Protocol)
		out = append(out, r)
	}
	if len(out) == 0 {
		return []doctorResult{{
			ID: "protocol", Status: doctorWarn,
			Message: "没有已安装且有 SwitchWriter 的软件可检查",
			Action:  "magic status 看这台机器装了哪些软件",
		}}
	}
	return out
}

// protocolIn reports whether protocol is one of the spellings in want.
func protocolIn(protocol string, want []string) bool {
	for _, w := range want {
		if catalog.ProtocolsMatch(protocol, w) {
			return true
		}
	}
	return false
}

// checkStubs reports installed software whose provider write is still a stub, and
// why. The reason text comes from internal/provider/stubdialects.go.
func checkStubs(userHome, only string) []doctorResult {
	var out []doctorResult
	for _, a := range doctorAdapters(userHome, only) {
		id := string(a.ID())
		w := provider.Writer(id)
		if w == nil || w.Implemented() {
			continue
		}
		reason, ok := provider.StubReason(id)
		if !ok {
			reason = "provider 写入尚未实现（原因见 internal/provider/stubdialects.go）"
		}
		out = append(out, doctorResult{
			ID: "stubs." + id, Status: doctorWarn,
			Message: fmt.Sprintf("%s 的 provider 写入未实现：%s", a.Name(), reason),
			Action:  fmt.Sprintf("magic adapters 看 %s 支持哪些投影；provider 仍要在软件里手动配置（apply 照常投影 skills / MCP）", id),
		})
	}
	if len(out) == 0 {
		return []doctorResult{{ID: "stubs", Status: doctorOK, Message: "已检查的软件都有 provider writer"}}
	}
	return out
}

// checkDrift reports the files apply would write, from the same core.BuildPlan
// magic plan and magic verify use.
func checkDrift(p core.Plan, planErr error) []doctorResult {
	if planErr != nil {
		return []doctorResult{{
			ID: "drift", Status: doctorFail,
			Message: "算不出 apply 的目标文件: " + planErr.Error(),
			Action:  "magic plan 看完整错误",
		}}
	}
	var dirty, skipped []string
	checked := 0
	for _, ap := range p.Adapters {
		if ap.Skipped != "" {
			skipped = append(skipped, string(ap.Adapter))
			continue
		}
		for _, c := range ap.Changes {
			checked++
			if c.Action == core.ActionUnchanged {
				continue
			}
			dirty = append(dirty, fmt.Sprintf("%s %s（%s）", c.Adapter, c.Rel, c.Action))
		}
	}
	if len(dirty) > 0 {
		msg := fmt.Sprintf("%d 个目标文件与配置库不一致: %s", len(dirty), joinFirst(dirty, 3))
		if len(dirty) > 3 {
			msg += fmt.Sprintf(" 等 %d 个", len(dirty))
		}
		return []doctorResult{{
			ID: "drift", Status: doctorWarn, Message: msg,
			Action: "magic plan（看逐键差异）/ magic apply（写回目标）",
		}}
	}
	if checked == 0 {
		return []doctorResult{{
			ID: "drift", Status: doctorWarn,
			Message: "没有可检查的文件：所有 adapter 都被跳过",
			Action:  "magic plan 看每个 adapter 的跳过原因",
		}}
	}
	msg := fmt.Sprintf("%d 个目标文件与配置库一致", checked)
	if len(skipped) > 0 {
		msg += fmt.Sprintf("（%d 个 adapter 未检查: %s）", len(skipped), joinFirst(skipped, 5))
	}
	return []doctorResult{{ID: "drift", Status: doctorOK, Message: msg}}
}

// checkWritable verifies the parent directory of every target file apply would
// write is writable. It stats the nearest existing ancestor and never creates a
// directory: doctor writes nothing.
func checkWritable(p core.Plan, planErr error) []doctorResult {
	if planErr != nil {
		return []doctorResult{{
			ID: "writable", Status: doctorFail,
			Message: "算不出 apply 的目标文件: " + planErr.Error(),
			Action:  "magic plan 看完整错误",
		}}
	}
	checked := 0
	badDirs := map[string]string{}
	var order []string
	for _, ap := range p.Adapters {
		for _, c := range ap.Changes {
			// unchanged writes nothing; create / update / delete all need a
			// writable parent directory.
			if c.Action == core.ActionUnchanged {
				continue
			}
			checked++
			dir, ok, why := doctorWritableDir(c.Path)
			if ok {
				continue
			}
			if _, seen := badDirs[dir]; !seen {
				order = append(order, dir)
			}
			badDirs[dir] = why
		}
	}
	if len(badDirs) == 0 {
		if checked == 0 {
			return []doctorResult{{ID: "writable", Status: doctorOK, Message: "没有要动的目标文件"}}
		}
		return []doctorResult{{
			ID: "writable", Status: doctorOK,
			Message: fmt.Sprintf("%d 个目标文件的父目录都可写", checked),
		}}
	}
	var details []string
	for _, dir := range order {
		details = append(details, fmt.Sprintf("%s（%s）", dir, badDirs[dir]))
	}
	return []doctorResult{{
		ID: "writable", Status: doctorFail,
		Message: "父目录不可写: " + strings.Join(details, "；"),
		Action:  "chmod u+w <目录>，或把 --home / MAGIC_HOME 指到可写位置",
	}}
}

// doctorWritableDir returns the nearest existing directory above path and whether
// the current user can write in it. Only the owner write bit is read, which is the
// bit apply needs; doctor does not try a real write.
func doctorWritableDir(path string) (dir string, ok bool, why string) {
	dir = filepath.Dir(path)
	for {
		st, err := os.Stat(dir)
		if err == nil {
			if !st.IsDir() {
				return dir, false, "不是目录"
			}
			if st.Mode().Perm()&0o200 == 0 {
				return dir, false, fmt.Sprintf("权限 %04o，没有写权限", st.Mode().Perm())
			}
			return dir, true, ""
		}
		if !os.IsNotExist(err) {
			return dir, false, err.Error()
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return dir, false, "找不到存在的父目录"
		}
		dir = parent
	}
}

// checkBackups reports whether rollback has anything to replay: the newest apply
// backup set, its manifest, and when it ran.
func checkBackups(magicHome string) []doctorResult {
	sets, err := core.ListBackups(magicHome)
	if err != nil {
		return []doctorResult{{
			ID: "backups", Status: doctorFail,
			Message: "读不了 backups/: " + err.Error(),
			Action:  "检查 $MAGIC_HOME/backups 的权限",
		}}
	}
	if len(sets) == 0 {
		return []doctorResult{{
			ID: "backups", Status: doctorWarn,
			Message: "还没有 apply 备份：rollback 要等 apply 之后才有东西可回",
			Action:  "magic apply",
		}}
	}
	newest := sets[len(sets)-1]
	dir := filepath.Join(magicHome, "backups", newest.Stamp)
	m, err := backup.Read(dir)
	if err != nil {
		return []doctorResult{{
			ID: "backups", Status: doctorFail,
			Message: fmt.Sprintf("备份 %s 的 manifest.json 无法解析: %v", newest.Stamp, err),
			Action:  "删掉这个备份目录，或重新 magic apply",
		}}
	}
	if m == nil {
		return []doctorResult{{
			ID: "backups", Status: doctorWarn,
			Message: fmt.Sprintf("最近一次备份 %s 没有 manifest.json（旧版 apply）：rollback 只能恢复文件，无法删除新建文件", newest.Stamp),
			Action:  "magic apply 生成带 manifest 的新备份",
		}}
	}
	when := newest.Stamp
	if t, err := time.Parse(core.BackupStampFormat, newest.Stamp); err == nil {
		when = t.UTC().Format(time.RFC3339)
	}
	return []doctorResult{{
		ID: "backups", Status: doctorOK,
		Message: fmt.Sprintf("最近一次备份: %s（%d 个文件，%d 个 adapter）", when, len(m.Entries), len(newest.Adapters)),
	}}
}

// doctorAdapters returns the adapters the per-software checks look at: the
// installed ones, or exactly --only when asked (an explicit request is answered
// even when the software is not detected here).
func doctorAdapters(userHome, only string) []adapter.Adapter {
	if only != "" {
		if a, ok := registry.ByID(only); ok {
			return []adapter.Adapter{a}
		}
		return nil
	}
	var out []adapter.Adapter
	for _, a := range registry.List() {
		if a.Detect(userHome) {
			out = append(out, a)
		}
	}
	return out
}

// joinFirst joins at most n items and says how many were left out.
func joinFirst(items []string, n int) string {
	if len(items) <= n {
		return strings.Join(items, "，")
	}
	return strings.Join(items[:n], "，") + "，…"
}

func printDoctor(rep doctorReport) {
	fmt.Printf("Magic %s doctor\n", core.Version)
	fmt.Printf("配置库: %s\n", rep.MagicHome)
	fmt.Printf("目标主目录: %s\n", rep.UserHome)
	for _, s := range rep.sections {
		fmt.Printf("\n## %s\n", s.Title)
		for _, r := range s.Results {
			fmt.Printf("%-4s  %s\n", r.Status, r.Message)
			if r.Action != "" {
				fmt.Printf("      → %s\n", r.Action)
			}
		}
	}
	fmt.Printf("\n%d OK / %d WARN / %d FAIL\n", rep.OK, rep.Warn, rep.Fail)
}
