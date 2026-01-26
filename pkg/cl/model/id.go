package model

import (
	"github.com/google/uuid"
)

// NewID generates a new UUID.
func NewID() uuid.UUID {
	return uuid.New()
}

// ParseID parses a string into a UUID.
// Returns uuid.Nil if parsing fails.
func ParseID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}

// IsValidID checks if a UUID is valid (not nil).
func IsValidID(id uuid.UUID) bool {
	return id != uuid.Nil
}
