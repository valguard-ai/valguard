package engine

import "testing"

func TestCanPassthroughStream(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		insts      []Instance
		shadowMode bool
		want       bool
	}{
		{
			name: "warn only",
			insts: []Instance{
				{Enabled: true, OnFail: "warn"},
				{Enabled: true, OnFail: "pass"},
			},
			want: true,
		},
		{
			name: "log allows passthrough",
			insts: []Instance{
				{Enabled: true, OnFail: "log"},
			},
			want: true,
		},
		{
			name: "block uses buffered path",
			insts: []Instance{
				{Enabled: true, OnFail: "warn"},
				{Enabled: true, OnFail: "block"},
			},
			want: false,
		},
		{
			name: "reask disables",
			insts: []Instance{
				{Enabled: true, OnFail: "reask"},
			},
			want: false,
		},
		{
			name: "disabled ignored",
			insts: []Instance{
				{Enabled: false, OnFail: "block"},
				{Enabled: true, OnFail: "warn"},
			},
			want: true,
		},
		{
			name:  "empty",
			insts: nil,
			want:  true,
		},
		{
			name:  "unknown_on_fail_allows_passthrough",
			insts: []Instance{{Enabled: true, OnFail: "other"}},
			want:  true,
		},
		{
			name: "shadow overrides reask",
			insts: []Instance{
				{Enabled: true, OnFail: "reask"},
			},
			shadowMode: true,
			want:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanPassthroughStream(tt.insts, tt.shadowMode); got != tt.want {
				t.Fatalf("CanPassthroughStream() = %v, want %v", got, tt.want)
			}
		})
	}
}
