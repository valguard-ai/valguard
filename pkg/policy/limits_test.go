package policy

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestParse_rejectsOversizePolicy(t *testing.T) {
	raw := []byte("version: 1\nname: x\nrules:\n  - id: valid_format\n")
	oversize := append(raw, make([]byte, MaxPolicyBytes)...)
	_, err := Parse(oversize)
	if err == nil || !strings.Contains(err.Error(), "policy exceeds") {
		t.Fatalf("Parse oversize: err=%v", err)
	}
}

func TestRun_rejectsOversizeOutput(t *testing.T) {
	doc := Document{
		Version: 1,
		Name:    "x",
		Rules:   []Rule{{ID: "valid_format", Params: map[string]any{"format": "json"}, OnFail: "block"}},
	}
	out := strings.Repeat("x", MaxOutputBytes+1)
	_, err := Run(doc, out, "")
	if err == nil || !strings.Contains(err.Error(), "output exceeds") {
		t.Fatalf("Run oversize output: err=%v", err)
	}
}

func TestRun_validationTimeout(t *testing.T) {
	doc := Document{
		Version: 1,
		Name:    "x",
		Rules:   []Rule{{ID: "valid_format", Params: map[string]any{"format": "json"}, OnFail: "block"}},
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	_, err := runWithContext(ctx, doc, `{"ok":true}`, "", time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "validation timed out") {
		t.Fatalf("timeout: err=%v", err)
	}
}
