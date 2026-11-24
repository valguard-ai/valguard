package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/valguard-ai/valguard/pkg/evidence"
	"github.com/valguard-ai/valguard/pkg/export"
	"github.com/valguard-ai/valguard/pkg/policy"
	"github.com/valguard-ai/valguard/pkg/report"
	"github.com/valguard-ai/valguard/pkg/rules"
	"github.com/valguard-ai/valguard/pkg/version"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "init":
		os.Exit(runInit(os.Args[2:]))
	case "check":
		os.Exit(runCheck(os.Args[2:]))
	case "test":
		os.Exit(runTestFixtures(os.Args[2:]))
	case "verify-proof":
		os.Exit(runVerifyProof(os.Args[2:]))
	case "rules":
		os.Exit(runRules(os.Args[2:]))
	case "report":
		os.Exit(runReport(os.Args[2:]))
	case "export":
		os.Exit(runExport(os.Args[2:]))
	case "version", "-version", "--version":
		fmt.Println(version.Version)
		os.Exit(0)
	default:
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Println(`valguard — offline validation CLI (40 MVP rules)

  valguard init [--path valguard.yaml]
  valguard check [--policy valguard.yaml] (--file PATH | --output TEXT | stdin)
  valguard test [--dir conformance/fixtures]
  valguard report --shadow [--policy valguard.yaml] (--file runs.jsonl | --dir conformance/fixtures) [--format markdown|junit|json]
  valguard export [--policy valguard.yaml] [--slug agent-slug] [--out playbook.json]
  valguard verify-proof --file envelope.json
  valguard version
  valguard rules list
  valguard rules explain <rule-id> [--json]

No network in check/test/report/export/verify-proof.
Hosted network CI uses a separate helper (valguard-cli) against agent_slug — not this binary.
Policy YAML v1: version, name, mode, rules[] (on_fail: block|warn|log).`)
}

