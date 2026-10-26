package validators

import (
	"context"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

type ValidFormat struct{}

func (ValidFormat) Name() string { return "valid_format" }

var validFormatPassMsgs = map[string]string{
	"json": "valid json",
	"yaml": "valid yaml",
	"xml":  "valid xml",
	"toml": "valid toml",
	"csv":  "valid csv",
}

func (ValidFormat) Run(ctx context.Context, output string, params map[string]any) Result {
	format := formatParam(params)
	output = strings.TrimSpace(output)
	if output == "" {
		return emptyFormatResult(ctx, format)
	}
	if err := validateFormatCtx(ctx, output, format); err != nil {
		return failResult("error", fmt.Sprintf("output is not valid %s", format), map[string]any{
			"format": format,
			"error":  err.Error(),
		})
	}
	msg, ok := validFormatPassMsgs[format]
	if !ok {
		msg = "valid json"
	}
	return passResult("error", msg)
}

func (ValidFormat) Compile(params map[string]any) CompiledRunner {
	format := formatParam(params)
	msg, ok := validFormatPassMsgs[format]
	if !ok {
		msg = "valid json"
	}
	return func(ctx context.Context, output string) Result {
		output = strings.TrimSpace(output)
		if output == "" {
			return emptyFormatResult(ctx, format)
		}
		if err := validateFormatCtx(ctx, output, format); err != nil {
			return failResult("error", fmt.Sprintf("output is not valid %s", format), map[string]any{
				"format": format,
				"error":  err.Error(),
			})
		}
		return passResult("error", msg)
	}
}

func emptyFormatResult(ctx context.Context, format string) Result {
	details := map[string]any{"format": format}
	reason := strings.TrimSpace(RunMetaFrom(ctx).FinishReason)
	if reason == "tool_calls" || reason == "function_call" {
		return passResult("error", "tool call response (no text content)")
	}
	msg := "output is empty"
	if reason != "" {
		details["finish_reason"] = reason
		msg = "output is empty (upstream finish_reason: " + reason + ")"
	}
	return failResult("error", msg, details)
}

func formatParam(params map[string]any) string {
	if params == nil {
		return "json"
	}
	f, _ := params["format"].(string)
	f = strings.ToLower(strings.TrimSpace(f))
	switch f {
	case "json", "yaml", "yml", "xml", "toml", "csv":
		if f == "yml" {
			return "yaml"
		}
		return f
	default:
		return "json"
	}
}

func validateFormatCtx(ctx context.Context, output, format string) error {
	switch format {
	case "json":
		return validateJSONFormatCtx(ctx, output)
	case "yaml":
		var v any
		return yaml.Unmarshal([]byte(output), &v)
	case "xml":
		return validateXML(output)
	case "toml":
		var v any
		return toml.Unmarshal([]byte(output), &v)
	case "csv":
		return validateCSV(output)
	default:
		return validateJSONFormatCtx(ctx, output)
	}
}

// ValidateFormat checks output against a format name (json, yaml, …).
func ValidateFormat(output, format string) error {
	return validateFormatCtx(context.Background(), output, format)
}

func validateFormat(output, format string) error {
	return ValidateFormat(output, format)
}

// ValidateXML checks well-formed XML.
func ValidateXML(output string) error {
	return validateXML(output)
}

func validateXML(output string) error {
	dec := xml.NewDecoder(strings.NewReader(output))
	for {
		_, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func validateCSV(output string) error {
	r := csv.NewReader(strings.NewReader(output))
	r.TrimLeadingSpace = true
	records, err := r.ReadAll()
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return fmt.Errorf("no CSV rows")
	}
	for i, row := range records {
		if len(row) == 0 {
			return fmt.Errorf("empty row at index %d", i)
		}
	}
	return nil
}
