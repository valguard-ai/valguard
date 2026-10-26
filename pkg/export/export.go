package export

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/valguard-ai/valguard/pkg/policy"
)

const playbookSchema = "valguard-playbook-template/1"

var slugSanitizer = regexp.MustCompile(`[^a-z0-9-]+`)

// PlaybookFragment is a minimal importable valguard-playbook-template/1 document.
type PlaybookFragment struct {
	Schema      string         `json:"$schema"`
	Version     int            `json:"version"`
	Name        string         `json:"name"`
	Slug        string         `json:"slug"`
	Description string         `json:"description,omitempty"`
	Agents      []AgentEntry   `json:"agents"`
	Graph       Graph          `json:"graph"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type AgentEntry struct {
	RoleKey       string           `json:"role_key"`
	RoleLabel     string           `json:"role_label"`
	SuggestedSlug string           `json:"suggested_slug,omitempty"`
	Validators    []ValidatorEntry `json:"validators"`
}

type ValidatorEntry struct {
	Name     string         `json:"name"`
	Enabled  bool           `json:"enabled"`
	Priority int            `json:"priority,omitempty"`
	Params   map[string]any `json:"params,omitempty"`
	OnFail   string         `json:"on_fail,omitempty"`
}

type Graph struct {
	EntryStepKey string      `json:"entry_step_key"`
	Steps        []GraphStep `json:"steps"`
	Routes       []any       `json:"routes"`
}

type GraphStep struct {
	StepKey   string `json:"step_key"`
	NodeType  string `json:"node_type"`
	AgentSlug string `json:"agent_slug"`
	InputMode string `json:"input_mode"`
}

// Options tune export output.
type Options struct {
	Slug string // optional override; derived from policy name when empty
}

// FromPolicy maps OSS YAML v1 to a single-agent playbook template fragment.
func FromPolicy(doc policy.Document, opts Options) (PlaybookFragment, error) {
	slug := strings.TrimSpace(opts.Slug)
	if slug == "" {
		slug = SlugFromName(doc.Name)
	}
	if !validSlug(slug) {
		return PlaybookFragment{}, fmt.Errorf("slug %q must match [a-z0-9-]{1,48}", slug)
	}
	roleKey := strings.ReplaceAll(slug, "-", "_")
	if len(roleKey) > 48 {
		roleKey = roleKey[:48]
	}

	validators := make([]ValidatorEntry, 0, len(doc.Rules))
	for i, rule := range doc.Rules {
		validators = append(validators, ValidatorEntry{
			Name:     rule.ID,
			Enabled:  true,
			Priority: (i + 1) * 10,
			Params:   rule.Params,
			OnFail:   rule.OnFail,
		})
	}

	desc := fmt.Sprintf("Exported from OSS policy %q (mode=%s). Import via dashboard validate-then-apply.", doc.Name, doc.Mode)
	frag := PlaybookFragment{
		Schema:      playbookSchema,
		Version:     1,
		Name:        doc.Name,
		Slug:        slug,
		Description: desc,
		Agents: []AgentEntry{{
			RoleKey:       roleKey,
			RoleLabel:     doc.Name,
			SuggestedSlug: slug,
			Validators:    validators,
		}},
		Graph: Graph{
			EntryStepKey: roleKey,
			Steps: []GraphStep{{
				StepKey:   roleKey,
				NodeType:  "agent",
				AgentSlug: "{{" + roleKey + "}}",
				InputMode: "pass_through",
			}},
			Routes: []any{},
		},
		Metadata: map[string]any{
			"exported_at":        time.Now().UTC().Format(time.RFC3339),
			"oss_policy_export":  true,
			"oss_policy_mode":    doc.Mode,
			"oss_policy_version": doc.Version,
			"source":             "valguard export",
		},
	}
	return frag, nil
}

// MarshalJSON indents playbook fragment for CLI output.
func MarshalJSON(frag PlaybookFragment) ([]byte, error) {
	return json.MarshalIndent(frag, "", "  ")
}

func SlugFromName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = slugSanitizer.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "oss-policy"
	}
	if len(s) > 48 {
		s = strings.Trim(s[:48], "-")
	}
	return s
}

func validSlug(slug string) bool {
	if slug == "" || len(slug) > 48 {
		return false
	}
	for _, c := range slug {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			continue
		}
		return false
	}
	return true
}
