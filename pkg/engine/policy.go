package engine

// AgentPolicyFlags summarizes validator on-fail policies from a single scan.
type AgentPolicyFlags struct {
	HasReask   bool
	HasEnforce bool
}

// ComputeAgentPolicyFlags derives chat-path policy flags from enabled validators.
func ComputeAgentPolicyFlags(instances []Instance) AgentPolicyFlags {
	var flags AgentPolicyFlags
	for _, inst := range instances {
		if !inst.Enabled {
			continue
		}
		switch inst.OnFail {
		case "reask":
			flags.HasReask = true
			flags.HasEnforce = true
		case "block":
			flags.HasEnforce = true
		}
	}
	return flags
}

// CanPassthroughStream reports whether live SSE passthrough is safe for these flags.
// Block and reask policies require the buffered path so clients never receive
// assistant tokens before enforcement completes.
func (f AgentPolicyFlags) CanPassthroughStream(shadowMode bool) bool {
	if shadowMode {
		return true
	}
	return !f.HasEnforce
}

// MaxAttempts returns upstream retry budget for the buffered chat path.
func (f AgentPolicyFlags) MaxAttempts(shadowMode bool, maxReasks int) int {
	if shadowMode || !f.HasReask {
		return 1
	}
	return 1 + maxReasks
}

// ValidationRunOptions picks dispatch options for live chat/output validation.
func (f AgentPolicyFlags) ValidationRunOptions(shadowMode bool) RunOptions {
	if shadowMode || f.HasEnforce {
		return RunOptions{StopOnFirstEnforce: true}
	}
	return RunOptions{}
}
