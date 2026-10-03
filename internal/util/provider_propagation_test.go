package util

import "testing"

func TestFilterProvidersByPropagation(t *testing.T) {
	cases := []struct {
		name      string
		providers []string
		allowlist map[string]bool
		want      []string
	}{
		{"nil allowlist keeps all", []string{"kiro", "github-copilot"}, nil, []string{"kiro", "github-copilot"}},
		{"empty allowlist keeps all", []string{"kiro", "github-copilot"}, map[string]bool{}, []string{"kiro", "github-copilot"}},
		{"all false keeps all", []string{"kiro"}, map[string]bool{"kiro": false}, []string{"kiro"}},
		{"drops unlisted", []string{"kiro", "github-copilot"}, map[string]bool{"kiro": true}, []string{"kiro"}},
		{"case insensitive", []string{"Kiro", "GitHub-Copilot"}, map[string]bool{"kiro": true, "github-copilot": true}, []string{"Kiro", "GitHub-Copilot"}},
		{"trims whitespace", []string{"kiro"}, map[string]bool{"  kiro  ": true}, []string{"kiro"}},
		{"preserves order", []string{"a", "b", "c"}, map[string]bool{"c": true, "a": true}, []string{"a", "c"}},
		{"everything dropped", []string{"github-copilot"}, map[string]bool{"kiro": true}, []string{}},
		{"nil providers", nil, map[string]bool{"kiro": true}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FilterProvidersByPropagation(tc.providers, tc.allowlist)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}
