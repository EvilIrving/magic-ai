package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"magic/internal/secrets"
)

// configKeys are field-level keys extracted from ~/.claude.json on capture.
// Huge runtime / project state is intentionally omitted.
var claudeConfigKeys = []string{
	"mcpServers",
	"theme",
	"model",
	"hasTrustDialogAccepted",
	"permissions",
	"enabledPlugins",
	"userID", // identity-ish but small; optional
	"autoUpdates",
	"preferredNotifChannel",
}

// ExtractClaudePartial keeps only clearly-config keys from ~/.claude.json.
func ExtractClaudePartial(data []byte) ([]byte, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	out := map[string]any{}
	for _, k := range claudeConfigKeys {
		if v, ok := raw[k]; ok {
			out[k] = v
		}
	}
	// Always include mcpServers key presence documentation
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func captureClaudePartial(opts CaptureOptions, destRoot string, copied *[]string, allSecrets map[string]string) error {
	p := filepath.Join(opts.UserHome, ".claude.json")
	st, err := os.Stat(p)
	if err != nil {
		return nil
	}
	if st.Size() > 8*1024*1024 {
		fmt.Fprintf(os.Stderr, "note: ~/.claude.json is huge (%d bytes); field extract still attempted for config keys only\n", st.Size())
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	partial, err := ExtractClaudePartial(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "note: ~/.claude.json not valid JSON (%v); skipped partial extract\n", err)
		return nil
	}
	rr := secrets.RedactBytes("claude.json.partial.json", partial)
	out := rr.Data
	if !rr.Redacted {
		out = partial
	}
	for k, s := range rr.Secrets {
		allSecrets[k] = s
	}
	dst := filepath.Join(destRoot, "claude.json.partial.json")
	if err := AtomicWriteFile(dst, out, 0o600); err != nil {
		return err
	}
	*copied = append(*copied, dst)
	return nil
}

func applyClaudePartial(opts ApplyOptions, backupDir string) ([]string, error) {
	var log []string
	src := filepath.Join(opts.MagicHome, "packs", "harness", "claude", "claude.json.partial.json")
	data, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			return log, nil
		}
		return log, err
	}
	pass := applyPassphrase(opts)
	if secMap, secErr := secrets.LoadSecrets(filepath.Join(opts.MagicHome, "secrets"), "claude", pass); secErr == nil && len(secMap) > 0 {
		data = secrets.ReinjectBytes(data, secMap)
	}
	var partial map[string]any
	if err := json.Unmarshal(data, &partial); err != nil {
		return log, err
	}
	dst := filepath.Join(opts.UserHome, ".claude.json")
	existing := map[string]any{}
	if raw, err := os.ReadFile(dst); err == nil {
		_ = json.Unmarshal(raw, &existing)
	}
	for k, v := range partial {
		existing[k] = v
	}
	out, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return log, err
	}
	out = append(out, '\n')
	if opts.DryRun {
		log = append(log, "[claude] would merge claude.json.partial.json → ~/.claude.json")
		return log, nil
	}
	if _, err := os.Stat(dst); err == nil && backupDir != "" {
		_ = BackupPath(dst, backupDir, "claude/_claude.json")
	}
	if err := AtomicWriteFile(dst, out, 0o600); err != nil {
		return log, err
	}
	log = append(log, "[claude] merged claude.json.partial.json → ~/.claude.json")
	return log, nil
}
