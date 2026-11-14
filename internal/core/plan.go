package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"magic/internal/adapter"
	"magic/internal/backup"
	"magic/internal/catalog"
	"magic/internal/mcp"
	"magic/internal/provider"
	"magic/internal/secrets"
)

// PlanVersion is the schema version of the plan document. Bump it when a field
// changes meaning, not when a field is added.
const PlanVersion = 1

// Action is what apply does to one target file.
type Action string

const (
	ActionCreate    Action = "create"
	ActionUpdate    Action = "update"
	ActionUnchanged Action = "unchanged"
	// ActionDelete: apply removes the target. Codex official mode drops the
	// Magic-managed model catalog this way.
	ActionDelete Action = "delete"
)

// KeyOp is how one key of a structured config file changes.
type KeyOp string

const (
	KeyAdd       KeyOp = "add"
	KeyOverwrite KeyOp = "overwrite"
	KeyKeep      KeyOp = "keep"
)

// Fingerprint is a content digest. It never carries plaintext, so a plan can be
// logged, diffed or committed without leaking a key from the library. It is the
// same type manifest.json records, so plan and rollback compare like for like.
type Fingerprint = backup.Fingerprint

// FingerprintOf returns a short digest of b, in the form plan.json and
// manifest.json both record.
func FingerprintOf(b []byte) Fingerprint {
	return backup.FingerprintOf(b)
}

// KeyChange is one key in a merged structured config file.
//
// KeyKeep records a target-only key. Merge never deletes it (decisions.md D5),
// and the plan says so instead of leaving it out.
type KeyChange struct {
	Key string `json:"key"`
	Op  KeyOp  `json:"op"`
}

// LineChange is the line-level summary for a text-merged file.
type LineChange struct {
	Added   int `json:"added"`
	Removed int `json:"removed"`
}

// Change is one file that apply writes.
type Change struct {
	Adapter adapter.ID  `json:"adapter"`
	Path    string      `json:"path"`
	Rel     string      `json:"rel"`
	Action  Action      `json:"action"`
	Reason  string      `json:"reason"`
	Mode    string      `json:"mode"`
	Before  Fingerprint `json:"before,omitempty"`
	After   Fingerprint `json:"after,omitempty"`
	Backup  bool        `json:"backup,omitempty"`
	Keys    []KeyChange `json:"keys,omitempty"`
	Lines   *LineChange `json:"lines,omitempty"`
}

// AdapterPlan is the plan for one harness. Skipped is set instead of Changes
// when apply would write nothing for it.
type AdapterPlan struct {
	Adapter adapter.ID `json:"adapter"`
	Skipped string     `json:"skipped,omitempty"`
	Changes []Change   `json:"changes,omitempty"`
}

// Summary counts the changes in a plan.
type Summary struct {
	Files     int `json:"files"`
	Create    int `json:"create"`
	Update    int `json:"update"`
	Delete    int `json:"delete"`
	Unchanged int `json:"unchanged"`
}

// Plan is what apply would do, computed without writing anything.
type Plan struct {
	Version   int           `json:"version"`
	MagicHome string        `json:"magic_home"`
	UserHome  string        `json:"user_home"`
	Adapters  []AdapterPlan `json:"adapters"`
	Summary   Summary       `json:"summary"`
}

