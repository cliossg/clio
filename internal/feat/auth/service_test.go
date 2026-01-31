package auth

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/cliossg/clio/internal/testutil"
	"github.com/cliossg/clio/pkg/cl/config"
	"github.com/cliossg/clio/pkg/cl/logger"
	"github.com/google/uuid"
)

func newTestLogger() logger.Logger {
	return logger.NewNoopLogger()
}

func setupTestService(t *testing.T) (Service, *sql.DB, func()) {
	t.Helper()

	db, err := testutil.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	cfg := &config.Config{
		Auth: config.AuthConfig{
			SessionTTL: "24h",
		},
	}

	svc := NewService(&testutil.TestDBProvider{DB: db}, cfg, newTestLogger())
	if err := svc.Start(context.Background()); err != nil {
		db.Close()
		t.Fatalf("Failed to start service: %v", err)
	}

	cleanup := func() {
		svc.Stop(context.Background())
		db.Close()
	}

	return svc, db, cleanup
}

func TestServiceStart(t *testing.T) {
	tests := []struct {
		name       string
		sessionTTL string
		wantTTL    time.Duration
	}{
		{
			name:       "valid TTL",
			sessionTTL: "24h",
			wantTTL:    24 * time.Hour,
		},
		{
			name:       "invalid TTL - falls back to default",
			sessionTTL: "invalid",
			wantTTL:    24 * time.Hour,
		},
		{
			name:       "empty TTL - falls back to default",
			sessionTTL: "",
			wantTTL:    24 * time.Hour,
		},
		{
			name:       "short TTL",
			sessionTTL: "1h",
			wantTTL:    time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := testutil.NewTestDB()
			if err != nil {
				t.Fatalf("Failed to create test database: %v", err)
			}
			defer db.Close()

			cfg := &config.Config{
				Auth: config.AuthConfig{
					SessionTTL: tt.sessionTTL,
				},
			}

			svc := NewService(&testutil.TestDBProvider{DB: db}, cfg, newTestLogger())
			err = svc.Start(context.Background())
			if err != nil {
				t.Errorf("Start() error = %v", err)
				return
			}

			if got := svc.GetSessionTTL(); got != tt.wantTTL {
				t.Errorf("GetSessionTTL() = %v, want %v", got, tt.wantTTL)
			}
		})
	}
}

func TestServiceCreateUser(t *testing.T) {
	tests := []struct {
		name               string
		email              string
		password           string
		userName           string
		roles              string
		mustChangePassword bool
		wantErr            bool
		checkUser          func(t *testing.T, u *User)
	}{
		{
			name:               "valid user",
			email:              "test@example.com",
			password:           "password123",
			userName:           "testuser",
			roles:              "admin",
			mustChangePassword: false,
			wantErr:            false,
			checkUser: func(t *testing.T, u *User) {
				if u.Email != "test@example.com" {
					t.Errorf("Email = %q, want %q", u.Email, "test@example.com")
				}
				if u.Roles != "admin" {
					t.Errorf("Roles = %q, want %q", u.Roles, "admin")
				}
			},
		},
		{
			name:               "user with mustChangePassword",
			email:              "change@example.com",
			password:           "temppass",
			userName:           "changeuser",
			roles:              "editor",
			mustChangePassword: true,
			wantErr:            false,
			checkUser: func(t *testing.T, u *User) {
				if !u.MustChangePassword {
					t.Error("MustChangePassword should be true")
				}
			},
		},
		{
			name:               "empty roles - uses default",
			email:              "default@example.com",
			password:           "password123",
			userName:           "defaultuser",
			roles:              "",
			mustChangePassword: false,
			wantErr:            false,
			checkUser: func(t *testing.T, u *User) {
				if u.Roles != RoleEditor {
					t.Errorf("Roles = %q, want %q", u.Roles, RoleEditor)
				}
			},
		},
		{
			name:               "password too long",
			email:              "long@example.com",
			password:           string(make([]byte, 100)),
			userName:           "longuser",
			roles:              "",
			mustChangePassword: false,
			wantErr:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, cleanup := setupTestService(t)
			defer cleanup()

			user, err := svc.CreateUser(context.Background(), tt.email, tt.password, tt.userName, tt.roles, tt.mustChangePassword)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkUser != nil {
				tt.checkUser(t, user)
			}
		})
	}
}

