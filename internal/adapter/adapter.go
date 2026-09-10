package adapter

// ID identifies a harness adapter.
type ID string

// Caps declares which projections an adapter supports.
type Caps struct {
	Skills bool
	Agents bool // sub-agent defs (packs/agents), NOT a harness named "agents"
	MCP    bool
	Models bool
	Themes bool
}

// RootSpec is a capture/apply root relative to $HOME.
type RootSpec struct {
	ID      string // "main", "userdata", …
	RelHome string // e.g. ".pi/agent"
}

// CapturePolicy is the adapter builtin allow/deny rules.
type CapturePolicy struct {
	Roots []RootSpec
	Allow []string
	Deny  []string
}

// Drift is a status difference between library and local.
type Drift struct {
	Adapter ID
	Path    string
	Kind    string // missing_local, missing_library, differ, stub
	Detail  string
}

// Result is an apply/capture outcome for one adapter.
type Result struct {
	Adapter ID
	OK      bool
	Skipped bool
	Message string
	Files   []string
}

// Adapter connects the config library to one harness's on-disk dialect.
type Adapter interface {
	ID() ID
	Name() string
	Platforms() []string
	Implemented() bool
	Caps() Caps
	Detect(home string) bool
	CapturePolicy() CapturePolicy
	SkillsDir(home string) string  // empty if no skills projection
	AgentsDir(home string) string  // empty if no sub-agent projection
	MirrorRoot(home string) string // primary mirror root for apply (rel to home abs)
	LibraryMirrorRel() string      // packs/harness/<id> relative segment
}
