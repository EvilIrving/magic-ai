package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"magic/internal/backup"
	"magic/internal/writefile"
)

// ApplyFiles writes one projection: Files computes it, this function records it
// in the backup set, takes the harness's stamped backup and writes only the
// files whose content or permission changed. A target that already holds what
// the projection computes keeps its inode, mtime and mode.
func ApplyFiles(ctx Context, w SwitchWriter, req ApplyRequest) (string, error) {
	writes, err := w.Files(ctx, req)
	if err != nil {
		return "", err
	}
	// Back up before writing, in Files order, so an update entry holds the
	// content the projection replaces.
	for _, fw := range writes {
		if err := recordProjectionFile(ctx, w, fw); err != nil {
			return "", err
		}
	}
	bak, err := CreateBackup(ctx.MagicHome, w.Harness(), w.ConfigPath(ctx.UserHome), w.BackupPrefix(), w.BackupExt())
	if err != nil {
		return "", err
	}

	written, removed, kept := 0, 0, 0
	for _, fw := range writes {
		if fw.Remove {
			if err := os.Remove(fw.Path); err != nil && !os.IsNotExist(err) {
				return "", err
			}
			removed++
			continue
		}
		if !writefile.NeedsWrite(fw.Path, fw.Data, fw.Mode) {
			kept++
			continue
		}
		if err := writefile.Write(fw.Path, fw.Data, fw.Mode); err != nil {
			return "", err
		}
		written++
	}

	var b strings.Builder
	if bak != "" {
		fmt.Fprintf(&b, "Backup: %s\n", bak)
	}
	if req.Official {
		if written == 0 && removed == 0 {
			// A clear that found nothing must say so: "cleared" would claim a
			// change the user cannot see in the file.
			fmt.Fprintf(&b, "no Magic-managed entries to clear in %s\n", w.ConfigPath(ctx.UserHome))
		} else {
			fmt.Fprintf(&b, "cleared Magic-managed entries in %s\n", w.ConfigPath(ctx.UserHome))
		}
	} else {
		fmt.Fprintf(&b, "provider %s (endpoint=%s protocol=%s) → %s\n",
			req.Name, req.Endpoint.ID, req.Endpoint.Protocol, w.ConfigPath(ctx.UserHome))
	}
	fmt.Fprintf(&b, "Files: %d written, %d removed, %d unchanged\n", written, removed, kept)
	return b.String(), nil
}

// recordProjectionFile adds one target to the backup set's manifest. An existing
// target is copied into the set first (update, so rollback restores what the
// projection replaced — or removed); a missing one is recorded as create, so
// rollback deletes it. Removing a file the target no longer has records nothing.
func recordProjectionFile(ctx Context, w SwitchWriter, fw FileWrite) error {
	if ctx.BackupDir == "" {
		return nil
	}
	rel := projectionFileRel(ctx.UserHome, fw.Path)
	entry := backup.Entry{
		Adapter: w.Harness(),
		Rel:     rel,
		Target:  fw.Path,
		Action:  backup.ActionCreate,
		Mode:    backup.ModeString(fw.Mode),
	}
	if !fw.Remove {
		// A removal leaves nothing to compare against afterwards.
		entry.After = backup.FingerprintOf(fw.Data)
	}
	switch {
	case fileExists(fw.Path):
		entry.Action, entry.Backup = backup.ActionUpdate, filepath.Join(w.Harness(), rel)
	case fw.Remove:
		return nil
	}
	return backup.Record(ctx.BackupDir, entry)
}

// projectionFileRel is where a projection target is recorded inside a backup
// set: its path relative to the user home, or its base name when it lives
// outside it.
func projectionFileRel(userHome, path string) string {
	if userHome != "" {
		if rel, err := filepath.Rel(userHome, path); err == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}
	return filepath.Base(path)
}

// readTarget returns the content a projection merges into: ctx.ReadFile when
// plan stacks layers onto the bytes it has produced so far, the disk otherwise.
// A missing target is empty content, not an error.
func readTarget(ctx Context, path string) ([]byte, error) {
	read := ctx.ReadFile
	if read == nil {
		read = os.ReadFile
	}
	data, err := read(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return data, nil
}
