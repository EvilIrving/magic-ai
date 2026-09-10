package policy

import (
	"os"

	"gopkg.in/yaml.v3"

	"magic/internal/adapter"
)

// OverlayHarness is per-adapter user overlay.
type OverlayHarness struct {
	Include    []string `yaml:"include,omitempty" json:"include,omitempty"`
	Exclude    []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
	ExtraRoots []string `yaml:"extra_roots,omitempty" json:"extra_roots,omitempty"`
}

// Overlay is the vault-synced capture.overlay.yaml content.
type Overlay struct {
	Harnesses map[string]OverlayHarness `yaml:"harnesses" json:"harnesses"`
}

// LoadOverlay loads YAML overlay; missing file → empty.
func LoadOverlay(path string) (*Overlay, error) {
	o := &Overlay{Harnesses: map[string]OverlayHarness{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return o, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(data, o); err != nil {
		return nil, err
	}
	if o.Harnesses == nil {
		o.Harnesses = map[string]OverlayHarness{}
	}
	return o, nil
}

// SaveOverlay writes overlay atomically-ish via rename.
func SaveOverlay(path string, o *Overlay) error {
	if o.Harnesses == nil {
		o.Harnesses = map[string]OverlayHarness{}
	}
	data, err := yaml.Marshal(o)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Classify returns decision ("allow"|"deny"|"unknown") and reason.
// Precedence: user.exclude > user.include > builtin.deny > builtin.allow > unknown DENY.
func Classify(rel string, builtin adapter.CapturePolicy, overlay *OverlayHarness) (decision, reason string) {
	ov := OverlayHarness{}
	if overlay != nil {
		ov = *overlay
	}
	if MatchAny(rel, ov.Exclude) {
		return "deny", "user.exclude"
	}
	if MatchAny(rel, ov.Include) {
		return "allow", "user.include"
	}
	if MatchAny(rel, builtin.Deny) {
		return "deny", "builtin.deny"
	}
	if MatchAny(rel, builtin.Allow) {
		return "allow", "builtin.allow"
	}
	return "unknown", "neither → default deny"
}

// Verdict is one classified file.
type Verdict struct {
	Harness  string `json:"harness"`
	AbsPath  string `json:"abs_path"`
	Rel      string `json:"rel"`
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
	Root     string `json:"root"`
}

// Report aggregates triage results.
type Report struct {
	ScannedRoots []string  `json:"scanned_roots"`
	MissingRoots []string  `json:"missing_roots"`
	Allow        []Verdict `json:"allow"`
	Deny         []Verdict `json:"deny"`
	Unknown      []Verdict `json:"unknown"`
}

// EffectiveOverlay returns overlay for harness name or empty.
func (o *Overlay) EffectiveOverlay(name string) *OverlayHarness {
	if o == nil || o.Harnesses == nil {
		return nil
	}
	h, ok := o.Harnesses[name]
	if !ok {
		return nil
	}
	return &h
}

// Approve adds rel to include (and removes from exclude if present).
func (o *Overlay) Approve(adapterID, rel string) {
	if o.Harnesses == nil {
		o.Harnesses = map[string]OverlayHarness{}
	}
	h := o.Harnesses[adapterID]
	h.Exclude = removeString(h.Exclude, rel)
	if !containsString(h.Include, rel) {
		h.Include = append(h.Include, rel)
	}
	o.Harnesses[adapterID] = h
}

// DenyPath adds rel to exclude (and removes from include if present).
func (o *Overlay) DenyPath(adapterID, rel string) {
	if o.Harnesses == nil {
		o.Harnesses = map[string]OverlayHarness{}
	}
	h := o.Harnesses[adapterID]
	h.Include = removeString(h.Include, rel)
	if !containsString(h.Exclude, rel) {
		h.Exclude = append(h.Exclude, rel)
	}
	o.Harnesses[adapterID] = h
}

func containsString(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func removeString(ss []string, s string) []string {
	out := ss[:0]
	for _, x := range ss {
		if x != s {
			out = append(out, x)
		}
	}
	return out
}
