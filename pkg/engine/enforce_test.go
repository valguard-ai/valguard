package engine

import "testing"

func TestHasEnforceOnFail(t *testing.T) {
	tests := []struct {
		name string
		inst []Instance
		want bool
	}{
		{"empty", nil, false},
		{"warn only", []Instance{{Enabled: true, OnFail: "warn"}}, false},
		{"block", []Instance{{Enabled: true, OnFail: "block"}}, true},
		{"reask", []Instance{{Enabled: true, OnFail: "reask"}}, true},
		{"disabled block", []Instance{{Enabled: false, OnFail: "block"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasEnforceOnFail(tt.inst); got != tt.want {
				t.Fatalf("HasEnforceOnFail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidationRunOptions_warnPassParallel(t *testing.T) {
	inst := []Instance{{Enabled: true, OnFail: "warn"}}
	opts := ValidationRunOptions(inst, false)
	if opts.StopOnFirstEnforce {
		t.Fatal("warn/pass-only agents should not use StopOnFirstEnforce")
	}
}

func TestValidationRunOptions_shadowFailFast(t *testing.T) {
	inst := []Instance{{Enabled: true, OnFail: "warn"}}
	opts := ValidationRunOptions(inst, true)
	if !opts.StopOnFirstEnforce {
		t.Fatal("shadow mode should use StopOnFirstEnforce")
	}
}

func TestValidationRunOptions_blockFailFast(t *testing.T) {
	inst := []Instance{{Enabled: true, OnFail: "block"}}
	opts := ValidationRunOptions(inst, false)
	if !opts.StopOnFirstEnforce {
		t.Fatal("block policy should use StopOnFirstEnforce")
	}
}

func TestCanPassthroughStream_enforceUsesBufferedPath(t *testing.T) {
	block := []Instance{{Enabled: true, OnFail: "block"}}
	if CanPassthroughStream(block, false) {
		t.Fatal("block policy must use buffered stream path")
	}
	reask := []Instance{{Enabled: true, OnFail: "reask"}}
	if CanPassthroughStream(reask, false) {
		t.Fatal("reask requires buffered stream path")
	}
}
