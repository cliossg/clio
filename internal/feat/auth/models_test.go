package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewUser(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		password    string
		userName    string
		wantErr     bool
		checkFields func(t *testing.T, u *User)
	}{
		{
			name:     "valid user creation",
			email:    "test@example.com",
			password: "securepassword123",
			userName: "testuser",
			wantErr:  false,
			checkFields: func(t *testing.T, u *User) {
				if u.Email != "test@example.com" {
					t.Errorf("Email = %q, want %q", u.Email, "test@example.com")
				}
				if u.Name != "testuser" {
					t.Errorf("Name = %q, want %q", u.Name, "testuser")
				}
				if u.Status != "active" {
					t.Errorf("Status = %q, want %q", u.Status, "active")
				}
				if u.Roles != RoleEditor {
					t.Errorf("Roles = %q, want %q", u.Roles, RoleEditor)
				}
				if u.ID == uuid.Nil {
					t.Error("ID should not be nil")
				}
				if u.ShortID == "" {
					t.Error("ShortID should not be empty")
				}
				if u.PasswordHash == "" {
					t.Error("PasswordHash should not be empty")
				}
				if u.PasswordHash == "securepassword123" {
					t.Error("PasswordHash should be hashed, not plain text")
				}
			},
		},
		{
			name:     "empty password",
			email:    "test@example.com",
			password: "",
			userName: "testuser",
			wantErr:  false, // bcrypt accepts empty passwords
		},
		{
			name:     "long password",
			email:    "test@example.com",
			password: string(make([]byte, 100)),
			userName: "testuser",
			wantErr:  true, // bcrypt has a 72-byte limit
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := NewUser(tt.email, tt.password, tt.userName)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkFields != nil {
				tt.checkFields(t, user)
			}
		})
	}
}

