package report

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/valguard-ai/valguard/pkg/policy"
)

const maxExampleLen = 120
const maxExamplesPerRule = 3

// InputRecord is one model output to evaluate in shadow report mode.
type InputRecord struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Output    string             `json:"output"`
	InputText string             `json:"input_text"`
	Policy    policy.Document `json:"-"`
}

// RuleExample is a sample output that would trigger a rule.
type RuleExample struct {
	RecordID   string `json:"record_id"`
	RecordName string `json:"record_name,omitempty"`
	Snippet    string `json:"snippet"`
	Message    string `json:"message,omitempty"`
}

// RuleAggregate counts would-enforce hits for one rule ID.
type RuleAggregate struct {
	RuleID          string        `json:"rule_id"`
	WouldBlockCount int           `json:"would_block_count"`
	Examples        []RuleExample `json:"examples,omitempty"`
}

// ShadowReport is aggregated shadow / would-enforce statistics.
type ShadowReport struct {
	PolicyName         string          `json:"policy_name"`
	PolicyMode         string          `json:"policy_mode"`
	TotalRecords       int             `json:"total_records"`
	RecordsWouldBlock  int             `json:"records_would_block"`
	Rules              []RuleAggregate `json:"rules"`
	CTAExportHint      string          `json:"-"`
}

// BuildShadowReport evaluates each record against policy (would-enforce semantics).
// When record.Policy is set (fixture dir), it overrides doc for that row.
func BuildShadowReport(doc policy.Document, records []InputRecord) (ShadowReport, error) {
	report := ShadowReport{
		PolicyName:    doc.Name,
		PolicyMode:    doc.Mode,
		CTAExportHint: "Enable the same policy in hosted shadow mode: valguard export --policy valguard.yaml",
	}
	byRule := map[string]*RuleAggregate{}
	order := make([]string, 0)

	for _, rec := range records {
		evalDoc := doc
		if rec.Policy.Version != 0 {
			evalDoc = rec.Policy
			if report.PolicyName == "" {
				report.PolicyName = evalDoc.Name
				report.PolicyMode = evalDoc.Mode
			}
		}
		for _, rule := range evalDoc.Rules {
			if _, ok := byRule[rule.ID]; !ok {
				byRule[rule.ID] = &RuleAggregate{RuleID: rule.ID}
				order = append(order, rule.ID)
			}
		}

		report.TotalRecords++
		detail, err := policy.RunDetailed(evalDoc, rec.Output, rec.InputText)
		if err != nil {
			return ShadowReport{}, err
		}
		if detail.WouldEnforce {
			report.RecordsWouldBlock++
		}
		for _, rr := range detail.Rules {
			if !rr.WouldEnforce {
				continue
			}
			agg, ok := byRule[rr.RuleID]
			if !ok {
				agg = &RuleAggregate{RuleID: rr.RuleID}
				byRule[rr.RuleID] = agg
				order = append(order, rr.RuleID)
			}
			agg.WouldBlockCount++
			if len(agg.Examples) < maxExamplesPerRule {
				agg.Examples = append(agg.Examples, RuleExample{
					RecordID:   rec.ID,
					RecordName: rec.Name,
					Snippet:    snippet(rec.Output),
					Message:    rr.Message,
				})
			}
		}
	}

	report.Rules = make([]RuleAggregate, 0, len(order))
	for _, id := range order {
		if agg := byRule[id]; agg != nil && agg.WouldBlockCount > 0 {
			report.Rules = append(report.Rules, *agg)
		}
	}
	sort.Slice(report.Rules, func(i, j int) bool {
		if report.Rules[i].WouldBlockCount != report.Rules[j].WouldBlockCount {
			return report.Rules[i].WouldBlockCount > report.Rules[j].WouldBlockCount
		}
		return report.Rules[i].RuleID < report.Rules[j].RuleID
	})
	return report, nil
}

func snippet(text string) string {
	s := strings.TrimSpace(text)
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxExampleLen {
		return s
	}
	return s[:maxExampleLen-3] + "..."
}

