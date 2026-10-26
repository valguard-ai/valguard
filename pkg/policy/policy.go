package policy

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/valguard-ai/valguard/pkg/engine"
	"github.com/valguard-ai/valguard/pkg/validators/mvp"
)

// B7 DoS limits for OSS CLI / CI (see docs/oss/policy-yaml-v1.md).
const (
	MaxPolicyBytes = 256 << 10
	MaxOutputBytes = 4 << 20
)

// DefaultValidationTimeout caps a single check/test validation run.
const DefaultValidationTimeout = 30 * time.Second

type Document struct {
	Version int    `yaml:"version" json:"version"`
	Name    string `yaml:"name" json:"name"`
	Mode    string `yaml:"mode" json:"mode"`
	Rules   []Rule `yaml:"rules" json:"rules"`
}

type Rule struct {
	ID     string         `yaml:"id" json:"id"`
	Params map[string]any `yaml:"params" json:"params"`
	OnFail string         `yaml:"on_fail" json:"on_fail"`
}

func LoadFile(path string) (Document, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Document{}, err
	}
	if len(raw) > MaxPolicyBytes {
		return Document{}, fmt.Errorf("policy file exceeds %d bytes", MaxPolicyBytes)
	}
	return Parse(raw)
}

func Parse(raw []byte) (Document, error) {
	if len(raw) > MaxPolicyBytes {
		return Document{}, fmt.Errorf("policy exceeds %d bytes", MaxPolicyBytes)
	}
	var doc Document
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return Document{}, err
	}
	if doc.Version != 1 {
		return Document{}, fmt.Errorf("unsupported policy version %d (want 1)", doc.Version)
	}
	if strings.TrimSpace(doc.Name) == "" {
		return Document{}, fmt.Errorf("policy name is required")
	}
	if len(doc.Rules) == 0 {
		return Document{}, fmt.Errorf("policy must include at least one rule")
	}
	mode := strings.TrimSpace(doc.Mode)
	if mode == "" {
		doc.Mode = "enforce"
	} else if mode != "enforce" && mode != "shadow" {
		return Document{}, fmt.Errorf("mode must be enforce or shadow")
	}
	for i := range doc.Rules {
		id := strings.TrimSpace(doc.Rules[i].ID)
		if id == "" {
			return Document{}, fmt.Errorf("rules[%d].id is required", i)
		}
		doc.Rules[i].ID = id
		if !mvp.IsRuleID(id) {
			return Document{}, fmt.Errorf("rules[%d]: %q is not in the OSS MVP rule set (40 rules); use hosted ValGuard for the full catalog", i, id)
		}
		onFail := strings.TrimSpace(doc.Rules[i].OnFail)
		if onFail == "" {
			doc.Rules[i].OnFail = "block"
		} else if onFail == "reask" {
			return Document{}, fmt.Errorf("rules[%d]: reask is not supported in OSS MVP", i)
		} else if onFail != "block" && onFail != "warn" && onFail != "log" {
			return Document{}, fmt.Errorf("rules[%d]: invalid on_fail %q", i, onFail)
		}
		if doc.Rules[i].Params == nil {
			doc.Rules[i].Params = map[string]any{}
		}
	}
	return doc, nil
}

func (doc Document) ToEngineInstances() []engine.Instance {
	out := make([]engine.Instance, 0, len(doc.Rules))
	for i, rule := range doc.Rules {
		out = append(out, engine.Instance{
			ID:       fmt.Sprintf("oss-%d", i),
			Name:     rule.ID,
			Enabled:  true,
			Priority: int32((i + 1) * 10),
			OnFail:   rule.OnFail,
			Params:   rule.Params,
		})
	}
	return out
}

const SampleYAML = `version: 1
name: my_agent
mode: enforce
rules:
  - id: valid_format
    params:
      format: json
    on_fail: block
  - id: required_fields
    params:
      fields: ["message"]
    on_fail: block
`
