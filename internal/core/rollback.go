package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"magic/internal/adapter"
	"magic/internal/backup"
	"magic/internal/writefile"
)

// ClaudeJSONBackupRel is the backup path of the merged ~/.claude.json. It is not
// under any adapter mirror root, so rollback maps it by name.
const ClaudeJSONBackupRel = backup.ClaudeJSONRel

// BackupSet is one apply backup directory under backups/.
type BackupSet struct {
	Stamp    string   `json:"stamp"`
	Adapters []string `json:"adapters"`
	Files    int      `json:"files"`
}

// Restore is one target put back or deleted by rollback.
type Restore struct {
	Stamp   string     `json:"stamp"`
	Adapter adapter.ID `json:"adapter"`
	Rel     string     `json:"rel"`
	Path    string     `json:"path"`
	Bytes   int        `json:"bytes"`
	// Delete marks a target apply created: it has no backup file, so rollback
	// removes it.
	Delete bool `json:"delete,omitempty"`
}

// RollbackResult is what one rollback did to the backup set it replayed.
type RollbackResult struct {
	// Stamp is the backup set that was used, whether or not any entry ran.
	Stamp string
	// Restored lists the targets put back or deleted.
	Restored []Restore
	// Skipped lists the entries --force would still apply, plus notes about the
	// set itself (an old backup without a manifest).
	Skipped []string
	// Refused lists the entries rollback never applies, --force included: the
	// target is outside the user home, the copy is outside the backup set, or the
	// manifest names no copy to replay.
	Refused []string
}

// RollbackOptions controls rollback.
type RollbackOptions struct {
	MagicHome string
	UserHome  string
	Stamp     string // empty: the newest set
	Adapter   string // empty: every adapter in the set
	DryRun    bool
	// Force skips the content check: rollback overwrites or deletes even when the
	// target no longer holds what apply wrote. The path range check still holds.
	Force bool
}

// ListBackups returns the apply backup sets under MAGIC_HOME/backups, oldest
// first. Only directories named like a backup stamp are sets; a directory named
// after a harness belongs to the provider layer (backups/<harness>/) and is not
// a rollback candidate.
func ListBackups(magicHome string) ([]BackupSet, error) {
	stamps, _, err := listBackupDirs(magicHome)
	if err != nil {
		return nil, err
	}
	root := filepath.Join(magicHome, "backups")
	var sets []BackupSet
	for _, stamp := range stamps {
		set := BackupSet{Stamp: stamp}
		dir := filepath.Join(root, stamp)
		adapters, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, a := range adapters {
			if !a.IsDir() {
				continue
			}
			set.Adapters = append(set.Adapters, a.Name())
			files, err := countFiles(filepath.Join(dir, a.Name()))
			if err != nil {
				return nil, err
			}
			set.Files += files
		}
		// A manifest knows better: a create entry has no copy, and an adapter can
		// appear only in the manifest.
		if m, err := backup.Read(dir); err != nil {
			return nil, err
		} else if m != nil {
			set.Files = len(m.Entries)
			set.Adapters = nil
			seen := map[string]bool{}
			for _, e := range m.Entries {
				if !seen[e.Adapter] {
					seen[e.Adapter] = true
					set.Adapters = append(set.Adapters, e.Adapter)
				}
			}
		}
		sort.Strings(set.Adapters)
		sets = append(sets, set)
	}
	return sets, nil
}

// ListProviderBackupDirs returns the backup directories that are not apply sets:
// the provider layer keeps backups/<harness>/ with flat, stamped file names.
// They are reported separately because rollback does not read them.
func ListProviderBackupDirs(magicHome string) ([]string, error) {
	_, others, err := listBackupDirs(magicHome)
	return others, err
}

func listBackupDirs(magicHome string) (stamps []string, others []string, err error) {
	root := filepath.Join(magicHome, "backups")
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, parseErr := time.Parse(BackupStampFormat, e.Name()); parseErr == nil {
			stamps = append(stamps, e.Name())
			continue
		}
		others = append(others, e.Name())
	}
	sort.Strings(stamps)
	sort.Strings(others)
	return stamps, others, nil
}

func countFiles(root string) (int, error) {
	n := 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			n++
		}
		return nil
	})
	return n, err
}

