package export

import (
	"encoding/json"
	"testing"

	"github.com/valguard-ai/valguard/pkg/policy"
)

func TestFromPolicy_playbookSchema(t *testing.T) {
	doc := policy.Document{
		Version: 1,
		Name:    "Invoice Agent",
		Mode:    "shadow",
		Rules: []policy.Rule{
			{ID: "valid_format", Params: map[string]any{"format": "json"}, OnFail: "block"},
			{ID: "required_fields", Params: map[string]any{"fields": []any{"total"}}, OnFail: "warn"},
		},
	}
	frag, err := FromPolicy(doc, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if frag.Schema != playbookSchema || frag.Slug != "invoice-agent" {
		t.Fatalf("schema/slug: %+v", frag)
	}
	if len(frag.Agents) != 1 || len(frag.Agents[0].Validators) != 2 {
		t.Fatalf("agents: %+v", frag.Agents)
	}
	if frag.Metadata["oss_policy_export"] != true {
		t.Fatalf("metadata: %+v", frag.Metadata)
	}
	raw, err := MarshalJSON(frag)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["$schema"] != playbookSchema {
		t.Fatalf("json schema field: %v", decoded["$schema"])
	}
}

func TestSlugFromName(t *testing.T) {
	if SlugFromName("My Agent!!!") != "my-agent" {
		t.Fatal(SlugFromName("My Agent!!!"))
	}
}