func TestServiceCreateUserDuplicateEmail(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	_, err := svc.CreateUser(ctx, "dup@example.com", "password", "user1", "", false)
	if err != nil {
		t.Fatalf("First CreateUser failed: %v", err)
	}

	_, err = svc.CreateUser(ctx, "dup@example.com", "password", "user2", "", false)
	if err == nil {
		t.Error("Expected error for duplicate email, got nil")
	}
}

func TestServiceGetUser(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	created, err := svc.CreateUser(ctx, "get@example.com", "password", "getuser", "", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr error
	}{
		{
			name:    "existing user",
			id:      created.ID,
			wantErr: nil,
		},
		{
			name:    "non-existent user",
			id:      uuid.New(),
			wantErr: ErrUserNotFound,
		},
		{
			name:    "nil UUID",
			id:      uuid.Nil,
			wantErr: ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := svc.GetUser(ctx, tt.id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetUser() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("GetUser() unexpected error = %v", err)
				return
			}
			if user.ID != tt.id {
				t.Errorf("GetUser() returned wrong user ID")
			}
		})
	}
}

func TestServiceGetUserByEmail(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	_, err := svc.CreateUser(ctx, "email@example.com", "password", "emailuser", "", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	tests := []struct {
		name    string
		email   string
		wantErr error
	}{
		{
			name:    "existing email",
			email:   "email@example.com",
			wantErr: nil,
		},
		{
			name:    "non-existent email",
			email:   "notfound@example.com",
			wantErr: ErrUserNotFound,
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: ErrUserNotFound,
		},
		{
			name:    "wrong case",
			email:   "EMAIL@example.com",
			wantErr: ErrUserNotFound, // SQLite is case-sensitive by default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := svc.GetUserByEmail(ctx, tt.email)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetUserByEmail() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("GetUserByEmail() unexpected error = %v", err)
				return
			}
			if user.Email != tt.email {
				t.Errorf("GetUserByEmail() returned wrong email")
			}
		})
	}
}

func TestServiceListUsers(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Initially empty
	users, err := svc.ListUsers(ctx)
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(users) != 0 {
		t.Errorf("ListUsers() returned %d users, want 0", len(users))
	}

	// Create some users
	for i := 0; i < 3; i++ {
		_, err := svc.CreateUser(ctx, "list"+string(rune('a'+i))+"@example.com", "password", "listuser"+string(rune('a'+i)), "", false)
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}
	}

	users, err = svc.ListUsers(ctx)
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(users) != 3 {
		t.Errorf("ListUsers() returned %d users, want 3", len(users))
	}
}

