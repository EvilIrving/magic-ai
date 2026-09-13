package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"magic/internal/adapter"
)

// AddPathOptions copies a harness-local file into packs/harness/<id>/<rel>.
type AddPathOptions struct {
	MagicHome string
	UserHome  string
	Source    string
	As        string // optional "harness/relative/path"
	Adapters  []adapter.Adapter
}

// AddPathResult is the library destination after a successful add.
type AddPathResult struct {
	Harness string
	Rel     string
	Dest    string
}

// AddPath copies src into the config library. The original file is not moved.
func AddPath(opts AddPathOptions) (AddPathResult, error) {
	src, err := filepath.Abs(opts.Source)
	if err != nil {
		return AddPathResult{}, err
	}
	st, err := os.Stat(src)
	if err != nil {
		return AddPathResult{}, fmt.Errorf("add path: %w", err)
	}
	if st.IsDir() {
		return AddPathResult{}, fmt.Errorf("add path: %s is a directory (want a file)", src)
	}

	harness, rel, err := resolveAddTarget(src, opts)
	if err != nil {
		return AddPathResult{}, err
	}
	a, ok := findAdapter(opts.Adapters, harness)
	if !ok {
		return AddPathResult{}, fmt.Errorf("add path: unknown harness %q (try --as <harness>/<rel>)", harness)
	}
	if rel == "" || rel == "." || strings.HasPrefix(rel, "..") {
		return AddPathResult{}, fmt.Errorf("add path: invalid library relative path %q", rel)
	}

	dest := filepath.Join(opts.MagicHome, a.LibraryMirrorRel(), filepath.FromSlash(rel))
	if sameFile(src, dest) {
		return AddPathResult{Harness: harness, Rel: filepath.ToSlash(rel), Dest: dest}, nil
	}
	if err := CopyFile(src, dest); err != nil {
		return AddPathResult{}, err
	}
	return AddPathResult{Harness: harness, Rel: filepath.ToSlash(rel), Dest: dest}, nil
}

func resolveAddTarget(src string, opts AddPathOptions) (harness, rel string, err error) {
	if as := strings.TrimSpace(opts.As); as != "" {
		as = filepath.ToSlash(as)
		parts := strings.SplitN(as, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("add path: --as must be <harness>/<rel> (got %q)", opts.As)
		}
		return parts[0], parts[1], nil
	}
	if opts.UserHome == "" {
		return "", "", fmt.Errorf("add path: cannot detect harness (pass --as <harness>/<rel>)")
	}
	bestLen := -1
	for _, a := range opts.Adapters {
		root := a.MirrorRoot(opts.UserHome)
		if root == "" {
			continue
		}
		relPath, ok := relIfSubpath(root, src)
		if !ok {
			continue
		}
		if len(root) > bestLen {
			bestLen = len(root)
			harness = string(a.ID())
			rel = relPath
		}
	}
	if harness == "" {
		return "", "", fmt.Errorf("add path: %s is not under a known harness root (pass --as <harness>/<rel>)", src)
	}
	return harness, rel, nil
}

func findAdapter(adapters []adapter.Adapter, id string) (adapter.Adapter, bool) {
	for _, a := range adapters {
		if string(a.ID()) == id {
			return a, true
		}
	}
	return nil, false
}

func relIfSubpath(root, file string) (string, bool) {
	root = filepath.Clean(root)
	file = filepath.Clean(file)
	rel, err := filepath.Rel(root, file)
	if err != nil {
		return "", false
	}
	if rel == "." || strings.HasPrefix(rel, "..") {
		return "", false
	}
	return rel, true
}

func sameFile(a, b string) bool {
	aa, err1 := filepath.Abs(a)
	bb, err2 := filepath.Abs(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return aa == bb
}
