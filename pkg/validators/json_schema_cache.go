package validators

import (
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
)

const defaultJSONSchemaCacheMax = 512

type jsonSchemaCacheEntry struct {
	hash   string
	runner CompiledRunner
}

type jsonSchemaLRUCache struct {
	mu    sync.Mutex
	max   int
	items map[string]*list.Element
	order *list.List
}

func newJSONSchemaLRUCache(max int) *jsonSchemaLRUCache {
	if max <= 0 {
		max = defaultJSONSchemaCacheMax
	}
	return &jsonSchemaLRUCache{
		max:   max,
		items: make(map[string]*list.Element),
		order: list.New(),
	}
}

func (c *jsonSchemaLRUCache) get(hash string) (CompiledRunner, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[hash]
	if !ok {
		return nil, false
	}
	c.order.MoveToFront(el)
	return el.Value.(jsonSchemaCacheEntry).runner, true
}

func (c *jsonSchemaLRUCache) put(hash string, runner CompiledRunner) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[hash]; ok {
		c.order.MoveToFront(el)
		el.Value = jsonSchemaCacheEntry{hash: hash, runner: runner}
		return
	}
	for c.order.Len() >= c.max {
		back := c.order.Back()
		if back == nil {
			break
		}
		ent := back.Value.(jsonSchemaCacheEntry)
		delete(c.items, ent.hash)
		c.order.Remove(back)
	}
	el := c.order.PushFront(jsonSchemaCacheEntry{hash: hash, runner: runner})
	c.items[hash] = el
}

func (c *jsonSchemaLRUCache) contains(hash string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.items[hash]
	return ok
}

func (c *jsonSchemaLRUCache) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len()
}

var jsonSchemaCompileCache = newJSONSchemaLRUCache(defaultJSONSchemaCacheMax)

func jsonSchemaHash(schema map[string]any) string {
	raw, err := json.Marshal(schema)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func compileJSONSchemaRunner(schema map[string]any) CompiledRunner {
	schemaCopy := cloneParamsDeep(schema)
	return func(ctx context.Context, output string) Result {
		return jsonSchemaValidateRunWithSchema(ctx, output, schemaCopy)
	}
}

func cachedJSONSchemaRunner(schema map[string]any) CompiledRunner {
	h := jsonSchemaHash(schema)
	if h == "" {
		return compileJSONSchemaRunner(schema)
	}
	if runner, ok := jsonSchemaCompileCache.get(h); ok {
		return runner
	}
	runner := compileJSONSchemaRunner(schema)
	if existing, ok := jsonSchemaCompileCache.get(h); ok {
		return existing
	}
	jsonSchemaCompileCache.put(h, runner)
	return runner
}
