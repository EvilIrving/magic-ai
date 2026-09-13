package provider

import "fmt"

// StubWriter exposes the SwitchWriter surface for adapters without a dialect writer yet.
type StubWriter struct {
	ID string
}

func (s StubWriter) Harness() string                   { return s.ID }
func (s StubWriter) Implemented() bool                 { return false }
func (s StubWriter) SupportsOfficial() bool            { return false }
func (s StubWriter) BackupPrefix() string              { return "config" }
func (s StubWriter) BackupExt() string                 { return ".bak" }
func (s StubWriter) ConfigPath(userHome string) string { return "" }
func (s StubWriter) Status(ctx Context) string {
	return fmt.Sprintf("===== %s provider status =====\nnot implemented yet — add a SwitchWriter\n", s.ID)
}
func (s StubWriter) List(ctx Context) string {
	return fmt.Sprintf("official | <providerName>  (harness %s: not implemented yet)\n", s.ID)
}
func (s StubWriter) Apply(ctx Context, req ApplyRequest) (string, error) {
	return "", ErrNotImplemented(s.ID)
}
