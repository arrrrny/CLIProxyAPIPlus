package cursor

import (
	"path/filepath"
	"testing"
)

func TestCredentialFileName(t *testing.T) {
	tests := []struct {
		name    string
		label   string
		subHash string
		want    string
	}{
		{
			name:    "safe label wins over the sub hash",
			label:   "work",
			subHash: "abc12345",
			want:    "cursor.work.json",
		},
		{
			name:  "dashes and underscores are preserved verbatim",
			label: "work-2_env",
			want:  "cursor.work-2_env.json",
		},
		{
			name:    "empty label falls back to the sub hash",
			label:   "",
			subHash: "abc12345",
			want:    "cursor.abc12345.json",
		},
		{
			name: "missing label and sub hash fall back to cursor.json",
			want: "cursor.json",
		},
		{
			name:    "surrounding whitespace is trimmed",
			label:   "  work  ",
			subHash: "abc12345",
			want:    "cursor.work.json",
		},
		{
			name:    "dot-prefixed label is rejected",
			label:   ".hidden",
			subHash: "abc12345",
			want:    "cursor.abc12345.json",
		},
		{
			name:    "dot-only label is rejected",
			label:   "..",
			subHash: "abc12345",
			want:    "cursor.abc12345.json",
		},
		{
			name:    "parent traversal label is rejected",
			label:   "../../auths/evil",
			subHash: "abc12345",
			want:    "cursor.abc12345.json",
		},
		{
			name:    "backslash traversal label is rejected",
			label:   `..\..\auths\evil`,
			subHash: "abc12345",
			want:    "cursor.abc12345.json",
		},
		{
			name:    "absolute path label is rejected",
			label:   "/etc/passwd",
			subHash: "abc12345",
			want:    "cursor.abc12345.json",
		},
		{
			name:  "rejected label without a sub hash falls back to cursor.json",
			label: "../../auths/evil",
			want:  "cursor.json",
		},
		{
			name:  "control characters are rejected",
			label: "work\naccount",
			want:  "cursor.json",
		},
		{
			name:    "sub hash traversal is rejected too",
			label:   "",
			subHash: "../../evil",
			want:    "cursor.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CredentialFileName(tt.label, tt.subHash); got != tt.want {
				t.Fatalf("CredentialFileName(%q, %q) = %q, want %q", tt.label, tt.subHash, got, tt.want)
			}
		})
	}
}

// TestCredentialFileNameNeverEscapesAuthDir pins the hardening: whatever the
// `?label=` query string carries, the resolved path stays inside the auth
// directory.
func TestCredentialFileNameNeverEscapesAuthDir(t *testing.T) {
	const authDir = "/auths"

	labels := []string{
		"../../auths/evil",
		`..\..\auths\evil`,
		"..",
		"../..",
		"/etc/passwd",
		".ssh/id_rsa",
		"a/../../b",
		"....//....//evil",
		"./evil",
	}

	for _, label := range labels {
		t.Run(label, func(t *testing.T) {
			resolved := filepath.Join(authDir, CredentialFileName(label, "abc12345"))
			if got := filepath.Dir(resolved); got != authDir {
				t.Fatalf("CredentialFileName(%q) resolved to %q, outside %q", label, resolved, authDir)
			}
		})
	}
}
