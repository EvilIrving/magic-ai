// Package backup records what an apply run wrote into backups/<stamp>/ so that
// rollback can undo it exactly: restore what apply overwrote, delete what apply
// created.
//
// It is a leaf package: core (plan/apply/rollback) and the MCP projection both
// write manifests without depending on each other.
package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// ManifestFileName is the index of one backup set, next to the copied files.
const ManifestFileName = "manifest.json"

// ManifestVersion is the schema version of manifest.json.
const ManifestVersion = 1

// ClaudeJSONRel is the backup name of the merged ~/.claude.json. It lives
// outside Claude's mirror root, so it is addressed by name; apply's claude
// partial merge and the MCP projection must agree on this one spelling.
const ClaudeJSONRel = "_claude.json"

// What apply did to one target.
const (
	// ActionCreate: the target did not exist; rollback deletes it.
	ActionCreate = "create"
	// ActionUpdate: the target existed and was copied into the set; rollback
	// restores the copy.
	ActionUpdate = "update"
)

// Entry is one target apply wrote.
type Entry struct {
	Adapter string `json:"adapter"`
	Rel     string `json:"rel"`
	Target  string `json:"target"`
	Backup  string `json:"backup"`
	Action  string `json:"action"`
	Mode    string `json:"mode"`
	// Before is the content apply replaced: the fingerprint of the copy taken
	// into the set. A create entry has no before.
	Before Fingerprint `json:"before,omitempty"`
	// After is the content apply wrote. rollback only restores or deletes a
	// target that still holds it; an entry without it (an older manifest) cannot
	// be checked and needs --force.
	After Fingerprint `json:"after,omitempty"`
}

// Manifest is the index of one backup set.
type Manifest struct {
	Version int     `json:"version"`
	Stamp   string  `json:"stamp"`
	Entries []Entry `json:"entries"`
}

// Read returns the manifest of the backup set in dir. A set without one is an
// old backup (written before manifests existed): it returns nil, nil.
func Read(dir string) (*Manifest, error) {
	path := filepath.Join(dir, ManifestFileName)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &m, nil
}

// Record adds e to the backup set in dir, creating the manifest on the first
// write of a set and appending afterwards. An update entry copies the current
// target into the set before it is recorded and fills Before from that copy;
// the caller sets After from the bytes it is about to write.
//
// A target is recorded once per set. apply and the MCP projection can both touch
// one file in a single run; rollback must see the copy taken before the first
// write, not a copy of apply's own output.
func Record(dir string, e Entry) error {
	m, err := Read(dir)
	if err != nil {
		return err
	}
	if m == nil {
		m = &Manifest{Version: ManifestVersion, Stamp: filepath.Base(dir)}
	}
	for _, have := range m.Entries {
		if have.Target == e.Target {
			return nil
		}
	}
	if e.Action == ActionUpdate {
		if e.Backup == "" {
			return fmt.Errorf("backup: update entry for %s has no backup path", e.Target)
		}
		before, err := copyFile(e.Target, filepath.Join(dir, e.Backup))
		if err != nil {
			return err
		}
		e.Before = before
	}
	m.Entries = append(m.Entries, e)
	return write(dir, m)
}

// ModeString renders a file mode the way plan and manifest.json record it.
func ModeString(mode os.FileMode) string {
	return fmt.Sprintf("%04o", mode.Perm())
}

// ParseMode reads back ModeString.
func ParseMode(s string) (os.FileMode, bool) {
	v, err := strconv.ParseUint(s, 8, 32)
	if err != nil {
		return 0, false
	}
	return os.FileMode(v).Perm(), true
}

func write(dir string, m *Manifest) error {
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, ManifestFileName)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// copyFile copies src to dst and returns the fingerprint of what it copied, so
// an update entry's before fingerprint is exactly the content saved in the set.
func copyFile(src, dst string) (Fingerprint, error) {
	data, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(src)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(dst, data, info.Mode().Perm()); err != nil {
		return "", err
	}
	if err := os.Chmod(dst, info.Mode().Perm()); err != nil {
		return "", err
	}
	return FingerprintOf(data), nil
}
