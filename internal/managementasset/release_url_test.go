package managementasset

import "testing"

func TestResolveReleaseTagURL(t *testing.T) {
	cases := []struct {
		name string
		repo string
		tag  string
		want string
	}{
		{
			name: "empty tag falls back to latest",
			repo: "https://github.com/router-for-me/Cli-Proxy-API-Management-Center",
			tag:  "",
			want: "https://api.github.com/repos/router-for-me/Cli-Proxy-API-Management-Center/releases/latest",
		},
		{
			name: "github repo with tag",
			repo: "https://github.com/router-for-me/Cli-Proxy-API-Management-Center",
			tag:  "v1.24.2",
			want: "https://api.github.com/repos/router-for-me/Cli-Proxy-API-Management-Center/releases/tags/v1.24.2",
		},
		{
			name: "api endpoint with tag",
			repo: "https://api.github.com/repos/router-for-me/Cli-Proxy-API-Management-Center",
			tag:  "v1.24.2",
			want: "https://api.github.com/repos/router-for-me/Cli-Proxy-API-Management-Center/releases/tags/v1.24.2",
		},
		{
			name: "api releases endpoint is not doubled",
			repo: "https://api.github.com/repos/acme/panel/releases",
			tag:  "v1.0.0",
			want: "https://api.github.com/repos/acme/panel/releases/tags/v1.0.0",
		},
		{
			name: "trailing slash tolerated",
			repo: "https://github.com/acme/panel/",
			tag:  "v2.0.0",
			want: "https://api.github.com/repos/acme/panel/releases/tags/v2.0.0",
		},
		{
			name: "default repo when unset",
			repo: "",
			tag:  "v1.24.2",
			want: "https://api.github.com/repos/router-for-me/Cli-Proxy-API-Management-Center/releases/tags/v1.24.2",
		},
		{
			name: "non-github host falls back to default repo, tag preserved",
			repo: "https://example.com/panel",
			tag:  "v1.24.2",
			want: "https://api.github.com/repos/router-for-me/Cli-Proxy-API-Management-Center/releases/tags/v1.24.2",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveReleaseTagURL(tc.repo, tc.tag); got != tc.want {
				t.Fatalf("resolveReleaseTagURL(%q, %q) = %q, want %q", tc.repo, tc.tag, got, tc.want)
			}
		})
	}
}
