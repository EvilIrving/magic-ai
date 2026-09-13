// Package stub holds unimplemented harness placeholders.
//
// Source: internal | docs/supported-harnesses.md (capability matrix) | verified
// A stub reports Implemented()=false so `magic status` shows n/a instead of
// pretending a projection exists. Never: let a stub write files.
package stub

import (
	"path/filepath"

	"magic/internal/adapter"
)

// Adapter is an unimplemented harness placeholder.
type Adapter struct {
	id   adapter.ID
	name string
}

func New(id, name string) *Adapter {
	return &Adapter{id: adapter.ID(id), name: name}
}

func (a *Adapter) ID() adapter.ID                { return a.id }
func (a *Adapter) Name() string                  { return a.name }
func (a *Adapter) Platforms() []string           { return []string{"any"} }
func (a *Adapter) Implemented() bool             { return false }
func (a *Adapter) Caps() adapter.Caps            { return adapter.Caps{} }
func (a *Adapter) Detect(home string) bool       { return false }
func (a *Adapter) SkillsDir(home string) string  { return "" }
func (a *Adapter) AgentsDir(home string) string  { return "" }
func (a *Adapter) MirrorRoot(home string) string { return "" }
func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", string(a.id))
}