func runInit(args []string) int {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	path := fs.String("path", "valguard.yaml", "")
	_ = fs.Parse(args)
	if _, err := os.Stat(*path); err == nil {
		fmt.Fprintf(os.Stderr, "%s already exists\n", *path)
		return 2
	}
	if err := os.WriteFile(*path, []byte(policy.SampleYAML), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("wrote %s\n", *path)
	return 0
}

func runCheck(args []string) int {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	policyPath := fs.String("policy", "valguard.yaml", "")
	file := fs.String("file", "", "")
	output := fs.String("output", "", "")
	inputText := fs.String("input", "", "")
	jsonOut := fs.Bool("json", false, "")
	timeout := fs.Duration("timeout", policy.DefaultValidationTimeout, "max wall time for validation")
	_ = fs.Parse(args)

	doc, err := policy.LoadFile(*policyPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	outText, err := readOutput(*file, *output)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	detail, err := policy.RunDetailedWithTimeout(doc, outText, strings.TrimSpace(*inputText), *timeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	res := detail.RunResult
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(res)
	} else {
		printHumanResult(res)
	}
	if doc.Mode == "shadow" {
		if res.WouldEnforce {
			return 1
		}
		return 0
	}
	if res.Blocked {
		return 1
	}
	return 0
}

func readOutput(filePath, inline string) (string, error) {
	if strings.TrimSpace(inline) != "" {
		if len(inline) > policy.MaxOutputBytes {
			return "", fmt.Errorf("output exceeds %d bytes", policy.MaxOutputBytes)
		}
		return inline, nil
	}
	if filePath == "" || filePath == "-" {
		raw, err := io.ReadAll(io.LimitReader(os.Stdin, policy.MaxOutputBytes+1))
		if err != nil {
			return "", err
		}
		if len(raw) > policy.MaxOutputBytes {
			return "", fmt.Errorf("stdin exceeds %d bytes", policy.MaxOutputBytes)
		}
		return string(raw), nil
	}
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	if len(raw) > policy.MaxOutputBytes {
		return "", fmt.Errorf("file exceeds %d bytes", policy.MaxOutputBytes)
	}
	return string(raw), nil
}

func printHumanResult(res policy.RunResult) {
	if res.Blocked {
		fmt.Println("blocked")
	} else if res.WouldEnforce && res.Shadow {
		fmt.Println("would block (shadow)")
	} else {
		fmt.Println("pass")
	}
	if len(res.FailedRules) > 0 {
		fmt.Println("failed:", strings.Join(res.FailedRules, ", "))
	}
}

func runVerifyProof(args []string) int {
	fs := flag.NewFlagSet("verify-proof", flag.ExitOnError)
	file := fs.String("file", "", "")
	_ = fs.Parse(args)
	path := strings.TrimSpace(*file)
	var raw []byte
	var err error
	if path == "" || path == "-" {
		raw, err = io.ReadAll(os.Stdin)
	} else {
		raw, err = os.ReadFile(path)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	rec, err := evidence.ParseRecord(raw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid JSON envelope:", err)
		return 2
	}
	if err := evidence.Verify(rec.KeyID, rec.Payload, rec.Signature); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println("ok")
	return 0
}

func runRules(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: valguard rules list | valguard rules explain <rule-id>")
		return 2
	}
	switch args[0] {
	case "list":
		for _, id := range rules.MVPRuleIDs {
			fmt.Println(id)
		}
		return 0
	case "explain":
		return runRulesExplain(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "usage: valguard rules list | valguard rules explain <rule-id>")
		return 2
	}
}

func runRulesExplain(args []string) int {
	fs := flag.NewFlagSet("rules explain", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "")
	_ = fs.Parse(args)
	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(os.Stderr, "usage: valguard rules explain <rule-id> [--json]")
		return 2
	}
	ex, err := rules.ExplainRule(rest[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(ex)
		return 0
	}
	fmt.Printf("id:          %s\n", ex.ID)
	fmt.Printf("oss:         %s\n", displayOSS(ex.OSS))
	fmt.Printf("category:    %s\n", ex.Category)
	fmt.Printf("description: %s\n", ex.Description)
	fmt.Printf("doc:         %s\n", ex.DocURL)
	return 0
}

func displayOSS(status string) string {
	if status == "" {
		return "not in OSS MVP set (production-only)"
	}
	return status
}

func runTestFixtures(args []string) int {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	dir := fs.String("dir", "conformance/fixtures", "")
	timeout := fs.Duration("timeout", policy.DefaultValidationTimeout, "max wall time per fixture validation")
	_ = fs.Parse(args)
	entries, err := os.ReadDir(*dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	failed := 0
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".json") {
			continue
		}
		if err := runFixtureFile(filepath.Join(*dir, ent.Name()), *timeout); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", ent.Name(), err)
			failed++
		}
	}
	if failed > 0 {
		return 1
	}
	return 0
}

type fixtureFile struct {
	Name                string             `json:"name"`
	Policy              policy.Document `json:"policy"`
	Output              string             `json:"output"`
	InputText           string             `json:"input_text"`
	ExpectBlocked       bool               `json:"expect_blocked"`
	ExpectWouldEnforce  *bool              `json:"expect_would_enforce,omitempty"`
	ExpectFailedRules   []string           `json:"expect_failed_rules,omitempty"`
}

