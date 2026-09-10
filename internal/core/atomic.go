package core

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

func secretBearingName(rel string) bool {
	base := strings.ToLower(filepath.Base(rel))
	switch base {
	case "auth.json", "settings.json", "settings.local.json",
		"opencode.json", "opencode.jsonc", "mcp.json", "mcp-config.json",
		"models.json", "claude.json.partial.json":
		return true
	}
	ext := strings.ToLower(filepath.Ext(rel))
	return ext == ".env" || ext == ".toml"
}

// AtomicWriteFile writes data via temp + rename.
func AtomicWriteFile(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp." + filepath.Base(path)
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// CopyFile copies src → dst (creates parent dirs).
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, dst)
}

// BackupPath copies existing path into backupDir preserving rel structure.
func BackupPath(absPath, backupDir, rel string) error {
	dst := filepath.Join(backupDir, rel)
	return CopyFile(absPath, dst)
}
