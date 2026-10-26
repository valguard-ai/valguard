//go:build !osspublic

package validators

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func phaseAResultsEqual(a, b Result) bool {
	if a.Status != b.Status || a.Severity != b.Severity || a.Message != b.Message {
		return false
	}
	return reflect.DeepEqual(a.Details, b.Details)
}

// validateJSONStringFieldsLegacy is the pre-PS-42 deep-parse implementation kept
// for parity regression tests.
func validateJSONStringFieldsLegacy(ctx context.Context, output string, params map[string]any, check func(string) bool, label string) Result {
	fields := jsonFieldsFromParams(params)
	if len(fields) == 0 {
		return skipResult("no fields configured")
	}
	data, err := parseJSONObjectCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	var bad []string
	for _, f := range fields {
		val, ok := data[f]
		if !ok {
			continue
		}
		s, ok := val.(string)
		if !ok {
			bad = append(bad, f+": not a string")
			continue
		}
		if !check(s) {
			bad = append(bad, f)
		}
	}
	if len(bad) > 0 {
		return failResult("error", label, map[string]any{"invalid": bad})
	}
	return passResult("error", label+" ok")
}

func TestValidateJSONStringFields_parity(t *testing.T) {
	ctx := context.Background()
	alwaysOK := func(string) bool { return true }
	alwaysBad := func(string) bool { return false }

	params := map[string]any{"fields": []any{"token", "version", "color", "ip", "extra"}}
	outputs := []string{
		`{"token":"aGVsbG8=","version":"1.2.3","color":"#abc","ip":"127.0.0.1"}`,
		`{"token":123,"version":true,"color":null,"ip":{"x":1},"extra":[1]}`,
		`{"token":"bad","version":"not-semver"}`,
		`{"token":"aGVsbG8=","version":"1.0.0","color":"#fff","ip":"::1"}`,
		`{"token":"with \"quotes\" and \\backslash"}`,
		`{"token":"unicode \u00E9"}`,
		`not json`,
		`{"token":"x"}`,
		`  {"token":"aGVsbG8="}  `,
		`{}`,
	}
	checks := []struct {
		name  string
		check func(string) bool
		label string
	}{
		{"alwaysOK", alwaysOK, "label ok"},
		{"alwaysBad", alwaysBad, "label"},
	}

	for _, chk := range checks {
		for _, output := range outputs {
			got := validateJSONStringFields(ctx, output, params, chk.check, chk.label)
			want := validateJSONStringFieldsLegacy(ctx, output, params, chk.check, chk.label)
			if !phaseAResultsEqual(got, want) {
				t.Errorf("%s output %q: got %+v want %+v", chk.name, output, got, want)
			}
		}
	}
}

func TestValidateJSONStringFields_parityValidators(t *testing.T) {
	ctx := context.Background()
	params := map[string]any{"fields": []any{"token", "version", "color", "ip"}}
	outputs := []string{
		`{"token":"aGVsbG8=","version":"1.2.3","color":"#abc123","ip":"10.0.0.1"}`,
		`{"token":"!!!","version":"v1","color":"red","ip":"999"}`,
		`{"token":1,"version":"1.2.3","color":"#abc","ip":"10.0.0.1"}`,
		`{bad`,
	}
	validators := []Validator{
		ValidBase64{},
		ValidJWTStructure{},
		ValidSemver{},
		ValidColorHex{},
		ValidIPv4OrIPv6{},
	}

	for _, v := range validators {
		for _, output := range outputs {
			got := v.Run(ctx, output, params)
			want := runStringFieldValidatorLegacy(ctx, output, params, v.Name())
			if !phaseAResultsEqual(got, want) {
				t.Errorf("%s output %q: got %+v want %+v", v.Name(), output, got, want)
			}
		}
	}
}

