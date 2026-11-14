// Package writefile writes a projection target only when its content or
// permission differs from what the projection computed, so "unchanged" in a
// plan means apply really touches nothing.
//
// It is a leaf package: core (mirror, claude partial) and provider (projection
// files) both write through it instead of each keeping its own copy.
package writefile

import (
	"bytes"
	"os"
	"path/filepath"
)

// NeedsWrite reports whether Write would touch path: a missing target, different
// content, or a different permission all need a write.
func NeedsWrite(path string, data []byte, mode os.FileMode) bool {
	existing, err := os.ReadFile(path)
	if err != nil {
		return true
	}
	if !bytes.Equal(existing, data) {
		return true
	}
	info, err := os.Stat(path)
	return err != nil || info.Mode().Perm() != mode.Perm()
}

// Write writes data to path only when NeedsWrite says so. An unchanged target
// keeps its inode, mtime and mode. Content that already matches is only
// re-permissioned, so a mode fix does not rewrite the file.
func Write(path string, data []byte, mode os.FileMode) error {
	if !NeedsWrite(path, data, mode) {
		return nil
	}
	existing, err := os.ReadFile(path)
	if err == nil && bytes.Equal(existing, data) {
		return os.Chmod(path, mode.Perm())
	}
	return Atomic(path, data, mode)
}

// Atomic writes data via temp file + rename, creating parent directories.
func Atomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp." + filepath.Base(path)
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	// The rename carries the temp file's creation mode, which the process umask
	// narrows; set the requested mode so a projection's permission is exact.
	return os.Chmod(path, mode.Perm())
}
