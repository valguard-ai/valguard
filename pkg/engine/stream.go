package engine

// CanPassthroughStream reports whether the proxy may forward upstream SSE directly.
// Shadow mode always passthroughs. Block and reask policies use the buffered path
// so enforcement completes before any assistant tokens reach the client.
func CanPassthroughStream(instances []Instance, shadowMode bool) bool {
	return ComputeAgentPolicyFlags(instances).CanPassthroughStream(shadowMode)
}
