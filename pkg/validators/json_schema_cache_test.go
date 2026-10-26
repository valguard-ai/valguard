package validators

import "testing"

func TestJSONSchemaCompileCache_sharedRunner(t *testing.T) {
	v := JSONSchemaValidate{}
	schema := map[string]any{
		"type":     "object",
		"required": []any{"id"},
	}
	params := map[string]any{"schema": schema}
	r1, ok := TryCompile(v, params)
	if !ok {
		t.Fatal("compile failed")
	}
	r2, ok := TryCompile(v, params)
	if !ok {
		t.Fatal("second compile failed")
	}
	h := jsonSchemaHash(schema)
	if h == "" {
		t.Fatal("expected schema hash")
	}
	if !jsonSchemaCompileCache.contains(h) {
		t.Fatal("expected cached runner")
	}
	out := `{"id":"x"}`
	if r1(nil, out).Status != "pass" || r2(nil, out).Status != "pass" {
		t.Fatal("runners should validate")
	}
}

func TestJSONSchemaCompileCache_deepCopyIsolatedFromMutation(t *testing.T) {
	v := JSONSchemaValidate{}
	schema := map[string]any{
		"type":     "object",
		"required": []any{"id"},
	}
	params := map[string]any{"schema": schema}
	runner, ok := TryCompile(v, params)
	if !ok {
		t.Fatal("compile failed")
	}

	schema["required"] = []any{"other"}

	if r := runner(nil, `{"id":"x"}`); r.Status != "pass" {
		t.Fatalf("cached runner should still require original schema, got %s (%s)", r.Status, r.Message)
	}
	if r := runner(nil, `{}`); r.Status != "fail" {
		t.Fatalf("cached runner should still fail missing id, got %s", r.Status)
	}
}

func TestJSONSchemaCompileCache_evictsOldestWhenFull(t *testing.T) {
	cache := newJSONSchemaLRUCache(2)
	schemaA := map[string]any{"type": "object", "title": "a"}
	schemaB := map[string]any{"type": "object", "title": "b"}
	schemaC := map[string]any{"type": "object", "title": "c"}
	hashA := jsonSchemaHash(schemaA)
	hashB := jsonSchemaHash(schemaB)
	hashC := jsonSchemaHash(schemaC)

	cache.put(hashA, compileJSONSchemaRunner(schemaA))
	cache.put(hashB, compileJSONSchemaRunner(schemaB))
	if cache.len() != 2 {
		t.Fatalf("expected 2 entries, got %d", cache.len())
	}

	cache.put(hashC, compileJSONSchemaRunner(schemaC))
	if cache.len() != 2 {
		t.Fatalf("expected cache size 2 after eviction, got %d", cache.len())
	}
	if cache.contains(hashA) {
		t.Fatal("expected oldest entry to be evicted")
	}
	if !cache.contains(hashB) || !cache.contains(hashC) {
		t.Fatal("expected newer entries to remain cached")
	}

	cache.get(hashB)
	cache.put(hashA, compileJSONSchemaRunner(schemaA))
	if cache.contains(hashC) {
		t.Fatal("expected LRU eviction to drop least-recently-used entry")
	}
	if !cache.contains(hashA) || !cache.contains(hashB) {
		t.Fatal("expected recently used entries to remain cached")
	}
}
