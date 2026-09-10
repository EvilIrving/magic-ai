package policy

import (
	"path/filepath"
	"strings"
)

// MatchAny reports whether rel matches any glob pattern (Python fnmatch + /** prefix).
func MatchAny(rel string, patterns []string) bool {
	rel = filepath.ToSlash(rel)
	for _, pat := range patterns {
		pat = filepath.ToSlash(pat)
		if fnmatch(rel, pat) {
			return true
		}
		if strings.HasSuffix(pat, "/**") {
			prefix := strings.TrimSuffix(pat, "/**")
			if rel == prefix || strings.HasPrefix(rel, prefix+"/") {
				return true
			}
		}
	}
	return false
}

// fnmatch approximates Python fnmatch.fnmatch with / as separator.
// * and ? do not cross '/'.
func fnmatch(name, pattern string) bool {
	return fnmatchParts(strings.Split(name, "/"), strings.Split(pattern, "/"))
}

func fnmatchParts(name, pat []string) bool {
	for len(pat) > 0 {
		if pat[0] == "**" {
			pat = pat[1:]
			if len(pat) == 0 {
				return true
			}
			for i := 0; i <= len(name); i++ {
				if fnmatchParts(name[i:], pat) {
					return true
				}
			}
			return false
		}
		if len(name) == 0 {
			return false
		}
		ok, err := filepath.Match(pat[0], name[0])
		if err != nil || !ok {
			return false
		}
		pat = pat[1:]
		name = name[1:]
	}
	return len(name) == 0
}