func TestUserCheckPassword(t *testing.T) {
	user, err := NewUser("test@example.com", "correctpassword", "testuser")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{
			name:     "correct password",
			password: "correctpassword",
			want:     true,
		},
		{
			name:     "incorrect password",
			password: "wrongpassword",
			want:     false,
		},
		{
			name:     "empty password",
			password: "",
			want:     false,
		},
		{
			name:     "password with extra characters",
			password: "correctpassword123",
			want:     false,
		},
		{
			name:     "password with different case",
			password: "CorrectPassword",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := user.CheckPassword(tt.password); got != tt.want {
				t.Errorf("User.CheckPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserUpdatePassword(t *testing.T) {
	tests := []struct {
		name        string
		newPassword string
		wantErr     bool
	}{
		{
			name:        "valid new password",
			newPassword: "newpassword123",
			wantErr:     false,
		},
		{
			name:        "empty password",
			newPassword: "",
			wantErr:     false,
		},
		{
			name:        "password too long for bcrypt",
			newPassword: string(make([]byte, 100)),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, _ := NewUser("test@example.com", "oldpassword", "testuser")
			oldHash := user.PasswordHash
			oldUpdatedAt := user.UpdatedAt

			time.Sleep(time.Millisecond)

			err := user.UpdatePassword(tt.newPassword)
			if (err != nil) != tt.wantErr {
				t.Errorf("User.UpdatePassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if user.PasswordHash == oldHash {
					t.Error("PasswordHash should have changed")
				}
				if !user.UpdatedAt.After(oldUpdatedAt) {
					t.Error("UpdatedAt should have been updated")
				}
				if !user.CheckPassword(tt.newPassword) {
					t.Error("New password should be valid")
				}
			}
		})
	}
}

func TestUserIsActive(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{
			name:   "active status",
			status: "active",
			want:   true,
		},
		{
			name:   "inactive status",
			status: "inactive",
			want:   false,
		},
		{
			name:   "suspended status",
			status: "suspended",
			want:   false,
		},
		{
			name:   "empty status",
			status: "",
			want:   false,
		},
		{
			name:   "Active with capital",
			status: "Active",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{Status: tt.status}
			if got := user.IsActive(); got != tt.want {
				t.Errorf("User.IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserHasRole(t *testing.T) {
	tests := []struct {
		name  string
		roles string
		role  string
		want  bool
	}{
		{
			name:  "single matching role",
			roles: "admin",
			role:  "admin",
			want:  true,
		},
		{
			name:  "single non-matching role",
			roles: "editor",
			role:  "admin",
			want:  false,
		},
		{
			name:  "multiple roles - first match",
			roles: "admin,editor,viewer",
			role:  "admin",
			want:  true,
		},
		{
			name:  "multiple roles - middle match",
			roles: "admin,editor,viewer",
			role:  "editor",
			want:  true,
		},
		{
			name:  "multiple roles - last match",
			roles: "admin,editor,viewer",
			role:  "viewer",
			want:  true,
		},
		{
			name:  "multiple roles - no match",
			roles: "admin,editor,viewer",
			role:  "superadmin",
			want:  false,
		},
		{
			name:  "roles with spaces",
			roles: "admin, editor, viewer",
			role:  "editor",
			want:  true,
		},
		{
			name:  "empty roles",
			roles: "",
			role:  "admin",
			want:  false,
		},
		{
			name:  "empty role to check",
			roles: "admin,editor",
			role:  "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{Roles: tt.roles}
			if got := user.HasRole(tt.role); got != tt.want {
				t.Errorf("User.HasRole(%q) = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestUserIsAdmin(t *testing.T) {
	tests := []struct {
		name  string
		roles string
		want  bool
	}{
		{
			name:  "only admin",
			roles: "admin",
			want:  true,
		},
		{
			name:  "admin with others",
			roles: "admin,editor",
			want:  true,
		},
		{
			name:  "only editor",
			roles: "editor",
			want:  false,
		},
		{
			name:  "no roles",
			roles: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{Roles: tt.roles}
			if got := user.IsAdmin(); got != tt.want {
				t.Errorf("User.IsAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserIsEditor(t *testing.T) {
	tests := []struct {
		name  string
		roles string
		want  bool
	}{
		{
			name:  "only editor",
			roles: "editor",
			want:  true,
		},
		{
			name:  "admin counts as editor",
			roles: "admin",
			want:  true,
		},
		{
			name:  "viewer not editor",
			roles: "viewer",
			want:  false,
		},
		{
			name:  "no roles",
			roles: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{Roles: tt.roles}
			if got := user.IsEditor(); got != tt.want {
				t.Errorf("User.IsEditor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserIsViewer(t *testing.T) {
	tests := []struct {
		name  string
		roles string
		want  bool
	}{
		{
			name:  "only viewer",
			roles: "viewer",
			want:  true,
		},
		{
			name:  "editor counts as viewer",
			roles: "editor",
			want:  true,
		},
		{
			name:  "admin counts as viewer",
			roles: "admin",
			want:  true,
		},
		{
			name:  "no roles",
			roles: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{Roles: tt.roles}
			if got := user.IsViewer(); got != tt.want {
				t.Errorf("User.IsViewer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserRolesList(t *testing.T) {
	tests := []struct {
		name  string
		roles string
		want  []string
	}{
		{
			name:  "single role",
			roles: "admin",
			want:  []string{"admin"},
		},
		{
			name:  "multiple roles",
			roles: "admin,editor,viewer",
			want:  []string{"admin", "editor", "viewer"},
		},
		{
			name:  "roles with spaces",
			roles: "admin, editor, viewer",
			want:  []string{"admin", "editor", "viewer"},
		},
		{
			name:  "empty string",
			roles: "",
			want:  nil,
		},
		{
			name:  "only commas",
			roles: ",,,",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{Roles: tt.roles}
			got := user.RolesList()
			if len(got) != len(tt.want) {
				t.Errorf("User.RolesList() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("User.RolesList()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestNewSession(t *testing.T) {
	tests := []struct {
		name   string
		userID uuid.UUID
		ttl    time.Duration
	}{
		{
			name:   "standard session",
			userID: uuid.New(),
			ttl:    24 * time.Hour,
		},
		{
			name:   "short TTL",
			userID: uuid.New(),
			ttl:    time.Minute,
		},
		{
			name:   "zero TTL",
			userID: uuid.New(),
			ttl:    0,
		},
		{
			name:   "nil UUID",
			userID: uuid.Nil,
			ttl:    time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now()
			session := NewSession(tt.userID, tt.ttl)
			after := time.Now()

			if session.UserID != tt.userID {
				t.Errorf("UserID = %v, want %v", session.UserID, tt.userID)
			}
			if session.ID == "" {
				t.Error("ID should not be empty")
			}
			if session.CreatedAt.Before(before) || session.CreatedAt.After(after) {
				t.Error("CreatedAt should be approximately now")
			}

			expectedExpiry := session.CreatedAt.Add(tt.ttl)
			if !session.ExpiresAt.Equal(expectedExpiry) {
				t.Errorf("ExpiresAt = %v, want %v", session.ExpiresAt, expectedExpiry)
			}
		})
	}
}

func TestSessionIsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "not expired - future",
			expiresAt: time.Now().Add(time.Hour),
			want:      false,
		},
		{
			name:      "expired - past",
			expiresAt: time.Now().Add(-time.Hour),
			want:      true,
		},
		{
			name:      "expired - just passed",
			expiresAt: time.Now().Add(-time.Millisecond),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{ExpiresAt: tt.expiresAt}
			if got := session.IsExpired(); got != tt.want {
				t.Errorf("Session.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSessionIsValid(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "valid - future",
			expiresAt: time.Now().Add(time.Hour),
			want:      true,
		},
		{
			name:      "invalid - past",
			expiresAt: time.Now().Add(-time.Hour),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{ExpiresAt: tt.expiresAt}
			if got := session.IsValid(); got != tt.want {
				t.Errorf("Session.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