func runStringFieldValidatorLegacy(ctx context.Context, output string, params map[string]any, name string) Result {
	switch name {
	case "valid_base64":
		return validateJSONStringFieldsLegacy(ctx, output, params, func(s string) bool {
			s = strings.TrimSpace(s)
			if s == "" {
				return false
			}
			_, err := base64.StdEncoding.DecodeString(s)
			return err == nil
		}, "invalid base64")
	case "valid_jwt_structure":
		return validateJSONStringFieldsLegacy(ctx, output, params, func(s string) bool {
			return jwtStructRe.MatchString(strings.TrimSpace(s))
		}, "invalid JWT structure")
	case "valid_semver":
		return validateJSONStringFieldsLegacy(ctx, output, params, func(s string) bool {
			return semverRe.MatchString(strings.TrimSpace(s))
		}, "invalid semver")
	case "valid_color_hex":
		return validateJSONStringFieldsLegacy(ctx, output, params, func(s string) bool {
			return colorHexRe.MatchString(strings.TrimSpace(s))
		}, "invalid color hex")
	case "valid_ipv4_or_ipv6":
		return validateJSONStringFieldsLegacy(ctx, output, params, func(s string) bool {
			s = strings.TrimSpace(s)
			if ipv4Re.MatchString(s) {
				return true
			}
			if strings.Contains(s, ":") && ipv6SimpleRe.MatchString(s) {
				return strings.Count(s, ":") >= 2
			}
			return false
		}, "invalid IP address")
	default:
		panic("unknown validator: " + name)
	}
}

func outputShapeJSONLegacyValid(trim string) bool {
	return json.Valid([]byte(trim))
}

func outputShapeJSONNewValid(ctx context.Context, output string) bool {
	return validateJSONFormatCtx(ctx, output) == nil
}

func TestOutputShape_json_parity(t *testing.T) {
	ctx := context.Background()
	outputs := []string{
		`{"a":1}`,
		`  {"a":1}  `,
		`[1,2,3]`,
		`"plain"`,
		`true`,
		`null`,
		`{"nested":{"deep":[1,2,3]}}`,
		`{bad`,
		``,
		`not json at all`,
		`{"a":1.0e+0}`,
		`"with \"quotes\""`,
	}
	for _, output := range outputs {
		trim := strings.TrimSpace(output)
		legacyOK := outputShapeJSONLegacyValid(trim)
		newOK := outputShapeJSONNewValid(ctx, output)
		if legacyOK != newOK {
			t.Errorf("JSON validity mismatch for %q: legacy=%v new=%v", output, legacyOK, newOK)
		}
	}
}

func TestOutputShape_parity(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		output string
		shape  string
	}{
		{`{"a":1}`, "json"},
		{`{bad`, "json"},
		{"- one\n- two", "list"},
		{"plain text", "list"},
		{"| a | b |\n|---|---|", "table"},
		{"no table here", "table"},
		{"# Title\n\n- item", "markdown"},
		{"plain prose", "markdown"},
	}
	for _, tc := range cases {
		params := map[string]any{"shape": tc.shape}
		got := (OutputShape{}).Run(ctx, tc.output, params)
		want := outputShapeLegacy(ctx, tc.output, params)
		if !phaseAResultsEqual(got, want) {
			t.Errorf("shape=%s output %q: got %+v want %+v", tc.shape, tc.output, got, want)
		}
	}
}

func outputShapeLegacy(_ context.Context, output string, params map[string]any) Result {
	shape, _ := params["shape"].(string)
	if shape == "" {
		shape = "json"
	}
	trim := strings.TrimSpace(output)
	switch shape {
	case "json":
		if !json.Valid([]byte(trim)) {
			return failResult("error", "output is not valid JSON", nil)
		}
	case "list":
		if trim == "" {
			return failResult("error", "output is not a list", nil)
		}
		for _, line := range strings.Split(trim, "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			if !strings.HasPrefix(strings.TrimSpace(line), "-") && !strings.HasPrefix(strings.TrimSpace(line), "*") {
				return failResult("error", "output is not a bullet list", nil)
			}
		}
	case "table":
		if !strings.Contains(trim, "|") {
			return failResult("error", "output is not a markdown table", nil)
		}
	case "markdown":
		if !looksLikeMarkdown(trim) {
			return failResult("error", "output is not markdown", nil)
		}
	default:
		return skipResult("unknown shape: " + shape)
	}
	return passResult("error", "shape ok")
}
