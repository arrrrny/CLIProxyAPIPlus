package managementasset

import (
	"strings"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
)

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
			name: "bare owner/name",
			repo: "acme/panel",
			tag:  "v2.0.0",
			want: "https://api.github.com/repos/acme/panel/releases/tags/v2.0.0",
		},
		{
			name: "default repo when unset keeps the tag",
			repo: "",
			tag:  "v1.24.2",
			want: "https://api.github.com/repos/router-for-me/Cli-Proxy-API-Management-Center/releases/tags/v1.24.2",
		},
		{
			name: "unrecognized host falls back to default repo and keeps the tag",
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

// The built-in panel repository must never resolve to a release that needs a v8
// management backend, because this build only serves v0.
func TestEffectivePanelReleasePinsBuiltinRepo(t *testing.T) {
	t.Run("builtin repo defaults to the v0-compatible release", func(t *testing.T) {
		got := effectivePanelRelease("", "")
		if got != defaultPanelCompatibleRelease {
			t.Fatalf("got %q, want %q", got, defaultPanelCompatibleRelease)
		}
		if strings.Contains(resolveReleaseTagURL("", got), "/releases/latest") {
			t.Fatal("builtin panel must not resolve to /releases/latest")
		}
	})

	t.Run("explicit release always wins", func(t *testing.T) {
		if got := effectivePanelRelease("", "v9.9.9"); got != "v9.9.9" {
			t.Fatalf("got %q, want v9.9.9", got)
		}
	})

	t.Run("custom repository stays on latest", func(t *testing.T) {
		if got := effectivePanelRelease("https://github.com/acme/panel", ""); got != "" {
			t.Fatalf("got %q, want empty (latest)", got)
		}
	})

	// config loading substitutes DefaultPanelGitHubRepository for an unset key,
	// so the pin must key off the value, not off emptiness.
	t.Run("builtin repo substituted by config loading is pinned", func(t *testing.T) {
		for _, repo := range []string{
			"",
			config.DefaultPanelGitHubRepository,
			"https://github.com/router-for-me/Cli-Proxy-API-Management-Center",
			"https://github.com/router-for-me/Cli-Proxy-API-Management-Center/",
			"https://github.com/router-for-me/Cli-Proxy-API-Management-Center.git",
			"https://api.github.com/repos/router-for-me/Cli-Proxy-API-Management-Center",
		} {
			if got := effectivePanelRelease(repo, ""); got != defaultPanelCompatibleRelease {
				t.Errorf("effectivePanelRelease(%q, %q) = %q, want %q", repo, "", got, defaultPanelCompatibleRelease)
			}
		}
	})
}
