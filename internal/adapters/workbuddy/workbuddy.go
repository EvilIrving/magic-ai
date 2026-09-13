package workbuddy

import (
	"os"
	"path/filepath"

	"magic/internal/adapter"
)

// Adapter mirrors WorkBuddy / CodeBuddy user-level models.json.
// Primary path is ~/.codebuddy; ~/.workbuddy is also detected for presence.
type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() adapter.ID      { return "workbuddy" }
func (a *Adapter) Name() string        { return "WorkBuddy" }
func (a *Adapter) Platforms() []string { return []string{"any"} }
func (a *Adapter) Implemented() bool   { return true }
func (a *Adapter) Caps() adapter.Caps {
	return adapter.Caps{Models: true}
}

func codebuddyRoot(home string) string { return filepath.Join(home, ".codebuddy") }
func workbuddyRoot(home string) string { return filepath.Join(home, ".workbuddy") }

func (a *Adapter) Detect(home string) bool {
	root := codebuddyRoot(home)
	if st, err := os.Stat(root); err == nil && st.IsDir() {
		return true
	}
	_, err := os.Stat(filepath.Join(root, "models.json"))
	return err == nil
}

func (a *Adapter) SkillsDir(home string) string { return "" }
func (a *Adapter) AgentsDir(home string) string { return "" }

func (a *Adapter) MirrorRoot(home string) string {
	// Prefer CodeBuddy path (DeepSeek / official integration docs).
	cb := codebuddyRoot(home)
	if st, err := os.Stat(cb); err == nil && st.IsDir() {
		return cb
	}
	if _, err := os.Stat(filepath.Join(cb, "models.json")); err == nil {
		return cb
	}
	wb := workbuddyRoot(home)
	if st, err := os.Stat(wb); err == nil && st.IsDir() {
		return wb
	}
	return cb
}

func (a *Adapter) LibraryMirrorRel() string {
	return filepath.Join("packs", "harness", "workbuddy")
}
