package auth

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system.
type User struct {
	ID                 uuid.UUID
	ShortID            string
	Email              string
	PasswordHash       string
	Name               string
	Status             string
	Roles              string
	MustChangePassword bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

const (
	RoleAdmin  = "admin"
	RoleEditor = "editor"
	RoleViewer = "viewer"
)

// NewUser creates a new user with the given email and password.
func NewUser(email, password, name string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &User{
		ID:           uuid.New(),
		ShortID:      uuid.New().String()[:8],
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
		Status:       "active",
		Roles:        RoleEditor,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// CheckPassword verifies if the provided password matches the stored hash.
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// UpdatePassword updates the user's password hash.
func (u *User) UpdatePassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	u.UpdatedAt = time.Now()
	return nil
}

// IsActive returns true if the user status is "active".
func (u *User) IsActive() bool {
	return u.Status == "active"
}

// HasRole returns true if the user has the specified role.
func (u *User) HasRole(role string) bool {
	for _, r := range strings.Split(u.Roles, ",") {
		if strings.TrimSpace(r) == role {
			return true
		}
	}
	return false
}

// IsAdmin returns true if the user has admin role.
func (u *User) IsAdmin() bool {
	return u.HasRole(RoleAdmin)
}

// IsEditor returns true if the user has editor or admin role.
func (u *User) IsEditor() bool {
	return u.HasRole(RoleEditor) || u.HasRole(RoleAdmin)
}

// IsViewer returns true if the user has viewer, editor, or admin role.
func (u *User) IsViewer() bool {
	return u.HasRole(RoleViewer) || u.HasRole(RoleEditor) || u.HasRole(RoleAdmin)
}

// RolesList returns the roles as a slice.
func (u *User) RolesList() []string {
	if u.Roles == "" {
		return nil
	}
	var roles []string
	for _, r := range strings.Split(u.Roles, ",") {
		if trimmed := strings.TrimSpace(r); trimmed != "" {
			roles = append(roles, trimmed)
		}
	}
	return roles
}

// Session represents a user session.
type Session struct {
	ID        string
	UserID    uuid.UUID
	ExpiresAt time.Time
	CreatedAt time.Time
}

// NewSession creates a new session for the given user ID with the specified TTL.
func NewSession(userID uuid.UUID, ttl time.Duration) *Session {
	now := time.Now()
	return &Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
	}
}

// IsExpired returns true if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsValid returns true if the session is not expired.
func (s *Session) IsValid() bool {
	return !s.IsExpired()
}