// JSON renders the plan as indented JSON with a trailing newline. HTML escaping
// is off so paths and Chinese reasons stay readable.
func (p Plan) JSON() ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(p); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// BuildPlan returns the plan for every adapter in adapters. It only reads files.
//
// The layers are the ones apply writes, in apply's order: the harness mirror
// tree, the shared instruction source (packs/AGENTS.md), the Claude partial merge
// into ~/.claude.json, the provider projection, then the MCP projection. Each
// layer merges into the previous layer's bytes, so the final fingerprint is what
// apply leaves on disk.
//
// Skills and sub-agent projection are not part of it; they are directory
// projections reported by `magic skills status` and `magic agents`.
func BuildPlan(adapters []adapter.Adapter, opts ApplyOptions) (Plan, error) {
	p := Plan{Version: PlanVersion, MagicHome: opts.MagicHome, UserHome: opts.UserHome}
	lib, err := loadProjectionLayers(opts)
	if err != nil {
		return Plan{}, err
	}
	for _, a := range adapters {
		if opts.Only != "" && string(a.ID()) != opts.Only {
			continue
		}
		ap, err := buildAdapterPlan(a, opts, lib)
		if err != nil {
			return Plan{}, fmt.Errorf("%s: %w", a.ID(), err)
		}
		p.Adapters = append(p.Adapters, ap)
		for _, c := range ap.Changes {
			p.Summary.Files++
			switch c.Action {
			case ActionCreate:
				p.Summary.Create++
			case ActionUpdate:
				p.Summary.Update++
			case ActionDelete:
				p.Summary.Delete++
			case ActionUnchanged:
				p.Summary.Unchanged++
			}
		}
	}
	sort.SliceStable(p.Adapters, func(i, j int) bool { return p.Adapters[i].Adapter < p.Adapters[j].Adapter })
	return p, nil
}

// projectionLayers is what the provider and MCP layers read once per plan: the
// effective provider catalog (builtin template under providers.yaml), the
// provider secrets, and packs/mcp/mcp.yaml. apply loads the same three, so plan
// resolves a provider or a server the same way apply does.
type projectionLayers struct {
	providers *catalog.File
	secrets   map[string]string
	mcp       *mcp.File
}

func loadProjectionLayers(opts ApplyOptions) (projectionLayers, error) {
	f, err := provider.LoadCatalog(opts.MagicHome)
	if err != nil {
		return projectionLayers{}, err
	}
	sec, err := secrets.LoadProviderSecrets(NewPaths(opts.MagicHome).Secrets)
	if err != nil {
		// ApplyProviders reports a broken secret store and projects nothing, so
		// plan resolves no key either: the two must agree on what apply writes.
		sec = map[string]string{}
	}
	mf, err := mcp.Load(filepath.Join(opts.MagicHome, filepath.FromSlash(mcp.CanonicalRelPath)))
	if err != nil {
		return projectionLayers{}, err
	}
	return projectionLayers{providers: f, secrets: sec, mcp: mf}, nil
}

// planFile is one target file while the layers are stacked onto it: the bytes on
// disk now, the bytes after the layers applied so far, and what wrote them.
type planFile struct {
	path   string
	rel    string
	mode   os.FileMode
	disk   []byte // content on disk now; nil when the file does not exist
	exists bool
	cur    []byte // content after the layers applied so far
	remove bool   // a layer deletes the target instead of writing it
	layers []string
	// keeps are the target's own top-level keys the library merges left in place.
	keeps []KeyChange
	lines bool
}

// planFiles is the set of simulated targets of one adapter, in first-touch order.
type planFiles struct {
	userHome   string
	mirrorRoot string
	byPath     map[string]*planFile
	order      []*planFile
}

func newPlanFiles(userHome, mirrorRoot string) *planFiles {
	return &planFiles{userHome: userHome, mirrorRoot: mirrorRoot, byPath: map[string]*planFile{}}
}

// read is how a projection reads a target while the plan stacks layers: the bytes
// the layers so far leave there, or the disk content of a file no layer touched.
// A missing file reports os.ErrNotExist, as os.ReadFile does.
func (pf *planFiles) read(path string) ([]byte, error) {
	if f, ok := pf.byPath[path]; ok && !f.remove {
		return f.cur, nil
	}
	return os.ReadFile(path)
}

