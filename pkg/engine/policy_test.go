package engine

import "testing"

func TestComputeAgentPolicyFlags(t *testing.T) {
	tests := []struct {
		name string
		inst []Instance
		want AgentPolicyFlags
	}{
		{"empty", nil, AgentPolicyFlags{}},
		{"warn only", []Instance{{Enabled: true, OnFail: "warn"}}, AgentPolicyFlags{}},
		{"block", []Instance{{Enabled: true, OnFail: "block"}}, AgentPolicyFlags{HasEnforce: true}},
		{"reask", []Instance{{Enabled: true, OnFail: "reask"}}, AgentPolicyFlags{HasReask: true, HasEnforce: true}},
		{"disabled block", []Instance{{Enabled: false, OnFail: "block"}}, AgentPolicyFlags{}},
		{"mixed warn and reask", []Instance{
			{Enabled: true, OnFail: "warn"},
			{Enabled: true, OnFail: "reask"},
		}, AgentPolicyFlags{HasReask: true, HasEnforce: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeAgentPolicyFlags(tt.inst)
			if got != tt.want {
				t.Fatalf("ComputeAgentPolicyFlags() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestAgentPolicyFlags_derivedDecisions(t *testing.T) {
	block := AgentPolicyFlags{HasEnforce: true}
	reask := AgentPolicyFlags{HasReask: true, HasEnforce: true}

	if block.CanPassthroughStream(false) {
		t.Fatal("block must use buffered path")
	}
	if reask.CanPassthroughStream(false) {
		t.Fatal("reask must use buffered path")
	}
	if !reask.CanPassthroughStream(true) {
		t.Fatal("shadow overrides reask passthrough gate")
	}
	if block.MaxAttempts(false, 2) != 1 {
		t.Fatalf("block maxAttempts = %d, want 1", block.MaxAttempts(false, 2))
	}
	if reask.MaxAttempts(false, 2) != 3 {
		t.Fatalf("reask maxAttempts = %d, want 3", reask.MaxAttempts(false, 2))
	}
	if reask.MaxAttempts(true, 2) != 1 {
		t.Fatalf("shadow maxAttempts = %d, want 1", reask.MaxAttempts(true, 2))
	}
	if !block.ValidationRunOptions(false).StopOnFirstEnforce {
		t.Fatal("block should fail fast")
	}
	if reask.ValidationRunOptions(false).StopOnFirstEnforce != true {
		t.Fatal("reask should fail fast")
	}
	warnOnly := AgentPolicyFlags{}
	if warnOnly.ValidationRunOptions(false).StopOnFirstEnforce {
		t.Fatal("warn-only should allow parallel dispatch")
	}
	if !warnOnly.ValidationRunOptions(true).StopOnFirstEnforce {
		t.Fatal("shadow should fail fast")
	}
}
