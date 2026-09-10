package core

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"magic/internal/adapter"
	"magic/internal/policy"
	"magic/internal/secrets"
)

// Triage walks adapter roots and classifies files.
// When allow-listed settings.json / settings.local.json (or config.toml) is found,
// local file refs under the harness root are appended as allow
// (reason settings.ref / config.ref) if they exist.
func Triage(userHome string, a adapter.Adapter, overlay *policy.OverlayHarness) *policy.Report {
	report := &policy.Report{}
	pol := a.CapturePolicy()
	for _, root := range pol.Roots {
		abs := filepath.Join(userHome, filepath.FromSlash(root.RelHome))
		walkRoot(report, string(a.ID()), abs, pol, overlay)
	}
	if overlay != nil {
		for _, extra := range overlay.ExtraRoots {
			eroot := filepath.Join(userHome, filepath.FromSlash(extra))
			walkExtra(report, string(a.ID()), extra, eroot, overlay)
		}
	}
	followLocalConfigRefs(report, userHome, a, overlay)
	return report
}

func walkRoot(report *policy.Report, harness, root string, builtin adapter.CapturePolicy, overlay *policy.OverlayHarness) {
	st, err := os.Stat(root)
	if err != nil || !st.IsDir() {
		report.MissingRoots = append(report.MissingRoots, root)
		return
	}
	report.ScannedRoots = append(report.ScannedRoots, root)
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		decision, reason := policy.Classify(rel, builtin, overlay)
		v := policy.Verdict{Harness: harness, AbsPath: path, Rel: rel, Decision: decision, Reason: reason, Root: root}
		switch decision {
		case "allow":
			report.Allow = append(report.Allow, v)
		case "deny":
			report.Deny = append(report.Deny, v)
		default:
			report.Unknown = append(report.Unknown, v)
		}
		return nil
	})
}

func walkExtra(report *policy.Report, harness, extraRel, eroot string, overlay *policy.OverlayHarness) {
	st, err := os.Stat(eroot)
	if err != nil || !st.IsDir() {
		report.MissingRoots = append(report.MissingRoots, eroot)
		return
	}
	report.ScannedRoots = append(report.ScannedRoots, eroot)
	_ = filepath.WalkDir(eroot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(eroot, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		var decision, reason string
		if policy.MatchAny(rel, overlay.Exclude) {
			decision, reason = "deny", "user.exclude (extra_root)"
		} else if len(overlay.Include) > 0 && policy.MatchAny(rel, overlay.Include) {
			decision, reason = "allow", "user.include (extra_root)"
		} else if len(overlay.Include) == 0 {
			decision, reason = "allow", "user.extra_root (all)"
		} else {
			decision, reason = "unknown", "extra_root not in user.include"
		}
		v := policy.Verdict{
			Harness: harness + "+extra", AbsPath: path,
			Rel:      filepath.ToSlash(filepath.Join(extraRel, rel)),
			Decision: decision, Reason: reason, Root: eroot,
		}
		switch decision {
		case "allow":
			report.Allow = append(report.Allow, v)
		case "deny":
			report.Deny = append(report.Deny, v)
		default:
			report.Unknown = append(report.Unknown, v)
		}
		return nil
	})
}

// CaptureOptions controls capture.
type CaptureOptions struct {
	MagicHome  string
	UserHome   string
	DryRun     bool
	Only       string
	Passphrase string
}

// CaptureAdapter copies allow-listed files into packs/harness/<id>.
func CaptureAdapter(a adapter.Adapter, overlay *policy.Overlay, opts CaptureOptions) (*policy.Report, []string, error) {
	ov := overlay.EffectiveOverlay(string(a.ID()))
	report := Triage(opts.UserHome, a, ov)
	var copied []string
	if opts.DryRun {
		return report, copied, nil
	}
	destRoot := filepath.Join(opts.MagicHome, a.LibraryMirrorRel())
	pass := opts.Passphrase
	if pass == "" {
		pass = os.Getenv("MAGIC_PASSPHRASE")
	}
	allSecrets := map[string]string{}
	warned := false
	for _, v := range report.Allow {
		data, err := os.ReadFile(v.AbsPath)
		if err != nil {
			return report, copied, err
		}
		rr := secrets.RedactBytes(v.AbsPath, data)
		out := rr.Data
		// Fold absolute $HOME/<harnessRel> paths in settings so library copies are portable.
		if isSettingsConfigRel(v.Rel) {
			if relHome := PrimaryRelHome(a); relHome != "" {
				out = FoldHomePathsInSettings(out, opts.UserHome, relHome)
			} else if v.Root != "" {
				if rh, err := filepath.Rel(opts.UserHome, v.Root); err == nil {
					out = FoldHomePathsInSettings(out, opts.UserHome, rh)
				}
			}
		}
		for k, s := range rr.Secrets {
			allSecrets[k] = s
		}
		dst := filepath.Join(destRoot, filepath.FromSlash(v.Rel))
		if err := AtomicWriteFile(dst, out, 0o644); err != nil {
			return report, copied, err
		}
		copied = append(copied, dst)
		if rr.Redacted && pass == "" {
			warned = true
		}
	}
	if w, err := secrets.StoreSecrets(filepath.Join(opts.MagicHome, "secrets"), string(a.ID()), allSecrets, pass); err != nil {
		return report, copied, err
	} else if w || warned {
		fmt.Fprintf(os.Stderr, "warning: secrets redacted in mirror but MAGIC_PASSPHRASE unset — raw values not stored in secrets/\n")
	}
	// Claude: field-level extract of ~/.claude.json → claude.json.partial.json
	if a.ID() == "claude" {
		if err := captureClaudePartial(opts, destRoot, &copied); err != nil {
			return report, copied, err
		}
	}
	return report, copied, nil
}

// FormatReport prints triage dry-run style.
func FormatReport(report *policy.Report) string {
	var b strings.Builder
	b.WriteString("=== magic capture triage ===\n")
	fmt.Fprintf(&b, "roots_found: %d  roots_missing: %d\n", len(report.ScannedRoots), len(report.MissingRoots))
	fmt.Fprintf(&b, "counts: allow=%d deny=%d unknown=%d\n", len(report.Allow), len(report.Deny), len(report.Unknown))
	b.WriteString("\n-- ALLOW --\n")
	for _, v := range report.Allow {
		fmt.Fprintf(&b, "  + [%s] %s  (%s)\n", v.Harness, v.Rel, v.Reason)
	}
	b.WriteString("\n-- DENY --\n")
	for _, v := range report.Deny {
		fmt.Fprintf(&b, "  - [%s] %s  (%s)\n", v.Harness, v.Rel, v.Reason)
	}
	b.WriteString("\n-- UNKNOWN (default DENY; review) --\n")
	if len(report.Unknown) == 0 {
		b.WriteString("  (none)\n")
	}
	for _, v := range report.Unknown {
		fmt.Fprintf(&b, "  ? [%s] %s  (%s)\n", v.Harness, v.Rel, v.Reason)
	}
	return b.String()
}
