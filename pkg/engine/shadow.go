package engine

import "github.com/valguard-ai/valguard/pkg/validators"

func ApplyShadowMode(run RunOutput) RunOutput {
	run.Blocked = false
	run.ShouldReask = false
	return run
}

func WouldEnforce(run RunOutput) bool {
	return run.Blocked || run.ShouldReask
}

func InstanceWouldEnforce(inst Instance, r validators.Result) bool {
	if !inst.Enabled || r.Status != "fail" {
		return false
	}
	switch inst.OnFail {
	case "block", "reask":
		return true
	default:
		return false
	}
}
