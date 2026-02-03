package api

import (
	"github.com/cliossg/clio/internal/db/sqlc"
	"github.com/google/uuid"
)

func apiTokenFromSQLC(t sqlc.ApiToken) *APIToken {
	token := &APIToken{
		ID:        parseUUID(t.ID),
		UserID:    parseUUID(t.UserID),
		Name:      t.Name,
		TokenHash: t.TokenHash,
		CreatedAt: t.CreatedAt,
	}
	if t.LastUsedAt.Valid {
		token.LastUsedAt = &t.LastUsedAt.Time
	}
	if t.ExpiresAt.Valid {
		token.ExpiresAt = &t.ExpiresAt.Time
	}
	return token
}

func parseUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}