// touch returns the simulated file for path, reading the disk content the first
// time it is seen. Later layers reuse it, so they merge into the previous
// layer's output.
func (pf *planFiles) touch(path string, mode os.FileMode) (*planFile, error) {
	if f, ok := pf.byPath[path]; ok {
		return f, nil
	}
	f := &planFile{path: path, rel: pf.rel(path), mode: mode}
	data, err := os.ReadFile(path)
	if err == nil {
		f.disk, f.exists, f.cur = data, true, data
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	pf.byPath[path] = f
	pf.order = append(pf.order, f)
	return f, nil
}

// discard forgets a target no layer writes, so it is not reported as a change.
// A target an earlier layer already wrote stays: only the projection on top of
// it turned out to write nothing.
func (pf *planFiles) discard(f *planFile) {
	if len(f.layers) > 0 {
		return
	}
	pf.remove(f)
}

func (pf *planFiles) remove(f *planFile) {
	delete(pf.byPath, f.path)
	for i, o := range pf.order {
		if o == f {
			pf.order = append(pf.order[:i], pf.order[i+1:]...)
			return
		}
	}
}

// rel names a target in the plan: the library-relative path for a mirrored file,
// the name apply and rollback use for ~/.claude.json, the user-home relative
// path for a file a layer names that way (provider, MCP, and the shared
// instruction layer, whose file sits inside the mirror root but is not mirrored),
// and the path under the mirror root otherwise.
func (pf *planFiles) rel(path string) string {
	if path == filepath.Join(pf.userHome, ".claude.json") {
		return backup.ClaudeJSONRel
	}
	if pf.mirrorRoot != "" {
		if rel, err := filepath.Rel(pf.mirrorRoot, path); err == nil && !escapesDir(rel) {
			return rel
		}
	}
	if rel, err := filepath.Rel(pf.userHome, path); err == nil && !escapesDir(rel) {
		return rel
	}
	return filepath.Base(path)
}

func escapesDir(rel string) bool {
	return rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// buildAdapterPlan stacks every layer for one adapter.
func buildAdapterPlan(a adapter.Adapter, opts ApplyOptions, lib projectionLayers) (AdapterPlan, error) {
	ap := AdapterPlan{Adapter: a.ID()}
	if !a.Implemented() {
		ap.Skipped = "adapter 未实现"
		return ap, nil
	}
	// The same gate apply uses: an adapter Detect does not find on the user home
	// gets nothing, so plan reports it skipped instead of listing its library
	// files as missing targets. --only bypasses the gate, the way it lets apply
	// pre-install a harness.
	if !a.Detect(opts.UserHome) && opts.Only != string(a.ID()) {
		ap.Skipped = "未安装"
		return ap, nil
	}
	mirrorRoot := a.MirrorRoot(opts.UserHome)
	files := newPlanFiles(opts.UserHome, mirrorRoot)
	srcRoot := filepath.Join(opts.MagicHome, a.LibraryMirrorRel())
	// Only the mirror layer needs the library tree. A missing tree must not skip
	// the provider and MCP layers: most libraries hold no packs/harness/codex,
	// yet apply still writes ~/.codex/config.toml through the MCP projection.
	mirrorProblem := ""
	if mirrorRoot == "" {
		mirrorProblem = "该 adapter 没有镜像目录"
	} else if st, err := os.Stat(srcRoot); err != nil || !st.IsDir() {
		mirrorProblem = "配置库里没有镜像目录 " + srcRoot
	} else if err := mirrorLayer(files, srcRoot); err != nil {
		return ap, err
	}
	if err := sharedInstructionLayer(a, opts, files); err != nil {
		return ap, err
	}
	if a.ID() == "claude" {
		if err := claudePartialLayer(opts, files); err != nil {
			return ap, err
		}
	}
	if err := providerLayer(a, opts, lib, files); err != nil {
		return ap, err
	}
	if err := mcpLayer(a, opts, lib, files); err != nil {
		return ap, err
	}
	if len(files.order) == 0 {
		// Nothing in the library mirror and no projection wrote this adapter: it
		// was not checked, and the plan says so instead of claiming it is clean.
		ap.Skipped = nothingToWriteReason(a, mirrorProblem)
		return ap, nil
	}
	for _, f := range files.order {
		ap.Changes = append(ap.Changes, f.change(a.ID()))
	}
	sort.SliceStable(ap.Changes, func(i, j int) bool { return ap.Changes[i].Rel < ap.Changes[j].Rel })
	return ap, nil
}

// nothingToWriteReason explains an adapter apply writes no file for. It names
// every fact that decided it, because the missing mirror tree alone reads as
// "put packs/harness/codex into the library and apply manages codex" — while
// apply never projects a provider into codex or grok, only `magic provider use`
// writes those.
func nothingToWriteReason(a adapter.Adapter, mirrorProblem string) string {
	var parts []string
	if mirrorProblem != "" {
		parts = append(parts, mirrorProblem)
	}
	if !providerProjectionHarnesses[string(a.ID())] {
		parts = append(parts, "apply 不投影该 adapter 的 provider（只有 magic provider use 会写）")
	}
	if len(parts) == 0 {
		return "库内镜像目录为空，provider / MCP 层也没有要写的文件"
	}
	return strings.Join(parts, "；")
}

// mirrorLayer stages layer 1: every file in the library mirror tree merged into
// its target by the rule apply uses (projectBytes).
func mirrorLayer(files *planFiles, srcRoot string) error {
	return filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		// Same exclusions as ApplyAdapter: partial metadata files are not
		// mirrored into the harness root as-is.
		if base == "claude.json.partial.json" || base == "_claude.json" {
			return nil
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return nil
		}
		srcData, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		if secretBearingName(rel) {
			mode = 0o600
		}
		f, err := files.touch(filepath.Join(files.mirrorRoot, rel), mode)
		if err != nil {
			return err
		}
		out, strategy, err := projectBytes(rel, srcData, f.cur, f.exists)
		if err != nil {
			return err
		}
		if strategy == StrategyJSONMerge {
			f.keeps = append(f.keeps, topLevelKeeps(f.disk, srcData)...)
		}
		if strategy == StrategyTOMLMerge {
			f.lines = true
		}
		f.cur = out
		f.layers = append(f.layers, mirrorLayerReason(strategy))
		return nil
	})
}