func TestServiceUpdateUser(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	user, err := svc.CreateUser(ctx, "update@example.com", "password", "updateuser", "editor", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	adminUser, err := svc.CreateUser(ctx, "admin@example.com", "password", "adminuser", "admin", false)
	if err != nil {
		t.Fatalf("CreateUser admin failed: %v", err)
	}

	tests := []struct {
		name    string
		setup   func() *User
		wantErr error
	}{
		{
			name: "update name",
			setup: func() *User {
				u, _ := svc.GetUser(ctx, user.ID)
				u.Name = "newname"
				u.UpdatedAt = time.Now()
				return u
			},
			wantErr: nil,
		},
		{
			name: "update email",
			setup: func() *User {
				u, _ := svc.GetUser(ctx, user.ID)
				u.Email = "newemail@example.com"
				u.UpdatedAt = time.Now()
				return u
			},
			wantErr: nil,
		},
		{
			name: "try to add admin role",
			setup: func() *User {
				u, _ := svc.GetUser(ctx, user.ID)
				u.Roles = "admin"
				u.UpdatedAt = time.Now()
				return u
			},
			wantErr: ErrCannotChangeAdmin,
		},
		{
			name: "try to remove admin role",
			setup: func() *User {
				u, _ := svc.GetUser(ctx, adminUser.ID)
				u.Roles = "editor"
				u.UpdatedAt = time.Now()
				return u
			},
			wantErr: ErrCannotChangeAdmin,
		},
		{
			name: "admin keeps admin role",
			setup: func() *User {
				u, _ := svc.GetUser(ctx, adminUser.ID)
				u.Name = "newadminname"
				u.UpdatedAt = time.Now()
				return u
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userToUpdate := tt.setup()
			err := svc.UpdateUser(ctx, userToUpdate)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("UpdateUser() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("UpdateUser() unexpected error = %v", err)
			}
		})
	}
}

func TestServiceDeleteUser(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	user, err := svc.CreateUser(ctx, "delete@example.com", "password", "deleteuser", "", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Create a session for the user
	_, err = svc.CreateSession(ctx, user.ID)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "delete existing user",
			id:      user.ID,
			wantErr: false,
		},
		{
			name:    "delete non-existent user",
			id:      uuid.New(),
			wantErr: false, // SQLite DELETE doesn't error on non-existent
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.DeleteUser(ctx, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteUser() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Verify user was deleted
	_, err = svc.GetUser(ctx, user.ID)
	if !errors.Is(err, ErrUserNotFound) {
		t.Error("User should have been deleted")
	}
}

func TestServiceAuthenticate(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Create an active user
	_, err := svc.CreateUser(ctx, "auth@example.com", "correctpassword", "authuser", "", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Create an inactive user
	inactiveUser, err := svc.CreateUser(ctx, "inactive@example.com", "password", "inactiveuser", "", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	inactiveUser.Status = "inactive"
	inactiveUser.UpdatedAt = time.Now()
	if err := svc.UpdateUser(ctx, inactiveUser); err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}

	tests := []struct {
		name     string
		email    string
		password string
		wantErr  error
	}{
		{
			name:     "valid credentials",
			email:    "auth@example.com",
			password: "correctpassword",
			wantErr:  nil,
		},
		{
			name:     "wrong password",
			email:    "auth@example.com",
			password: "wrongpassword",
			wantErr:  ErrInvalidCredentials,
		},
		{
			name:     "non-existent email",
			email:    "notfound@example.com",
			password: "anypassword",
			wantErr:  ErrInvalidCredentials,
		},
		{
			name:     "inactive user",
			email:    "inactive@example.com",
			password: "password",
			wantErr:  ErrUserNotActive,
		},
		{
			name:     "empty email",
			email:    "",
			password: "password",
			wantErr:  ErrInvalidCredentials,
		},
		{
			name:     "empty password",
			email:    "auth@example.com",
			password: "",
			wantErr:  ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := svc.Authenticate(ctx, tt.email, tt.password)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Authenticate() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("Authenticate() unexpected error = %v", err)
				return
			}
			if user == nil {
				t.Error("Authenticate() returned nil user")
			}
		})
	}
}

func TestServiceCreateSession(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	user, err := svc.CreateUser(ctx, "session@example.com", "password", "sessionuser", "", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	tests := []struct {
		name    string
		userID  uuid.UUID
		wantErr bool
	}{
		{
			name:    "valid user",
			userID:  user.ID,
			wantErr: false,
		},
		{
			name:    "non existent user fails FK constraint",
			userID:  uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, err := svc.CreateSession(ctx, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSession() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if session.ID == "" {
					t.Error("Session ID should not be empty")
				}
				if session.UserID != tt.userID {
					t.Error("Session UserID mismatch")
				}
			}
		})
	}
}

func TestServiceValidateSession(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	user, err := svc.CreateUser(ctx, "validate@example.com", "password", "validateuser", "", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	session, err := svc.CreateSession(ctx, user.ID)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	tests := []struct {
		name      string
		sessionID string
		wantErr   error
	}{
		{
			name:      "valid session",
			sessionID: session.ID,
			wantErr:   nil,
		},
		{
			name:      "non-existent session",
			sessionID: uuid.New().String(),
			wantErr:   ErrSessionNotFound,
		},
		{
			name:      "empty session ID",
			sessionID: "",
			wantErr:   ErrSessionNotFound,
		},
		{
			name:      "invalid session ID format",
			sessionID: "invalid",
			wantErr:   ErrSessionNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := svc.ValidateSession(ctx, tt.sessionID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateSession() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("ValidateSession() unexpected error = %v", err)
				return
			}
			if info.UserID != user.ID.String() {
				t.Errorf("ValidateSession() userID = %v, want %v", info.UserID, user.ID.String())
			}
			if info.UserName != user.Name {
				t.Errorf("ValidateSession() userName = %v, want %v", info.UserName, user.Name)
			}
			if info.UserRoles != user.Roles {
				t.Errorf("ValidateSession() userRoles = %v, want %v", info.UserRoles, user.Roles)
			}
		})
	}
}

func TestServiceDeleteSession(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	user, err := svc.CreateUser(ctx, "delsession@example.com", "password", "delsessionuser", "", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	session, err := svc.CreateSession(ctx, user.ID)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	tests := []struct {
		name      string
		sessionID string
		wantErr   bool
	}{
		{
			name:      "delete existing session",
			sessionID: session.ID,
			wantErr:   false,
		},
		{
			name:      "delete non-existent session",
			sessionID: uuid.New().String(),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.DeleteSession(ctx, tt.sessionID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteSession() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Verify session was deleted
	_, err = svc.ValidateSession(ctx, session.ID)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Error("Session should have been deleted")
	}
}

func TestServiceSetUserProfile(t *testing.T) {
	svc, db, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	user, err := svc.CreateUser(ctx, "profile@example.com", "password", "profileuser", "", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Create a real profile in the database
	profileID := uuid.New()
	_, err = db.Exec(`INSERT INTO profile (id, short_id, slug, name, created_by, updated_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		profileID.String(), "abc12345", "test-profile", "Test Profile", user.ID.String(), user.ID.String())
	if err != nil {
		t.Fatalf("Failed to create test profile: %v", err)
	}

	tests := []struct {
		name      string
		userID    uuid.UUID
		profileID uuid.UUID
		wantErr   bool
	}{
		{
			name:      "set profile for existing user",
			userID:    user.ID,
			profileID: profileID,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.SetUserProfile(ctx, tt.userID, tt.profileID)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetUserProfile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServiceGetSessionTTL(t *testing.T) {
	db, err := testutil.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	cfg := &config.Config{
		Auth: config.AuthConfig{
			SessionTTL: "12h",
		},
	}

	svc := NewService(&testutil.TestDBProvider{DB: db}, cfg, newTestLogger())
	svc.Start(context.Background())

	if got := svc.GetSessionTTL(); got != 12*time.Hour {
		t.Errorf("GetSessionTTL() = %v, want %v", got, 12*time.Hour)
	}
}

func TestServiceDeleteSessionVerifyDeleted(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	user, err := svc.CreateUser(ctx, "deletesession@test.com", "password123", "deletesessionuser", "", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Create a session
	session, err := svc.CreateSession(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Delete the session
	if err := svc.DeleteSession(ctx, session.ID); err != nil {
		t.Errorf("DeleteSession() error = %v", err)
	}

	// Verify session is deleted by checking it's no longer valid
	_, err = svc.ValidateSession(ctx, session.ID)
	if err == nil {
		t.Error("Expected error when validating deleted session")
	}
}

func TestServiceDeleteUserVerifyDeleted(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	user, err := svc.CreateUser(ctx, "todelete@test.com", "password123", "todeleteuser", "", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Delete the user
	if err := svc.DeleteUser(ctx, user.ID); err != nil {
		t.Errorf("DeleteUser() error = %v", err)
	}

	// Verify user is deleted
	_, err = svc.GetUser(ctx, user.ID)
	if err == nil {
		t.Error("Expected error when getting deleted user")
	}
}

func TestServiceUpdateUserChangeEmail(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	user, err := svc.CreateUser(ctx, "oldemail@test.com", "password", "updateemailuser", "editor", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Update user email
	user.Email = "newemail@test.com"

	if err := svc.UpdateUser(ctx, user); err != nil {
		t.Errorf("UpdateUser() error = %v", err)
	}

	// Verify email was changed
	updated, err := svc.GetUser(ctx, user.ID)
	if err != nil {
		t.Errorf("GetUser() error = %v", err)
	}
	if updated.Email != "newemail@test.com" {
		t.Errorf("Email = %q, want %q", updated.Email, "newemail@test.com")
	}
}

func TestServiceListUsersReturnsMultiple(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Create multiple users
	for i := 0; i < 3; i++ {
		_, err := svc.CreateUser(ctx, "listtest"+string(rune('1'+i))+"@test.com", "password", "listtestuser"+string(rune('1'+i)), "", false)
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}
	}

	users, err := svc.ListUsers(ctx)
	if err != nil {
		t.Errorf("ListUsers() error = %v", err)
	}
	if len(users) < 3 {
		t.Errorf("ListUsers() returned %d users, expected at least 3", len(users))
	}
}

func TestServiceGetUserWithProfile(t *testing.T) {
	svc, db, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Create a profile first
	profileID := uuid.New()
	_, err := db.Exec(`INSERT INTO profile (id, short_id, slug, name, created_by, updated_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		profileID.String(), "prof1234", "profile-user", "Profile User", uuid.New().String(), uuid.New().String())
	if err != nil {
		t.Fatalf("Failed to create profile: %v", err)
	}

	// Create user
	user, err := svc.CreateUser(ctx, "profileuser@test.com", "password", "profileuser", "", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Set user profile
	err = svc.SetUserProfile(ctx, user.ID, profileID)
	if err != nil {
		t.Fatalf("SetUserProfile failed: %v", err)
	}

	// Get user and verify profile ID is loaded (tests fromSQLCUser with ProfileID.Valid)
	got, err := svc.GetUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}
	if got.ProfileID == nil {
		t.Error("ProfileID should not be nil after SetUserProfile")
	} else if *got.ProfileID != profileID {
		t.Errorf("ProfileID = %v, want %v", *got.ProfileID, profileID)
	}
}

func TestServiceDeleteUserWithSessions(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	user, err := svc.CreateUser(ctx, "withsessions@test.com", "password", "sessionsuser", "", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Create multiple sessions
	session1, _ := svc.CreateSession(ctx, user.ID)
	session2, _ := svc.CreateSession(ctx, user.ID)

	// Verify sessions exist
	_, err = svc.ValidateSession(ctx, session1.ID)
	if err != nil {
		t.Fatalf("Session 1 should exist: %v", err)
	}
	_, err = svc.ValidateSession(ctx, session2.ID)
	if err != nil {
		t.Fatalf("Session 2 should exist: %v", err)
	}

	// Delete user (should also delete sessions)
	err = svc.DeleteUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}

	// Verify all sessions are deleted
	_, err = svc.ValidateSession(ctx, session1.ID)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Session 1 should be deleted, got error: %v", err)
	}
	_, err = svc.ValidateSession(ctx, session2.ID)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Session 2 should be deleted, got error: %v", err)
	}
}

func TestServiceUpdateUserStatus(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	user, err := svc.CreateUser(ctx, "statususer@test.com", "password", "statususer", "editor", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Update status
	user.Status = "suspended"
	user.UpdatedAt = time.Now()

	err = svc.UpdateUser(ctx, user)
	if err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}

	// Verify
	got, err := svc.GetUser(ctx, user.ID)
	if err != nil {
		t.Errorf("GetUser() error = %v", err)
	}
	if got.Status != "suspended" {
		t.Errorf("Status = %q, want %q", got.Status, "suspended")
	}
}

func TestServiceUpdateUserMustChangePassword(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	user, err := svc.CreateUser(ctx, "changepass@test.com", "password", "changepassuser", "editor", true)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Verify initially MustChangePassword is true
	if !user.MustChangePassword {
		t.Error("MustChangePassword should be true initially")
	}

	// Update to false
	user.MustChangePassword = false
	user.UpdatedAt = time.Now()

	err = svc.UpdateUser(ctx, user)
	if err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}

	// Verify
	got, err := svc.GetUser(ctx, user.ID)
	if err != nil {
		t.Errorf("GetUser() error = %v", err)
	}
	if got.MustChangePassword {
		t.Error("MustChangePassword should be false after update")
	}
}

func TestServiceListUsersWithProfile(t *testing.T) {
	svc, db, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Create a profile
	profileID := uuid.New()
	_, err := db.Exec(`INSERT INTO profile (id, short_id, slug, name, created_by, updated_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		profileID.String(), "list1234", "list-profile", "List Profile", uuid.New().String(), uuid.New().String())
	if err != nil {
		t.Fatalf("Failed to create profile: %v", err)
	}

	// Create user with profile
	user, _ := svc.CreateUser(ctx, "listwithprofile@test.com", "password", "listwithprofile", "", false)
	svc.SetUserProfile(ctx, user.ID, profileID)

	// Create user without profile
	svc.CreateUser(ctx, "listwithoutprofile@test.com", "password", "listwithoutprofile", "", false)

	// List all users
	users, err := svc.ListUsers(ctx)
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}

	// Verify one has profile, one doesn't
	profileCount := 0
	for _, u := range users {
		if u.ProfileID != nil {
			profileCount++
		}
	}
	if profileCount != 1 {
		t.Errorf("Expected 1 user with profile, got %d", profileCount)
	}
}

func TestServiceDeleteUserNonExistent(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Try to delete non-existent user - SQLite DELETE doesn't error
	err := svc.DeleteUser(ctx, uuid.New())
	// Just exercise the code path
	if err != nil {
		t.Logf("DeleteUser for non-existent returned: %v", err)
	}
}

func TestServiceDeleteSessionNonExistent(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Try to delete non-existent session
	err := svc.DeleteSession(ctx, uuid.New().String())
	// Just exercise the code path
	if err != nil {
		t.Logf("DeleteSession for non-existent returned: %v", err)
	}
}

func TestServiceUpdateUserName(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	user, err := svc.CreateUser(ctx, "updatename@test.com", "password", "oldname", "editor", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Update user name
	user.Name = "newname"
	user.UpdatedAt = time.Now()

	if err := svc.UpdateUser(ctx, user); err != nil {
		t.Errorf("UpdateUser() error = %v", err)
	}

	// Verify name was changed
	updated, err := svc.GetUser(ctx, user.ID)
	if err != nil {
		t.Errorf("GetUser() error = %v", err)
	}
	if updated.Name != "newname" {
		t.Errorf("Name = %q, want %q", updated.Name, "newname")
	}
}

func TestServiceValidateSessionExpired(t *testing.T) {
	svc, db, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	user, err := svc.CreateUser(ctx, "expired@test.com", "password", "expireduser", "", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Create session that's already expired
	sessionID := uuid.New().String()
	_, err = db.Exec(`INSERT INTO session (id, user_id, expires_at, created_at) VALUES (?, ?, datetime('now', '-1 hour'), datetime('now'))`,
		sessionID, user.ID.String())
	if err != nil {
		t.Fatalf("Failed to create expired session: %v", err)
	}

	// Try to validate expired session
	_, err = svc.ValidateSession(ctx, sessionID)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Expected ErrSessionNotFound for expired session, got: %v", err)
	}
}
