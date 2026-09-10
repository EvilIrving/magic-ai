package skills

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// ListEntries returns skill entry names under central (dirs or files).
func ListEntries(central string) ([]string, error) {
	ents, err := os.ReadDir(central)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range ents {
		names = append(names, e.Name())
	}
	return names, nil
}

// ListSkillDirs returns first-level directory (or symlink-to-dir) names under source.
func ListSkillDirs(source string) ([]string, error) {
	ents, err := os.ReadDir(source)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range ents {
		p := filepath.Join(source, e.Name())
		fi, err := os.Stat(p) // follow link
		if err != nil || !fi.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	return names, nil
}

// EnsureLink creates or repairs a link at linkPath pointing to target.
// Strategy: symlink → (Windows) junction → copy.
// Never deletes a real directory/file.
func EnsureLink(linkPath, target string, dryRun bool) (action string, err error) {
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	repair := false
	fi, err := os.Lstat(linkPath)
	if err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			cur, readErr := os.Readlink(linkPath)
			if readErr == nil {
				curAbs := cur
				if !filepath.IsAbs(cur) {
					curAbs = filepath.Join(filepath.Dir(linkPath), cur)
				}
				curAbs, _ = filepath.Abs(curAbs)
				if curAbs == absTarget {
					return "ok", nil
				}
			}
			repair = true
			if dryRun {
				return "repair-link", nil
			}
			if err := os.Remove(linkPath); err != nil {
				return "", err
			}
		} else if isJunction(linkPath) {
			repair = true
			if dryRun {
				return "repair-link", nil
			}
			if err := os.Remove(linkPath); err != nil {
				return "", err
			}
		} else {
			return "skip-real", fmt.Errorf("refusing to replace real path %s", linkPath)
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}

	if dryRun {
		if repair {
			return "repair-link", nil
		}
		return "create-link", nil
	}
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		return "", err
	}

	mode, err := linkOrCopy(absTarget, linkPath)
	if err != nil {
		return "", err
	}
	if repair {
		return "repair-" + mode, nil
	}
	return "create-" + mode, nil
}

// linkOrCopy tries symlink, then Windows junction, then recursive copy.
func linkOrCopy(absTarget, linkPath string) (mode string, err error) {
	if err := os.Symlink(absTarget, linkPath); err == nil {
		return "symlink", nil
	} else {
		symlinkErr := err
		if runtime.GOOS == "windows" {
			if jerr := makeJunction(absTarget, linkPath); jerr == nil {
				return "junction", nil
			}
		}
		if cerr := copyPath(absTarget, linkPath); cerr != nil {
			return "", fmt.Errorf("symlink failed (%v); copy failed (%w)", symlinkErr, cerr)
		}
		return "copy", nil
	}
}

func makeJunction(target, link string) error {
	cmd := exec.Command("cmd", "/C", "mklink", "/J", link, target)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mklink /J: %w (%s)", err, string(out))
	}
	return nil
}

func isJunction(path string) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	fi, err := os.Lstat(path)
	if err != nil || !fi.IsDir() {
		return false
	}
	return false
}

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
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
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	ents, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range ents {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if err := copyPath(s, d); err != nil {
			return err
		}
	}
	return nil
}

// ApplyProjects links each central skill into each harness skills dir.
// Used by magic apply for global packs/skills projection.
func ApplyProjects(central string, harnessSkillsDirs []string, dryRun bool) ([]string, error) {
	entries, err := ListEntries(central)
	if err != nil {
		return nil, err
	}
	var log []string
	for _, dir := range harnessSkillsDirs {
		if dir == "" {
			continue
		}
		centralAbs, _ := filepath.Abs(central)
		dirAbs, _ := filepath.Abs(dir)
		if centralAbs != "" && dirAbs == centralAbs {
			continue
		}
		if !dryRun {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return log, err
			}
		}
		for _, name := range entries {
			target := filepath.Join(central, name)
			link := filepath.Join(dir, name)
			action, err := EnsureLink(link, target, dryRun)
			msg := fmt.Sprintf("%s -> %s [%s]", link, target, action)
			if err != nil {
				msg += " error=" + err.Error()
			}
			log = append(log, msg)
		}
	}
	return log, nil
}