// claudePartialLayer stages layer 2: the Claude custom partial merged into
// ~/.claude.json. It calls the same claudePartialProjected as applyClaudePartial.
func claudePartialLayer(opts ApplyOptions, files *planFiles) error {
	src := filepath.Join(opts.MagicHome, claudePartialFile)
	srcData, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	dst := filepath.Join(opts.UserHome, ".claude.json")
	f, err := files.touch(dst, 0o600)
	if err != nil {
		return err
	}
	out, err := claudePartialProjected(srcData, f.cur, f.exists)
	if err != nil {
		return err
	}
	f.keeps = append(f.keeps, topLevelKeeps(f.disk, srcData)...)
	f.cur, f.mode = out, 0o600
	f.layers = append(f.layers, "claude partial 深合并")
	return nil
}

// providerLayer stages layer 3: the projection ApplyProviders performs. It calls
// the harness's SwitchWriter.Files — the same computation apply writes — so the
// plan cannot predict a projection apply would not make.
//
// The gates repeat ApplyProviders: only a harness apply projects is eligible,
// only a detected adapter is projected unless --only names it, and a provider
// without a resolvable key writes nothing.
func providerLayer(a adapter.Adapter, opts ApplyOptions, lib projectionLayers, files *planFiles) error {
	id := string(a.ID())
	if !providerProjectionHarnesses[id] {
		return nil
	}
	w := provider.Writer(id)
	if w == nil || !w.Implemented() {
		return nil
	}
	if !a.Detect(opts.UserHome) && opts.Only != id {
		return nil
	}
	req, why := providerProjectionRequest(w, lib.providers, lib.secrets)
	if why != "" {
		return nil
	}
	ctx := provider.Context{
		MagicHome: opts.MagicHome,
		UserHome:  opts.UserHome,
		Catalog:   lib.providers,
		ReadFile:  files.read,
	}
	writes, err := w.Files(ctx, req)
	if err != nil {
		return err
	}
	for _, fw := range writes {
		f, err := files.touch(fw.Path, fw.Mode)
		if err != nil {
			return err
		}
		f.layers = append(f.layers, providerLayerReason(fw))
		if fw.Remove {
			f.remove = true
			continue
		}
		f.cur, f.mode = fw.Data, fw.Mode
	}
	return nil
}

