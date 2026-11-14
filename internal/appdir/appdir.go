// Package appdir resolves the per-user directories a harness reads its config
// from, so Magic adapters do not hard-code ~/.config.
//
// Source: spec | XDG Base Directory Specification
//
//	(https://specifications.freedesktop.org/basedir-spec/latest/)
//	"All paths set in these environment variables must be absolute. If an
//	implementation encounters a relative path in any of these variables it
//	should consider the path invalid and ignore it." checked 2026-11-14 | verified
//
// Two consequences are implemented here on purpose:
//   - a relative value is ignored, never joined onto the home directory;
//   - an absolute value is honoured whether or not the directory exists yet. The
//     spec makes no existence demand, a fresh machine has none of these, and
//     falling back to ~/.config when the XDG dir is merely missing would write
//     where the harness does not read.
package appdir

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	configHomeEnv = "XDG_CONFIG_HOME"
	dataHomeEnv   = "XDG_DATA_HOME"
)

// ConfigHome is the base config directory: $XDG_CONFIG_HOME when it holds an
// absolute path, else ~/.config under home.
func ConfigHome(home string) string {
	if dir, ok := absEnv(configHomeEnv); ok {
		return dir
	}
	return filepath.Join(home, ".config")
}

// DataHome is the base data directory: $XDG_DATA_HOME when it holds an absolute
// path, else ~/.local/share under home.
//
// No adapter mirrors a data dir today — OpenCode keeps auth.json and Kilo its
// relocated data dir there, both outside Magic's scope — but the rule belongs in
// one place rather than being re-derived by whichever adapter needs it first.
func DataHome(home string) string {
	if dir, ok := absEnv(dataHomeEnv); ok {
		return dir
	}
	return filepath.Join(home, ".local", "share")
}

// ExpandConfig expands an $HOME-relative path: a path under .config/ follows
// ConfigHome, anything else is returned unchanged (still $HOME-relative) so the
// caller joins it with home as before.
func ExpandConfig(home, rel string) string {
	rest, ok := strings.CutPrefix(filepath.ToSlash(rel), ".config/")
	if !ok {
		return rel
	}
	return filepath.Join(ConfigHome(home), filepath.FromSlash(rest))
}

// absEnv reads name and reports whether it holds a usable absolute path. An
// empty value is treated as unset (the usual "declared but empty" case).
func absEnv(name string) (string, bool) {
	v := os.Getenv(name)
	if v == "" || !filepath.IsAbs(v) {
		return "", false
	}
	return filepath.Clean(v), true
}
