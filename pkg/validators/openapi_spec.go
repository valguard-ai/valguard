//go:build !osspublic

package validators

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	gojson "github.com/goccy/go-json"
	"gopkg.in/yaml.v3"
)

var (
	openAPIYAMLVersionRe = regexp.MustCompile(`(?m)^openapi:\s*["']?([0-9.]+)`)
	openAPIYAMLSwaggerRe = regexp.MustCompile(`(?m)^swagger:\s*["']?([0-9.]+)`)
	openAPIHTTPMethods   = map[string]struct{}{
		"get": {}, "put": {}, "post": {}, "delete": {},
		"options": {}, "head": {}, "patch": {}, "trace": {},
	}
)

type openAPIShallowJSON struct {
	OpenAPI string          `json:"openapi"`
	Swagger string          `json:"swagger"`
	Paths   json.RawMessage `json:"paths"`
}

func validateOpenAPISpec(content, expectVersion string) Result {
	content = strings.TrimSpace(content)
	if content == "" {
		return failResult("error", "OpenAPI spec is empty", nil)
	}

	var openapiVer, swaggerVer string
	var hasPaths bool

	if strings.HasPrefix(content, "{") {
		var doc openAPIShallowJSON
		if err := gojson.Unmarshal([]byte(content), &doc); err != nil {
		 return failResult("error", "invalid JSON OpenAPI", map[string]any{"error": err.Error()})
		}
		openapiVer = doc.OpenAPI
		swaggerVer = doc.Swagger
		hasPaths = openAPIPathsJSONPresent(doc.Paths)
	} else {
		var err error
		openapiVer, swaggerVer, hasPaths, err = openAPIShallowYAML(content)
		if err != nil {
			return failResult("error", "invalid YAML OpenAPI", map[string]any{"error": err.Error()})
		}
	}

	if openapiVer == "" && swaggerVer == "" {
		return failResult("error", "missing openapi or swagger version field", nil)
	}
	if !hasPaths {
		return failResult("error", "OpenAPI spec must define paths", nil)
	}
	if expectVersion != "" {
		ver := openapiVer
		if ver == "" {
			ver = swaggerVer
		}
		if ver != "" && !openAPIVersionMatches(expectVersion, ver) {
			return failResult("error", "openapi version mismatch", map[string]any{"expected": expectVersion, "got": ver})
		}
	}
	return passResult("error", "OpenAPI spec looks valid")
}

func openAPIPathsJSONPresent(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	switch strings.TrimSpace(string(raw)) {
	case "", "null", "{}":
		return false
	}
	var paths map[string]map[string]any
	if err := json.Unmarshal(raw, &paths); err != nil || len(paths) == 0 {
		return false
	}
	for _, pathItem := range paths {
		if openAPIPathItemHasOperation(pathItem) {
			return true
		}
	}
	return false
}

func openAPIPathItemHasOperation(pathItem map[string]any) bool {
	for method := range pathItem {
		if _, ok := openAPIHTTPMethods[strings.ToLower(method)]; ok {
			return true
		}
	}
	return false
}

func openAPIShallowYAML(content string) (openapiVer, swaggerVer string, hasPaths bool, err error) {
	if m := openAPIYAMLVersionRe.FindStringSubmatch(content); len(m) > 1 {
		openapiVer = m[1]
	}
	if m := openAPIYAMLSwaggerRe.FindStringSubmatch(content); len(m) > 1 {
		swaggerVer = m[1]
	}

	var doc struct {
		OpenAPI string                            `yaml:"openapi"`
		Swagger string                            `yaml:"swagger"`
		Paths   map[string]map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return "", "", false, err
	}
	if openapiVer == "" {
		openapiVer = doc.OpenAPI
	}
	if swaggerVer == "" {
		swaggerVer = doc.Swagger
	}
	for _, pathItem := range doc.Paths {
		if openAPIPathItemHasOperation(pathItem) {
			hasPaths = true
			break
		}
	}
	return openapiVer, swaggerVer, hasPaths, nil
}

func openAPIVersionMatches(expected, got string) bool {
	expected = strings.TrimSpace(expected)
	got = strings.TrimSpace(got)
	if expected == "" || got == "" {
		return true
	}
	if got == expected {
		return true
	}
	if strings.HasPrefix(got, expected) {
		return true
	}
	// loose match for params like "3.0" vs "3.0.3"
	if strings.HasPrefix(expected, "3.") && strings.HasPrefix(got, "3.") {
		return true
	}
	return strings.HasPrefix(got, strings.TrimPrefix(expected, "3."))
}

func (ValidOpenAPISpec) Compile(params map[string]any) CompiledRunner {
	expectVersion, _ := params["version"].(string)
	return func(_ context.Context, output string) Result {
		return validateOpenAPISpec(output, expectVersion)
	}
}
