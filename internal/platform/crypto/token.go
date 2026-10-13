package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"
)

// NewToken returns a 256-bit random token, URL-safe. Used for session cookies and share links.
// Store only HashToken(token); the raw token is shown/sent once.
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken is what goes in sessions.token_hash / share_links.token_hash.
func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// Signer makes short-lived, tamper-proof values for cookies that are not backed by a table
// (OAuth state + PKCE verifier, share-link unlock). Values are signed, NOT encrypted:
// never put a secret in them.
type Signer struct{ key []byte }

func NewSigner(secretKey []byte, purpose string) *Signer {
	// derive a separate key per purpose so a value signed for one cookie is useless in another
	m := hmac.New(sha256.New, secretKey)
	m.Write([]byte("netopology:" + purpose))
	return &Signer{key: m.Sum(nil)}
}

var ErrBadSignature = errors.New("invalid or expired signed value")

// Sign returns "<payload b64>.<expiry unix b64>.<mac b64>".
func (s *Signer) Sign(payload []byte, ttl time.Duration) string {
	b64 := base64.RawURLEncoding
	exp := []byte(time.Now().Add(ttl).UTC().Format(time.RFC3339))
	body := b64.EncodeToString(payload) + "." + b64.EncodeToString(exp)
	return body + "." + b64.EncodeToString(s.mac(body))
}

// Verify returns the payload if the signature is valid and not expired.
func (s *Signer) Verify(value string) ([]byte, error) {
	b64 := base64.RawURLEncoding
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return nil, ErrBadSignature
	}
	body := parts[0] + "." + parts[1]
	mac, err := b64.DecodeString(parts[2])
	if err != nil || !hmac.Equal(mac, s.mac(body)) {
		return nil, ErrBadSignature
	}
	expRaw, err := b64.DecodeString(parts[1])
	if err != nil {
		return nil, ErrBadSignature
	}
	exp, err := time.Parse(time.RFC3339, string(expRaw))
	if err != nil || time.Now().After(exp) {
		return nil, ErrBadSignature
	}
	return b64.DecodeString(parts[0])
}

func (s *Signer) mac(body string) []byte {
	m := hmac.New(sha256.New, s.key)
	m.Write([]byte(body))
	return m.Sum(nil)
}
