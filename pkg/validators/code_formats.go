//go:build !osspublic

package validators

import (
	"context"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

type XMLValidity struct{}

func (XMLValidity) Name() string { return "xml_validity" }

func (XMLValidity) Run(ctx context.Context, output string, _ map[string]any) Result {
	if err := validateXML(strings.TrimSpace(output)); err != nil {
		return failResult("error", "invalid XML", map[string]any{"error": err.Error()})
	}
	return passResult("error", "valid XML")
}

type XMLXSDSchema struct{}

func (XMLXSDSchema) Name() string { return "xml_xsd_schema" }

func (XMLXSDSchema) Run(ctx context.Context, output string, params map[string]any) Result {
	content := strings.TrimSpace(output)
	if content == "" {
		return failResult("error", "XML is empty", nil)
	}
	if err := validateXML(content); err != nil {
		return failResult("error", "XML not well-formed for XSD check", map[string]any{"error": err.Error()})
	}
	schemaPath, _ := params["schema_path"].(string)
	if strings.TrimSpace(schemaPath) == "" {
		return passResult("error", "XML well-formed (no schema_path configured for full XSD validation)")
	}
	return passResult("error", "XML well-formed (XSD file validation requires schema host; structural check only)")
}

type XMLXPathRequired struct{}

func (XMLXPathRequired) Name() string { return "xml_xpath_required" }

func (XMLXPathRequired) Run(ctx context.Context, output string, params map[string]any) Result {
	content := strings.TrimSpace(output)
	if content == "" {
		return failResult("error", "XML is empty", nil)
	}
	paths := stringSliceParam(params["xpaths"])
	if len(paths) == 0 {
		return skipResult("no xpaths configured")
	}
	var missing []string
	for _, xp := range paths {
		tag := xpathToTagHint(xp)
		if tag != "" && !strings.Contains(content, "<"+tag) {
			missing = append(missing, xp)
		}
	}
	if len(missing) > 0 {
		return failResult("error", "required XML elements not found", map[string]any{"missing": missing})
	}
	return passResult("error", "required XML paths present")
}

func xpathToTagHint(xp string) string {
	xp = strings.TrimSpace(xp)
	if xp == "" {
		return ""
	}
	parts := strings.Split(xp, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		p := strings.TrimSpace(parts[i])
		if p == "" || p == "." || p == ".." {
			continue
		}
		if idx := strings.Index(p, "["); idx > 0 {
			p = p[:idx]
		}
		if idx := strings.Index(p, "@"); idx == 0 {
			continue
		}
		return p
	}
	return ""
}

type YAMLValidity struct{}

func (YAMLValidity) Name() string { return "yaml_validity" }

func (YAMLValidity) Run(ctx context.Context, output string, _ map[string]any) Result {
	if err := validateFormat(strings.TrimSpace(output), "yaml"); err != nil {
		return failResult("error", "invalid YAML", map[string]any{"error": err.Error()})
	}
	return passResult("error", "valid YAML")
}

type CSVValidity struct{}

func (CSVValidity) Name() string { return "csv_validity" }

func (CSVValidity) Run(ctx context.Context, output string, params map[string]any) Result {
	content := strings.TrimSpace(output)
	if content == "" {
		return failResult("error", "CSV is empty", nil)
	}
	delimiter := ','
	if d, ok := params["delimiter"].(string); ok && len(d) > 0 {
		delimiter = rune(d[0])
	}
	minCols := 1
	if v, ok := params["min_columns"].(float64); ok && v > 0 {
		minCols = int(v)
	}
	r := csv.NewReader(strings.NewReader(content))
	r.Comma = delimiter
	records, err := r.ReadAll()
	if err != nil {
		return failResult("error", "invalid CSV", map[string]any{"error": err.Error()})
	}
	if len(records) == 0 {
		return failResult("error", "CSV has no rows", nil)
	}
	cols := len(records[0])
	if cols < minCols {
		return failResult("error", "CSV has too few columns", map[string]any{"columns": cols, "min": minCols})
	}
	for i, row := range records {
		if len(row) != cols {
			return failResult("error", fmt.Sprintf("inconsistent column count at row %d", i+1), map[string]any{"expected": cols, "got": len(row)})
		}
	}
	return passResult("error", "valid CSV")
}

type TOMLValidity struct{}

func (TOMLValidity) Name() string { return "toml_validity" }

func (TOMLValidity) Run(ctx context.Context, output string, _ map[string]any) Result {
	if err := validateFormat(strings.TrimSpace(output), "toml"); err != nil {
		return failResult("error", "invalid TOML", map[string]any{"error": err.Error()})
	}
	return passResult("error", "valid TOML")
}

func xmlContainsPath(content, path string) bool {
	tag := xpathToTagHint(path)
	if tag == "" {
		return strings.Contains(content, path)
	}
	dec := xml.NewDecoder(strings.NewReader(content))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return false
		}
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == tag {
			return true
		}
	}
	return false
}
