package provider

import (
	"encoding/json"
	"fmt"
	"path/filepath"
)

// ProjectionStateFile is the Magic bookkeeping sidecar a projection writes when
// it has to displace a value the harness (not Magic) owns: Kimi's default_model,
// Vibe's active_model, Hermes's model block.
//
// Why a sidecar instead of guessing on clear: `provider use none` must put the
// file back the way the user had it ("库里没有的东西不动"), and the harness itself
// may have rewritten those keys since. Recording the value Magic displaced is
// the only way to restore it without pattern-matching someone else's data.
//
// The sidecar is a Magic-managed helper file (RoleSidecar): a projection writes
// it, clearing removes it.
const ProjectionStateFile = "magic-provider-state.json"

// projectionState records, per dialect field, what the harness config held
// before Magic first wrote that field, and what Magic wrote last.
//
// Displaced: a nil value means "the key was absent"; a string means "the key
// held this". Written lets a clear tell "Magic's value is still there" from
// "the user chose something else after the projection", in which case their
// choice wins.
type projectionState struct {
	Displaced map[string]*string `json:"displaced,omitempty"`
	Written   map[string]string  `json:"written,omitempty"`
}

// ProjectionStatePath is the sidecar path inside a harness's own directory.
func ProjectionStatePath(dir string) string {
	return filepath.Join(dir, ProjectionStateFile)
}

// loadProjectionState reads the sidecar (missing → empty state).
func loadProjectionState(ctx Context, path string) (projectionState, error) {
	raw, err := readTarget(ctx, path)
	if err != nil {
		return projectionState{}, err
	}
	st := projectionState{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &st); err != nil {
			return projectionState{}, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
		}
	}
	return st, nil
}

// remember records what key held the first time Magic displaced it, and what
// Magic writes now. Later projections keep the original, so re-running
// `provider use` does not make a Magic value look like the user's own.
func (s *projectionState) remember(key, current string, present bool, written string) {
	if s == nil {
		return
	}
	if s.Displaced == nil {
		s.Displaced = map[string]*string{}
	}
	if s.Written == nil {
		s.Written = map[string]string{}
	}
	if _, seen := s.Displaced[key]; !seen {
		if !present {
			s.Displaced[key] = nil
		} else {
			value := current
			s.Displaced[key] = &value
		}
	}
	s.Written[key] = written
}

// prior returns the value key held before Magic took it over.
func (s projectionState) prior(key string) (string, bool) {
	value, ok := s.Displaced[key]
	if !ok || value == nil {
		return "", false
	}
	return *value, true
}

// touched reports whether Magic ever displaced key.
func (s projectionState) touched(key string) bool {
	_, ok := s.Displaced[key]
	return ok
}

// stillMagic reports whether the field still holds what Magic wrote. A value
// the user changed afterwards is theirs, and a clear leaves it alone.
func (s projectionState) stillMagic(key, current string) bool {
	written, ok := s.Written[key]
	return !ok || written == current
}

// file renders the sidecar as one projection write.
func (s projectionState) file(path string) (FileWrite, error) {
	out, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return FileWrite{}, err
	}
	return FileWrite{Path: path, Data: append(out, '\n'), Mode: 0o600, Role: RoleSidecar}, nil
}

// clearFile is the sidecar write that removes it: once a harness is cleared,
// Magic owns nothing there and has nothing left to restore.
func clearFile(path string) FileWrite {
	return FileWrite{Path: path, Mode: 0o600, Role: RoleSidecar, Remove: true}
}
