package engine

// WouldBlockOnFail reports whether any logged failure uses on_fail=block (shadow would-block metric).
func WouldBlockOnFail(run RunOutput) bool {
	for _, f := range run.Failures {
		if f.OnFail == "block" {
			return true
		}
	}
	return false
}

// ShadowWouldBlockApprox is true when the request finished as shadow and would have blocked on a block rule.
func ShadowWouldBlockApprox(status string, run RunOutput) bool {
	return status == "shadow" && WouldBlockOnFail(run)
}