// ProviderLayerLabel prefixes the reason of a file the provider projection
// wrote. Change.Reason joins layer labels with " → ", so a reader that needs to
// tell a provider-driven file from a mirror file matches this one spelling.
const ProviderLayerLabel = "provider 投影"

// providerLayerReason names the provider layer by the file it wrote: the env
// block inside a shared config, or the file's own name.
func providerLayerReason(fw provider.FileWrite) string {
	if fw.Role == provider.RoleSettingsEnv {
		return ProviderLayerLabel + " env"
	}
	return ProviderLayerLabel + " " + filepath.Base(fw.Path)
}

// mcpLayer stages layer 4: the projection mcp.Apply performs for one adapter,
// merged into whatever the earlier layers left in the target file.
func mcpLayer(a adapter.Adapter, opts ApplyOptions, lib projectionLayers, files *planFiles) error {
	id := string(a.ID())
	if len(lib.mcp.MCP) == 0 || !a.Caps().MCP {
		return nil
	}
	if !a.Detect(opts.UserHome) && opts.Only != id {
		return nil
	}
	t, ok := mcp.TargetFor(id)
	if !ok {
		return nil
	}
	path := t.Path(opts.UserHome)
	f, err := files.touch(path, 0o600)
	if err != nil {
		return err
	}
	out, err := mcp.MergeProjection(t, path, lib.mcp, f.cur)
	if err != nil {
		return err
	}
	if out == nil {
		files.discard(f)
		return nil
	}
	if t.Format == mcp.FormatTOML {
		f.lines = true
	}
	f.cur, f.mode = out, 0o600
	f.layers = append(f.layers, "mcp 投影")
	return nil
}

// change turns the simulated file into the plan's report. Before is the disk
// fingerprint, After the final one; reason names the layers that wrote it.
func (f *planFile) change(id adapter.ID) Change {
	c := Change{
		Adapter: id,
		Path:    f.path,
		Rel:     f.rel,
		Mode:    backup.ModeString(f.mode),
		After:   FingerprintOf(f.cur),
		Reason:  strings.Join(f.layers, " → "),
	}
	if f.exists {
		c.Before = FingerprintOf(f.disk)
	}
	switch {
	case f.remove:
		c.Action = ActionDelete
		c.After = ""
		c.Backup = true
	case !f.exists:
		c.Action = ActionCreate
	case bytes.Equal(f.cur, f.disk):
		c.Action = ActionUnchanged
		c.Reason += "（结果与磁盘一致）"
	default:
		c.Action = ActionUpdate
		c.Backup = true
	}
	c.Keys = f.keyChanges()
	if f.lines && c.Action != ActionDelete {
		lines := diffLines(f.disk, f.cur)
		c.Lines = &lines
	}
	return c
}

// keyChanges describes the keys of a structured config file the way the plan
// documents them: the keys the final content adds or overwrites against what the
// disk holds now, plus the target's own top-level keys a merge left in place.
//
// The base is the disk, not the layer input. A file whose final bytes equal the
// disk bytes is rewritten by nothing, so it reports no key change at all —
// otherwise an unchanged target would claim to overwrite the keys apply already
// wrote.
func (f *planFile) keyChanges() []KeyChange {
	if f.remove || (f.exists && bytes.Equal(f.cur, f.disk)) {
		return nil
	}
	return mergedKeyChanges(append(diffJSONKeys(f.disk, f.cur), f.keeps...))
}

