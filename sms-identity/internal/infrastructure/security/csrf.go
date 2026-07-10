package security

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"time"
)

const (
	csrfTokenLength = 32
	csrfCookieName  = "csrf_token"
	csrfHeaderName  = "X-CSRF-Token"
	csrfTTL         = 24 * time.Hour
)

type CSRFManager struct{}

func NewCSRFManager() *CSRFManager {
	return &CSRFManager{}
}

// GenerateCSRFToken creates a cryptographically random CSRF token.
func (m *CSRFManager) GenerateCSRFToken() (string, error) {
	b := make([]byte, csrfTokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ValidateCSRFToken compares two CSRF tokens in constant time to prevent timing attacks.
// The cookieValue is the raw token from the csrf_token cookie.
// The headerValue is the raw token from the X-CSRF-Token header.
func (m *CSRFManager) ValidateCSRFToken(cookieValue, headerValue string) error {
	if cookieValue == "" || headerValue == "" {
		return errors.New("missing CSRF token")
	}

	cookieHash := sha256.Sum256([]byte(cookieValue))
	headerHash := sha256.Sum256([]byte(headerValue))

	if subtle.ConstantTimeCompare(cookieHash[:], headerHash[:]) != 1 {
		return errors.New("CSRF token mismatch")
	}

	return nil
}

func CSRFCookieName() string { return csrfCookieName }
func CSRFHeaderName() string { return csrfHeaderName }
func CSRFTTL() time.Duration { return csrfTTL }