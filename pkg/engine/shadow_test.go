package engine

import "testing"

func TestApplyShadowMode_passthrough(t *testing.T) {
	run := RunOutput{
		Blocked:     true,
		ShouldReask: true,
		Failures: []ValidationFailure{
			{ValidatorName: "valid_format", OnFail: "block"},
		},
	}
	if !WouldEnforce(run) {
		t.Fatal("expected would enforce")
	}
	shadow := ApplyShadowMode(run)
	if shadow.Blocked || shadow.ShouldReask {
		t.Fatal("shadow must not block or reask")
	}
	if len(shadow.Failures) != 1 {
		t.Fatal("failures should remain for logging")
	}
}