// mirrorLayerReason names layer 1 by the strategy projectBytes actually used, so
// a plan never claims a merge that apply would not perform.
func mirrorLayerReason(s ProjectStrategy) string {
	switch s {
	case StrategyCreate:
		return "镜像新建"
	case StrategyJSONMerge:
		return "镜像 JSON 合并"
	case StrategyJSONFallback:
		return "镜像整文件覆盖（目标不是可解析的 JSON）"
	case StrategyTOMLMerge:
		return "镜像 TOML 合并"
	}
	return "镜像整文件覆盖"
}

// mergedKeyChanges flattens the key diffs of every layer. A later layer wins the
// same key, and the result is sorted by key.
func mergedKeyChanges(all []KeyChange) []KeyChange {
	if len(all) == 0 {
		return nil
	}
	last := map[string]KeyOp{}
	for _, k := range all {
		last[k.Key] = k.Op
	}
	out := make([]KeyChange, 0, len(last))
	for key, op := range last {
		out = append(out, KeyChange{Key: key, Op: op})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// diffJSONKeys lists the keys dst gains or changes on the way to src. Nested
// objects recurse; arrays and scalars stop at their path. Keys whose value is the
// same on both sides are left out: they are not a change. A missing target is an
// empty object, so a file apply creates reports every key it starts with.
func diffJSONKeys(dstData, srcData []byte) []KeyChange {
	dst := map[string]any{}
	if len(dstData) > 0 {
		if json.Unmarshal(secrets.StripJSONC(dstData), &dst) != nil {
			return nil
		}
	}
	var src map[string]any
	if json.Unmarshal(secrets.StripJSONC(srcData), &src) != nil {
		return nil
	}
	var out []KeyChange
	collectKeyChanges(dst, src, "", &out)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func collectKeyChanges(dst, src map[string]any, prefix string, out *[]KeyChange) {
	for _, k := range sortedKeys(src) {
		path := joinKey(prefix, k)
		dstVal, inDst := dst[k]
		if !inDst {
			// A new object expands to the keys it starts with; an array or a
			// scalar stops at its own path.
			if srcMap, ok := src[k].(map[string]any); ok {
				collectKeyChanges(map[string]any{}, srcMap, path, out)
				continue
			}
			*out = append(*out, KeyChange{Key: path, Op: KeyAdd})
			continue
		}
		dstMap, dstIsMap := dstVal.(map[string]any)
		srcMap, srcIsMap := src[k].(map[string]any)
		if dstIsMap && srcIsMap {
			collectKeyChanges(dstMap, srcMap, path, out)
			continue
		}
		if !reflect.DeepEqual(dstVal, src[k]) {
			*out = append(*out, KeyChange{Key: path, Op: KeyOverwrite})
		}
	}
}

// topLevelKeeps lists the target's own top-level keys a library merge leaves in
// place: the library defines none of them and merge never deletes them
// (decisions.md D5), so the plan says so instead of leaving them out.
func topLevelKeeps(dstData, srcData []byte) []KeyChange {
	var dst, src map[string]any
	if json.Unmarshal(secrets.StripJSONC(dstData), &dst) != nil {
		return nil
	}
	if json.Unmarshal(secrets.StripJSONC(srcData), &src) != nil {
		return nil
	}
	var out []KeyChange
	for _, k := range sortedKeys(dst) {
		if _, ok := src[k]; !ok {
			out = append(out, KeyChange{Key: k, Op: KeyKeep})
		}
	}
	return out
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func joinKey(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

// diffLines counts lines added and removed between two texts. It is a line
// multiset difference, not an ordered diff: the count is exact, the position is
// not reported.
func diffLines(before, after []byte) LineChange {
	beforeCount := lineCounts(before)
	afterCount := lineCounts(after)
	var lc LineChange
	for line, n := range afterCount {
		if have := beforeCount[line]; n > have {
			lc.Added += n - have
		}
	}
	for line, n := range beforeCount {
		if have := afterCount[line]; n > have {
			lc.Removed += n - have
		}
	}
	return lc
}

func lineCounts(b []byte) map[string]int {
	counts := map[string]int{}
	for _, line := range bytes.Split(b, []byte("\n")) {
		counts[string(line)]++
	}
	return counts
}
