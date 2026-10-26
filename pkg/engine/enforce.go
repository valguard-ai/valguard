package engine

// HasEnforceOnFail reports whether any enabled validator uses block or reask on failure.
func HasEnforceOnFail(instances []Instance) bool {
	return ComputeAgentPolicyFlags(instances).HasEnforce
}

// HasReaskOnFail reports whether any enabled validator may trigger an upstream reask loop.
func HasReaskOnFail(instances []Instance) bool {
	return ComputeAgentPolicyFlags(instances).HasReask
}

// ValidationRunOptions picks dispatch options for live chat/output validation.
// Enforce policies (block/reask) or org shadow mode use fail-fast sequential dispatch.
// Warn/pass-only agents may run rules in parallel when the pack is large.
func ValidationRunOptions(instances []Instance, shadowMode bool) RunOptions {
	return ComputeAgentPolicyFlags(instances).ValidationRunOptions(shadowMode)
}
