package main

import (
	"encoding/json"
	"strings"
	"testing"

	"magic/internal/provider"
)

// adaptersJSON is the --json shape of `magic adapters`, decoded for the fields
// this test reads.
type adaptersJSON struct {
	ID          string `json:"id"`
	Implemented bool   `json:"implemented"`
	Reason      string `json:"reason"`
}

func runAdaptersJSON(t *testing.T) []adaptersJSON {
	t.Helper()
	out, err := runMagic(t, t.TempDir(), cmdAdapters(), "--json")
	if err != nil {
		t.Fatalf("adapters --json: %v\n%s", err, out)
	}
	var rows []adaptersJSON
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("parse adapters JSON: %v\n%s", err, out)
	}
	return rows
}

// Every stub SwitchWriter reports the reason stubdialects.go records, and a
// harness with a real writer reports none.
func TestAdaptersJSONCarriesStubReason(t *testing.T) {
	rows := runAdaptersJSON(t)
	byID := map[string]adaptersJSON{}
	for _, r := range rows {
		byID[r.ID] = r
	}
	for _, id := range []string{"cline", "zcode", "gemini", "copilot", "trae"} {
		want, ok := provider.StubReason(id)
		if !ok {
			t.Fatalf("stubdialects.go has no reason for %s", id)
		}
		if got := byID[id].Reason; got != want {
			t.Errorf("%s reason = %q, want %q", id, got, want)
		}
	}
	if got := byID["claude"].Reason; got != "" {
		t.Errorf("claude has a real writer but reports reason %q", got)
	}
}

// The terminal output carries the same one-line reason for a stub adapter.
func TestAdaptersHumanShowsStubReason(t *testing.T) {
	out, err := runMagic(t, t.TempDir(), cmdAdapters())
	if err != nil {
		t.Fatalf("adapters: %v\n%s", err, out)
	}
	for _, id := range []string{"cline", "zcode"} {
		reason, ok := provider.StubReason(id)
		if !ok {
			t.Fatalf("stubdialects.go has no reason for %s", id)
		}
		if !strings.Contains(out, reason) {
			t.Fatalf("adapters output lacks the %s reason %q:\n%s", id, reason, out)
		}
	}
}
