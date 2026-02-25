package lsp

import "testing"

func TestAllowedBlocksForDirective(t *testing.T) {
	tests := []struct {
		name string
		dir  string
		want []string
	}{
		{name: "upstream config", dir: "upstream_config", want: []string{"defaults"}},
		{name: "upstream", dir: "upstream", want: []string{"match"}},
		{name: "auth", dir: "oauth_mode", want: []string{"auth"}},
		{name: "request", dir: "req_map", want: []string{"request"}},
		{name: "multi blocks", dir: "set_header", want: []string{"request", "balance", "models"}},
		{name: "response", dir: "resp_map", want: []string{"response"}},
		{name: "error", dir: "error_map", want: []string{"error"}},
		{name: "metrics", dir: "usage_extract", want: []string{"metrics"}},
		{name: "upstream block", dir: "set_path", want: []string{"upstream"}},
		{name: "base url", dir: "base_url", want: []string{"upstream_config"}},
		{name: "block keyword", dir: "provider", want: nil},
		{name: "unknown", dir: "not_exists", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := allowedBlocksForDirective(tt.dir)
			if len(got) != len(tt.want) {
				t.Fatalf("len mismatch: got=%v want=%v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("item mismatch at %d: got=%v want=%v", i, got, tt.want)
				}
			}
		})
	}
}
