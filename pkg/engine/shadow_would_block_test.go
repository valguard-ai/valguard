package engine

import "testing"

func TestShadowWouldBlockApprox(t *testing.T) {
	run := RunOutput{
		Failures: []ValidationFailure{
			{ValidatorName: "pii", OnFail: "block"},
		},
	}
	if !ShadowWouldBlockApprox("shadow", run) {
		t.Fatal("expected shadow + block failure to count")
	}
	if ShadowWouldBlockApprox("ok", run) {
		t.Fatal("non-shadow status should not count")
	}
	run2 := RunOutput{
		Failures: []ValidationFailure{
			{ValidatorName: "x", OnFail: "warn"},
		},
	}
	if ShadowWouldBlockApprox("shadow", run2) {
		t.Fatal("warn-only failure should not count")
	}
}
