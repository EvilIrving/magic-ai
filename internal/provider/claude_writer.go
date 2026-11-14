package provider

// ClaudeWriter projects providers.yaml → ~/.claude/settings.json (or clears for official).
type ClaudeWriter struct{}

func (ClaudeWriter) Harness() string        { return HarnessClaude }
func (ClaudeWriter) Implemented() bool      { return true }
func (ClaudeWriter) SupportsOfficial() bool { return true }
func (ClaudeWriter) BackupPrefix() string   { return "settings" }
func (ClaudeWriter) BackupExt() string      { return ".json" }
func (ClaudeWriter) ConfigPath(userHome string) string {
	return ClaudeSettingsPath(userHome)
}

func (w ClaudeWriter) Files(ctx Context, req ApplyRequest) ([]FileWrite, error) {
	return filesClaude(ctx, req)
}

func (w ClaudeWriter) Status(ctx Context) string { return claudeStatusText(ctx) }
func (w ClaudeWriter) List(ctx Context) string   { return claudeListText(ctx) }
