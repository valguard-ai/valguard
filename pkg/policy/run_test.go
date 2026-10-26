package policy

import "testing"

func TestRun_shadowWouldEnforce(t *testing.T) {
	doc := Document{
		Version: 1,
		Name:    "shadow",
		Mode:    "shadow",
		Rules: []Rule{
			{
				ID:     "forbidden_patterns",
				Params: map[string]any{"patterns": []any{"SECRET"}},
				OnFail: "block",
			},
		},
	}
	res, err := Run(doc, "leaked SECRET value", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Blocked {
		t.Fatal("shadow must not block")
	}
	if !res.WouldEnforce {
		t.Fatal("expected would enforce in shadow")
	}
	if !res.Shadow {
		t.Fatal("expected shadow flag")
	}
}
