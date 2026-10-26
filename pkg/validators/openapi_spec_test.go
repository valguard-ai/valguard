//go:build !osspublic

package validators

import (
	"strings"
	"testing"
)

const sampleOpenAPIJSON = `{
  "openapi": "3.0.3",
  "info": {"title": "Demo", "version": "1.0.0"},
  "paths": {
    "/health": {
      "get": {"responses": {"200": {"description": "ok"}}}
    }
  }
}`

const sampleOpenAPIYAML = `openapi: "3.0.3"
info:
  title: Demo
  version: "1.0.0"
paths:
  /health:
    get:
      responses:
        "200":
          description: ok
`

func TestValidateOpenAPISpec_JSONPass(t *testing.T) {
	r := validateOpenAPISpec(sampleOpenAPIJSON, "")
	if r.Status != "pass" {
		t.Fatalf("expected pass, got %s: %s", r.Status, r.Message)
	}
}

func TestValidateOpenAPISpec_YAMLPPass(t *testing.T) {
	r := validateOpenAPISpec(sampleOpenAPIYAML, "")
	if r.Status != "pass" {
		t.Fatalf("expected pass, got %s: %s", r.Status, r.Message)
	}
}

func TestValidateOpenAPISpec_MissingPaths(t *testing.T) {
	r := validateOpenAPISpec(`{"openapi":"3.0.3","info":{"title":"x","version":"1"}}`, "")
	if r.Status != "fail" || !strings.Contains(r.Message, "paths") {
		t.Fatalf("expected paths fail, got %+v", r)
	}
}

func TestValidateOpenAPISpec_InvalidPathsObject(t *testing.T) {
	r := validateOpenAPISpec(`{"openapi":"3.0.3","paths":{"not_a_path":"garbage"}}`, "")
	if r.Status != "fail" || !strings.Contains(r.Message, "paths") {
		t.Fatalf("expected paths fail for non-operation paths, got %+v", r)
	}
	r = validateOpenAPISpec(`{"openapi":"3.0.3","paths":{"/health":{"parameters":[]}}}`, "")
	if r.Status != "fail" || !strings.Contains(r.Message, "paths") {
		t.Fatalf("expected paths fail when no HTTP methods defined, got %+v", r)
	}
}

func TestValidateOpenAPISpec_CompileHotPath(t *testing.T) {
	run := ValidOpenAPISpec{}.Compile(map[string]any{"version": "3.0"})
	r := run(t.Context(), sampleOpenAPIJSON)
	if r.Status != "pass" {
		t.Fatalf("compiled runner: %s", r.Message)
	}
}

func BenchmarkValidateOpenAPISpec_JSON(b *testing.B) {
	v := ValidOpenAPISpec{}
	params := map[string]any{}
	ctx := b.Context()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.Run(ctx, sampleOpenAPIJSON, params)
	}
}

func BenchmarkValidateOpenAPISpec_JSONCompiled(b *testing.B) {
	run := ValidOpenAPISpec{}.Compile(map[string]any{})
	ctx := b.Context()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = run(ctx, sampleOpenAPIJSON)
	}
}