// LoadJSONL reads newline-delimited JSON records ({ "output", optional "input_text", "id", "name" }).
func LoadJSONL(r io.Reader) ([]InputRecord, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 4<<20)
	var out []InputRecord
	line := 0
	for sc.Scan() {
		line++
		raw := strings.TrimSpace(sc.Text())
		if raw == "" {
			continue
		}
		var rec InputRecord
		if err := json.Unmarshal([]byte(raw), &rec); err != nil {
			return nil, fmt.Errorf("jsonl line %d: %w", line, err)
		}
		if strings.TrimSpace(rec.Output) == "" {
			return nil, fmt.Errorf("jsonl line %d: output is required", line)
		}
		if rec.ID == "" {
			rec.ID = fmt.Sprintf("line-%d", line)
		}
		out = append(out, rec)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("jsonl: no records")
	}
	return out, nil
}

// LoadFixtureDir loads conformance-style fixture JSON files as report inputs.
func LoadFixtureDir(dir string) ([]InputRecord, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []InputRecord
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, ent.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var fx struct {
			Name      string             `json:"name"`
			Policy    policy.Document `json:"policy"`
			Output    string             `json:"output"`
			InputText string             `json:"input_text"`
		}
		if err := json.Unmarshal(raw, &fx); err != nil {
			return nil, fmt.Errorf("%s: %w", ent.Name(), err)
		}
		id := strings.TrimSuffix(ent.Name(), ".json")
		out = append(out, InputRecord{
			ID:        id,
			Name:      fx.Name,
			Output:    fx.Output,
			InputText: fx.InputText,
			Policy:    fx.Policy,
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("fixture dir: no .json files in %s", dir)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// FormatMarkdown renders a PR-friendly shadow summary.
func FormatMarkdown(r ShadowReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# ValGuard shadow report — %s\n\n", r.PolicyName)
	fmt.Fprintf(&b, "Policy mode: `%s`\n\n", r.PolicyMode)
	fmt.Fprintf(&b, "| Metric | Value |\n|--------|------:|\n")
	fmt.Fprintf(&b, "| Records evaluated | %d |\n", r.TotalRecords)
	fmt.Fprintf(&b, "| Records that would block | %d |\n\n", r.RecordsWouldBlock)
	if len(r.Rules) == 0 {
		b.WriteString("_No rules would enforce on this sample._\n\n")
	} else {
		b.WriteString("## Would-enforce by rule\n\n")
		b.WriteString("| Rule | Would block | Examples |\n|------|------------:|----------|\n")
		for _, rule := range r.Rules {
			ex := formatExampleList(rule.Examples)
			fmt.Fprintf(&b, "| `%s` | %d | %s |\n", rule.RuleID, rule.WouldBlockCount, ex)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "\n%s\n", r.CTAExportHint)
	return b.String()
}

func formatExampleList(ex []RuleExample) string {
	if len(ex) == 0 {
		return "—"
	}
	parts := make([]string, 0, len(ex))
	for _, e := range ex {
		label := e.RecordID
		if e.RecordName != "" {
			label = e.RecordName
		}
		parts = append(parts, fmt.Sprintf("`%s`: %s", label, e.Snippet))
	}
	return strings.Join(parts, "; ")
}

// FormatJUnit emits a CI-friendly testsuite (one testcase per record).
func FormatJUnit(r ShadowReport, defaultDoc policy.Document, records []InputRecord) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, `<?xml version="1.0" encoding="UTF-8"?>`+"\n")
	fmt.Fprintf(&b, `<testsuite name=%q tests=%d failures=%d>`+"\n",
		xmlEscape(r.PolicyName), len(records), r.RecordsWouldBlock)
	for _, rec := range records {
		name := rec.Name
		if name == "" {
			name = rec.ID
		}
		evalDoc := defaultDoc
		if rec.Policy.Version != 0 {
			evalDoc = rec.Policy
		}
		detail, err := policy.RunDetailed(evalDoc, rec.Output, rec.InputText)
		if err != nil {
			return "", err
		}
		if !detail.WouldEnforce {
			fmt.Fprintf(&b, `  <testcase name=%q/>`+"\n", xmlEscape(name))
			continue
		}
		var failed []string
		for _, rr := range detail.Rules {
			if rr.WouldEnforce {
				failed = append(failed, rr.RuleID)
			}
		}
		msg := fmt.Sprintf("would enforce: %s", strings.Join(failed, ", "))
		fmt.Fprintf(&b, `  <testcase name=%q>`+"\n", xmlEscape(name))
		fmt.Fprintf(&b, `    <failure message=%q>%s</failure>`+"\n", xmlEscape(msg), xmlEscape(msg))
		fmt.Fprintf(&b, `  </testcase>`+"\n")
	}
	b.WriteString("</testsuite>\n")
	return b.String(), nil
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