func runReport(args []string) int {
	if len(args) < 1 || args[0] != "--shadow" {
		fmt.Fprintln(os.Stderr, "usage: valguard report --shadow [--policy valguard.yaml] (--file runs.jsonl | --dir conformance/fixtures) [--format markdown|junit|json]")
		return 2
	}
	fs := flag.NewFlagSet("report", flag.ExitOnError)
	policyPath := fs.String("policy", "valguard.yaml", "")
	file := fs.String("file", "", "")
	dir := fs.String("dir", "", "")
	format := fs.String("format", "markdown", "")
	outPath := fs.String("out", "", "")
	_ = fs.Parse(args[1:])

	if *file == "" && *dir == "" {
		fmt.Fprintln(os.Stderr, "report --shadow requires --file or --dir")
		return 2
	}
	if *file != "" && *dir != "" {
		fmt.Fprintln(os.Stderr, "use only one of --file or --dir")
		return 2
	}

	var records []report.InputRecord
	var err error
	var policyDoc policy.Document
	policyLoaded := false

	if *dir != "" {
		records, err = report.LoadFixtureDir(*dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		if _, err := os.Stat(*policyPath); err == nil {
			policyDoc, err = policy.LoadFile(*policyPath)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 2
			}
			policyLoaded = true
		}
	} else {
		policyDoc, err = policy.LoadFile(*policyPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		policyLoaded = true
		f, err := os.Open(*file)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		defer f.Close()
		records, err = report.LoadJSONL(f)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
	}

	if !policyLoaded && len(records) > 0 && records[0].Policy.Version != 0 {
		policyDoc = records[0].Policy
	} else if !policyLoaded {
		fmt.Fprintln(os.Stderr, "report --shadow: provide --policy or use --dir with fixture policies")
		return 2
	}

	rep, err := report.BuildShadowReport(policyDoc, records)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	var output string
	switch strings.ToLower(*format) {
	case "markdown", "md":
		output = report.FormatMarkdown(rep)
	case "junit", "xml":
		evalDoc := policyDoc
		output, err = report.FormatJUnit(rep, evalDoc, records)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
	case "json":
		raw, err := json.MarshalIndent(rep, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		output = string(raw) + "\n"
	default:
		fmt.Fprintf(os.Stderr, "unknown format %q\n", *format)
		return 2
	}

	if err := writeOut(*outPath, output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func writeOut(path, content string) error {
	if path == "" {
		_, err := io.WriteString(os.Stdout, content)
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func runExport(args []string) int {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	policyPath := fs.String("policy", "valguard.yaml", "")
	slug := fs.String("slug", "", "")
	outPath := fs.String("out", "", "")
	_ = fs.Parse(args)

	doc, err := policy.LoadFile(*policyPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	frag, err := export.FromPolicy(doc, export.Options{Slug: *slug})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	raw, err := export.MarshalJSON(frag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if *outPath == "" {
		_, _ = os.Stdout.Write(raw)
		_, _ = os.Stdout.Write([]byte("\n"))
		return 0
	}
	if err := os.WriteFile(*outPath, raw, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("wrote %s\n", *outPath)
	return 0
}

func runFixtureFile(path string, timeout time.Duration) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var fx fixtureFile
	if err := json.Unmarshal(raw, &fx); err != nil {
		return err
	}
	if fx.Policy.Version == 0 {
		return fmt.Errorf("fixture missing policy")
	}
	if len(fx.Output) > policy.MaxOutputBytes {
		return fmt.Errorf("fixture output exceeds %d bytes", policy.MaxOutputBytes)
	}
	detail, err := policy.RunDetailedWithTimeout(fx.Policy, fx.Output, fx.InputText, timeout)
	if err != nil {
		return err
	}
	res := detail.RunResult
	expectWould := fx.ExpectWouldEnforce
	if expectWould == nil {
		v := fx.ExpectBlocked
		if fx.Policy.Mode != "shadow" {
			expectWould = &v
		} else {
			expectWould = &fx.ExpectBlocked
		}
	}
	if res.WouldEnforce != *expectWould {
		return fmt.Errorf("would_enforce=%v want %v (blocked=%v failed=%v)", res.WouldEnforce, *expectWould, res.Blocked, res.FailedRules)
	}
	effectiveBlocked := res.Blocked
	if fx.Policy.Mode == "shadow" {
		effectiveBlocked = res.WouldEnforce
	}
	if effectiveBlocked != fx.ExpectBlocked {
		return fmt.Errorf("effective_blocked=%v want %v failed=%v", effectiveBlocked, fx.ExpectBlocked, res.FailedRules)
	}
	if fx.ExpectFailedRules != nil {
		want := append([]string(nil), fx.ExpectFailedRules...)
		slices.Sort(want)
		got := append([]string(nil), res.FailedRules...)
		slices.Sort(got)
		if !slices.Equal(want, got) {
			return fmt.Errorf("failed_rules=%v want %v", got, want)
		}
	}
	return nil
}