// Rollback undoes one backup set. A set with a manifest is undone exactly: update
// entries are restored from their copy and create entries are deleted, but only
// while the target still holds what apply wrote. A set without one is an old
// backup, undone the old way (restore the copied files and report that created
// targets cannot be identified).
//
// The result names the set it replayed, so a run that skipped every entry still
// says which backup was used.
func Rollback(adapters []adapter.Adapter, opts RollbackOptions) (RollbackResult, error) {
	sets, err := ListBackups(opts.MagicHome)
	if err != nil {
		return RollbackResult{}, err
	}
	if len(sets) == 0 {
		providers, _ := ListProviderBackupDirs(opts.MagicHome)
		if len(providers) > 0 {
			return RollbackResult{}, fmt.Errorf("没有 apply 备份：backups/ 下只有 provider 备份 %s（不参与回滚）", strings.Join(providers, ", "))
		}
		return RollbackResult{}, fmt.Errorf("没有备份：%s", filepath.Join(opts.MagicHome, "backups"))
	}
	stamp := opts.Stamp
	if stamp == "" {
		stamp = sets[len(sets)-1].Stamp
	} else if !hasStamp(sets, stamp) {
		return RollbackResult{}, fmt.Errorf("backup set %q not found (see magic rollback --list)", stamp)
	}
	root := filepath.Join(opts.MagicHome, "backups", stamp)
	m, err := backup.Read(root)
	if err != nil {
		return RollbackResult{}, err
	}
	if m == nil {
		res, err := rollbackByCopy(adapters, stamp, root, opts)
		res.Skipped = append([]string{fmt.Sprintf("备份 %s 没有 manifest.json（旧备份，只恢复文件不删除新建）", stamp)}, res.Skipped...)
		return res, err
	}
	return rollbackManifest(stamp, root, m, opts)
}

// rollbackManifest replays a manifest: restore what apply overwrote, delete what
// apply created.
//
// It never discards work the user did after apply: an entry is only applied when
// the target still holds what apply wrote (or no longer exists), unless opts.Force
// says otherwise. Every entry left alone is reported with its reason, in the
// bucket --force can or cannot change.
func rollbackManifest(stamp, root string, m *backup.Manifest, opts RollbackOptions) (RollbackResult, error) {
	res := RollbackResult{Stamp: stamp}
	skip := func(e backup.Entry, reason string) {
		res.Skipped = append(res.Skipped, fmt.Sprintf("%s/%s: %s", e.Adapter, e.Rel, reason))
	}
	refuse := func(e backup.Entry, reason string) {
		res.Refused = append(res.Refused, fmt.Sprintf("%s/%s: %s", e.Adapter, e.Rel, reason))
	}
	for _, e := range m.Entries {
		if opts.Adapter != "" && opts.Adapter != e.Adapter {
			continue
		}
		if reason := pathProblem(root, opts.UserHome, e); reason != "" {
			refuse(e, reason)
			continue
		}
		switch e.Action {
		case backup.ActionCreate:
			if !opts.Force {
				if reason, err := createSkipReason(e); err != nil {
					return res, err
				} else if reason != "" {
					skip(e, reason)
					continue
				}
			}
			if !opts.DryRun {
				if err := os.Remove(e.Target); err != nil && !os.IsNotExist(err) {
					return res, err
				}
			}
			res.Restored = append(res.Restored, Restore{
				Stamp:   stamp,
				Adapter: adapter.ID(e.Adapter),
				Rel:     e.Rel,
				Path:    e.Target,
				Delete:  true,
			})
		case backup.ActionUpdate:
			if e.Backup == "" {
				refuse(e, "manifest 没有备份路径，无法恢复（--force 也不执行）")
				continue
			}
			if !opts.Force {
				if reason, err := updateSkipReason(e); err != nil {
					return res, err
				} else if reason != "" {
					skip(e, reason)
					continue
				}
			}
			src := filepath.Join(root, e.Backup)
			data, err := os.ReadFile(src)
			if err != nil {
				return res, err
			}
			mode := backupMode(e.Mode, src)
			if !opts.DryRun {
				if err := writefile.Atomic(e.Target, data, mode); err != nil {
					return res, err
				}
			}
			res.Restored = append(res.Restored, Restore{
				Stamp:   stamp,
				Adapter: adapter.ID(e.Adapter),
				Rel:     e.Rel,
				Path:    e.Target,
				Bytes:   len(data),
			})
		default:
			refuse(e, fmt.Sprintf("manifest 里的 action %q 不认识，无法执行（--force 也不执行）", e.Action))
		}
	}
	sort.Slice(res.Restored, func(i, j int) bool {
		if res.Restored[i].Adapter != res.Restored[j].Adapter {
			return res.Restored[i].Adapter < res.Restored[j].Adapter
		}
		return res.Restored[i].Rel < res.Restored[j].Rel
	})
	return res, nil
}

// pathProblem reports why the entry's target or copy is out of bounds, so
// rollback refuses to write or delete it. The check applies to --force too: it
// guards the filesystem, not the content.
func pathProblem(root, userHome string, e backup.Entry) string {
	if !filepath.IsAbs(e.Target) || !withinDir(userHome, e.Target) {
		return fmt.Sprintf("目标 %s 不在 user home %s 之下，拒绝写入", e.Target, userHome)
	}
	if e.Backup != "" && !withinDir(root, filepath.Join(root, e.Backup)) {
		return fmt.Sprintf("备份路径 %s 逃出备份集合，拒绝读取", e.Backup)
	}
	return ""
}

