package report

import (
	"strings"
	"testing"

	"github.com/valguard-ai/valguard/pkg/policy"
)

func TestBuildShadowReport_aggregatesRules(t *testing.T) {
	doc := policy.Document{
		Version: 1,
		Name:    "demo",
		Mode:    "shadow",
		Rules: []policy.Rule{
			{ID: "forbidden_patterns", Params: map[string]any{"patterns": []any{"SECRET"}}, OnFail: "block"},
		},
	}
	records := []InputRecord{
		{ID: "a", Output: "ok"},
		{ID: "b", Output: "leaked SECRET"},
		{ID: "c", Output: "another SECRET leak"},
	}
	rep, err := BuildShadowReport(doc, records)
	if err != nil {
		t.Fatal(err)
	}
	if rep.TotalRecords != 3 || rep.RecordsWouldBlock != 2 {
		t.Fatalf("counts: total=%d would=%d", rep.TotalRecords, rep.RecordsWouldBlock)
	}
	if len(rep.Rules) != 1 || rep.Rules[0].WouldBlockCount != 2 {
		t.Fatalf("rule agg: %+v", rep.Rules)
	}
}

func TestFormatMarkdown_containsTable(t *testing.T) {
	rep := ShadowReport{
		PolicyName:        "demo",
		PolicyMode:        "shadow",
		TotalRecords:      1,
		RecordsWouldBlock: 1,
		CTAExportHint:     "valguard export",
		Rules: []RuleAggregate{{
			RuleID:          "valid_format",
			WouldBlockCount: 1,
			Examples:        []RuleExample{{RecordID: "x", Snippet: "bad"}},
		}},
	}
	md := FormatMarkdown(rep)
	if !strings.Contains(md, "valid_format") || !strings.Contains(md, "valguard export") {
		t.Fatalf("markdown: %s", md)
	}
}

func TestFormatJUnit_failureCase(t *testing.T) {
	doc := policy.Document{
		Version: 1,
		Name:    "demo",
		Mode:    "shadow",
		Rules: []policy.Rule{
			{ID: "forbidden_patterns", Params: map[string]any{"patterns": []any{"SECRET"}}, OnFail: "block"},
		},
	}
	records := []InputRecord{{ID: "1", Name: "leak", Output: "SECRET"}}
	rep, err := BuildShadowReport(doc, records)
	if err != nil {
		t.Fatal(err)
	}
	xml, err := FormatJUnit(rep, doc, records)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(xml, `<failure`) || !strings.Contains(xml, "forbidden_patterns") {
		t.Fatalf("junit: %s", xml)
	}
}

func TestLoadJSONL(t *testing.T) {
	raw := strings.NewReader(`{"output":"hello"}
{"id":"b","output":"world","input_text":"ctx"}`)
	recs, err := LoadJSONL(raw)
	if err != nil || len(recs) != 2 || recs[0].ID != "line-1" {
		t.Fatalf("load: err=%v recs=%+v", err, recs)
	}
}
