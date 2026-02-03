package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// APIToken represents an API authentication token.
type APIToken struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	Name       string     `json:"name"`
	TokenHash  string     `json:"-"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// NewAPIToken creates a new APIToken and returns the raw token string.
// The raw token is only available at creation time; only the hash is stored.
func NewAPIToken(userID uuid.UUID, name string) (rawToken string, token *APIToken, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, fmt.Errorf("cannot generate token: %w", err)
	}

	rawToken = base64.RawURLEncoding.EncodeToString(b)
	hash := HashToken(rawToken)

	token = &APIToken{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      name,
		TokenHash: hash,
		CreatedAt: time.Now(),
	}

	return rawToken, token, nil
}

// HashToken returns the SHA-256 hash of a raw token string.
func HashToken(rawToken string) string {
	h := sha256.Sum256([]byte(rawToken))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