// withinDir reports whether path is root itself or below it.
func withinDir(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil || filepath.IsAbs(rel) {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// createSkipReason reports why a create entry must not delete its target: the
// file no longer holds what apply wrote, so it is the user's now.
func createSkipReason(e backup.Entry) (string, error) {
	current, exists, err := targetFingerprint(e.Target)
	if err != nil || !exists {
		return "", err
	}
	if e.After == "" {
		return "manifest 没有内容指纹（旧备份），无法判断目标是否被改动，跳过（加 --force 删除）", nil
	}
	if current != e.After {
		return "目标在 apply 之后被改动，跳过（加 --force 删除）", nil
	}
	return "", nil
}

// updateSkipReason reports why an update entry must not restore its copy: the
// target holds neither what apply wrote (after) nor what apply replaced (before),
// so restoring would throw away an edit made after apply.
func updateSkipReason(e backup.Entry) (string, error) {
	current, exists, err := targetFingerprint(e.Target)
	if err != nil || !exists {
		return "", err
	}
	if e.After == "" {
		return "manifest 没有内容指纹（旧备份），无法判断目标是否被改动，跳过（加 --force 覆盖）", nil
	}
	if current == e.After || (e.Before != "" && current == e.Before) {
		return "", nil
	}
	return "目标在 apply 之后被改动，跳过（加 --force 覆盖）", nil
}

// targetFingerprint returns the current content digest of path and whether the
// file exists.
func targetFingerprint(path string) (backup.Fingerprint, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	return backup.FingerprintOf(data), true, nil
}

// backupMode prefers the mode apply recorded, falling back to the copy's own
// permission, then 0644.
func backupMode(recorded, copyPath string) os.FileMode {
	if mode, ok := backup.ParseMode(recorded); ok {
		return mode
	}
	if info, err := os.Stat(copyPath); err == nil {
		return info.Mode().Perm()
	}
	return 0o644
}

// rollbackByCopy is the pre-manifest behaviour: put every copied file back under
// the adapter's mirror root. It cannot know which targets apply created.
func rollbackByCopy(adapters []adapter.Adapter, stamp, root string, opts RollbackOptions) (RollbackResult, error) {
	byID := map[string]adapter.Adapter{}
	for _, a := range adapters {
		byID[string(a.ID())] = a
	}

	res := RollbackResult{Stamp: stamp}
	adapterDirs, err := os.ReadDir(root)
	if err != nil {
		return res, err
	}
	for _, ad := range adapterDirs {
		if !ad.IsDir() {
			continue
		}
		id := ad.Name()
		if opts.Adapter != "" && opts.Adapter != id {
			continue
		}
		a, ok := byID[id]
		if !ok {
			res.Skipped = append(res.Skipped, fmt.Sprintf("%s: 没有这个 adapter，跳过整个目录", id))
			continue
		}
		mirrorRoot := a.MirrorRoot(opts.UserHome)
		err := filepath.WalkDir(filepath.Join(root, id), func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, err := filepath.Rel(filepath.Join(root, id), path)
			if err != nil {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			dst := restoreTarget(opts.UserHome, mirrorRoot, rel)
			if dst == "" {
				res.Skipped = append(res.Skipped, fmt.Sprintf("%s/%s: 该 adapter 没有镜像目录，跳过", id, rel))
				return nil
			}
			mode := os.FileMode(0o644)
			if info, statErr := os.Stat(path); statErr == nil {
				mode = info.Mode().Perm()
			}
			if !opts.DryRun {
				if err := writefile.Atomic(dst, data, mode); err != nil {
					return err
				}
			}
			res.Restored = append(res.Restored, Restore{
				Stamp:   stamp,
				Adapter: a.ID(),
				Rel:     rel,
				Path:    dst,
				Bytes:   len(data),
			})
			return nil
		})
		if err != nil {
			return res, err
		}
	}
	sort.Slice(res.Restored, func(i, j int) bool {
		if res.Restored[i].Adapter != res.Restored[j].Adapter {
			return res.Restored[i].Adapter < res.Restored[j].Adapter
		}
		return res.Restored[i].Rel < res.Restored[j].Rel
	})
	return res, nil
}

// restoreTarget maps a backup-relative path to the file it came from.
//
// The merged ~/.claude.json is backed up as claude/_claude.json even though it
// lives outside that adapter's mirror root, so it is matched by file name. The
// MCP projection used to back it up as claude/.claude.json, which would land at
// ~/.claude/.claude.json if it were treated as a mirror-relative path; both
// names map to ~/.claude.json. ApplyAdapter never mirrors a file with either
// name, so no mirror can collide.
func restoreTarget(userHome, mirrorRoot, rel string) string {
	switch filepath.Base(rel) {
	case ClaudeJSONBackupRel, ".claude.json":
		return filepath.Join(userHome, ".claude.json")
	}
	if mirrorRoot == "" {
		return ""
	}
	return filepath.Join(mirrorRoot, rel)
}

func hasStamp(sets []BackupSet, stamp string) bool {
	for _, s := range sets {
		if s.Stamp == stamp {
			return true
		}
	}
	return false
}
