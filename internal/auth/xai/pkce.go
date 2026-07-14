package xai

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// PKCECodes holds PKCE verification codes for OAuth2 PKCE flow.
type PKCECodes struct {
	// CodeVerifier is the cryptographically random string used to correlate
	// the authorization request to the token request.
	CodeVerifier string `json:"code_verifier"`

	// CodeChallenge is the SHA256 hash of the code verifier, base64url encoded.
	CodeChallenge string `json:"code_challenge"`
}

// GeneratePKCECodes creates a verifier/challenge pair for the OAuth flow.
func GeneratePKCECodes() (*PKCECodes, error) {
	bytes := make([]byte, 96)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("xai pkce: generate verifier: %w", err)
	}
	verifier := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes)
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:])
	return &PKCECodes{CodeVerifier: verifier, CodeChallenge: challenge}, nil
}
