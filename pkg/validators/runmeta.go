package validators

import (
	"context"
	"strings"
)

// RAGSource is a retrieved chunk passed with the request (X-VG-Sources header).
type RAGSource struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// RunMeta carries request-level data for meta validators.
type RunMeta struct {
	LatencyMs        int         `json:"latency_ms,omitempty"`
	PromptTokens     int         `json:"prompt_tokens,omitempty"`
	CompletionTokens int         `json:"completion_tokens,omitempty"`
	InputText        string      `json:"input_text,omitempty"`
	Model            string      `json:"model,omitempty"`
	FinishReason     string      `json:"finish_reason,omitempty"`
	RAGSources       []RAGSource `json:"rag_sources,omitempty"`
	ReaskAttempt     int         `json:"reask_attempt,omitempty"`
	PriorOutputs     []string    `json:"prior_outputs,omitempty"`
}

// IsZero reports whether m carries no request-level metadata.
func (m RunMeta) IsZero() bool {
	return m.LatencyMs == 0 &&
		m.PromptTokens == 0 &&
		m.CompletionTokens == 0 &&
		m.InputText == "" &&
		m.Model == "" &&
		m.FinishReason == "" &&
		len(m.RAGSources) == 0 &&
		m.ReaskAttempt == 0 &&
		len(m.PriorOutputs) == 0
}

type metaContextKey struct{}
type lazyOutputContextKey struct{}

// MetaContextKey is the context key for RunMeta (read via RunMetaFrom).
var MetaContextKey = metaContextKey{}

// LazyOutputContextKey exposes pooled lazy output trim for JSON format checks.
var LazyOutputContextKey = lazyOutputContextKey{}

// RunMetaFrom returns request meta attached to ctx (empty when unset).
func RunMetaFrom(ctx context.Context) RunMeta {
	if ctx == nil {
		return RunMeta{}
	}
	if m, ok := ctx.Value(MetaContextKey).(RunMeta); ok {
		return m
	}
	return RunMeta{}
}

// buildRagSourceIDs returns stable source IDs for RAG validators.
func buildRagSourceIDs(sources []RAGSource) []string {
	if len(sources) == 0 {
		return nil
	}
	ids := make([]string, 0, len(sources))
	for _, s := range sources {
		id := strings.TrimSpace(s.ID)
		if id == "" {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

// LazyOutputFrom returns lazy-trimmed output when the proxy attachCtx set it.
func LazyOutputFrom(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if s, ok := ctx.Value(LazyOutputContextKey).(string); ok {
		return s
	}
	return ""
}
