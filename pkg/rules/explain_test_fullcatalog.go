//go:build !osspublic

package rules

import "testing"

func TestExplainRule_nonMVPRegistered(t *testing.T) {
	ex, err := ExplainRule("latency_validator")
	if err != nil {
		t.Fatal(err)
	}
	if ex.OSS != "" {
		t.Fatalf("oss=%q want empty for non-MVP", ex.OSS)
	}
}
