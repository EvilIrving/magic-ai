package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BackupDir returns $MAGIC_HOME/backups/<harness>/.
func BackupDir(magicHome, harness string) string {
	return filepath.Join(magicHome, "backups", NormalizeHarness(harness))
}

// ListBackups returns backup files newest-first (by name mtime preference: modtime).
func ListBackups(magicHome, harness string) ([]string, error) {
	dir := BackupDir(magicHome, harness)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	type item struct {
		path string
		mod  time.Time
	}
	var items []item
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Accept settings.*.json, config.*.toml, *.bak
		if !strings.Contains(name, ".") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		items = append(items, item{path: filepath.Join(dir, name), mod: info.ModTime()})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].mod.After(items[j].mod) })
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.path
	}
	return out, nil
}

// CreateBackup copies src into MAGIC_HOME/backups/<harness>/ with stamped name.
// prefix should be "settings" or "config". Ext includes the dot (".json", ".toml").
// Trims to BackupLimit after create. Returns backup path (empty if src missing).
func CreateBackup(magicHome, harness, src, prefix, ext string) (string, error) {
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	dir := BackupDir(magicHome, harness)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	stamp := time.Now().UTC().Format("20060102-150405")
	name := fmt.Sprintf("%s.%s%s", prefix, stamp, ext)
	dst := filepath.Join(dir, name)
	data, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		return "", err
	}
	if err := TrimBackups(magicHome, harness, BackupLimit); err != nil {
		return dst, err
	}
	return dst, nil
}

// TrimBackups keeps the newest limit backups; deletes older ones.
func TrimBackups(magicHome, harness string, limit int) error {
	files, err := ListBackups(magicHome, harness)
	if err != nil {
		return err
	}
	if limit < 0 {
		limit = 0
	}
	for i := limit; i < len(files); i++ {
		_ = os.Remove(files[i])
	}
	return nil
}

// RestoreLatest copies the newest backup over destPath (creates parent dirs).
// Also creates a pre-restore backup of dest if it exists.
func RestoreLatest(magicHome, harness, destPath, prefix, ext string) (restoredFrom string, err error) {
	files, err := ListBackups(magicHome, harness)
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no configuration backup is available to restore for %s", harness)
	}
	src := files[0]
	if _, err := CreateBackup(magicHome, harness, destPath, prefix, ext); err != nil {
		return "", err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return "", err
	}
	tmp := destPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, destPath); err != nil {
		return "", err
	}
	_ = TrimBackups(magicHome, harness, BackupLimit)
	return src, nil
}

// ClearBackups removes all backups for harness.
func ClearBackups(magicHome, harness string) (int, error) {
	files, err := ListBackups(magicHome, harness)
	if err != nil {
		return 0, err
	}
	for _, f := range files {
		_ = os.Remove(f)
	}
	dir := BackupDir(magicHome, harness)
	_ = os.Remove(dir) // only if empty
	return len(files), nil
}
