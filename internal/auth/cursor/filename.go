package cursor

import (
	"fmt"
	"strings"
)

// CredentialFileName returns the filename used to persist Cursor credentials.
// Priority: explicit label > auto-generated from JWT sub hash.
// If both label and subHash are empty, falls back to "cursor.json".
func CredentialFileName(label, subHash string) string {
	label = safeFileNameSegment(label)
	subHash = safeFileNameSegment(subHash)
	if label != "" {
		return fmt.Sprintf("cursor.%s.json", label)
	}
	if subHash != "" {
		return fmt.Sprintf("cursor.%s.json", subHash)
	}
	return "cursor.json"
}

// safeFileNameSegment returns value when it is safe to embed in a credential
// filename, and "" otherwise. The label comes straight from the `?label=` query
// string, and the token store resolves the resulting filename with
// filepath.Join, which cleans ".." segments — so a label carrying path
// separators or a leading dot would place the credential file outside the auth
// directory. Callers fall back to the auto-generated sub hash when a label is
// rejected, so an accepted label keeps its exact filename.
func safeFileNameSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, `/\`) || strings.HasPrefix(value, ".") {
		return ""
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return ""
		}
	}
	return value
}

// DisplayLabel returns a human-readable label for the Cursor account.
func DisplayLabel(label, subHash string) string {
	label = strings.TrimSpace(label)
	if label != "" {
		return "Cursor " + label
	}
	if subHash != "" {
		return "Cursor " + subHash
	}
	return "Cursor User"
}
