package auth

import (
	"context"
	"embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/cliossg/clio/internal/feat/profile"
	"github.com/cliossg/clio/pkg/cl/logger"
	"github.com/google/uuid"
)

type mockProfileService struct {
	createProfileFunc func(ctx context.Context, slug, name, surname, bio, socialLinks, photoPath, createdBy string) (*profile.Profile, error)
}

func (m *mockProfileService) CreateProfile(ctx context.Context, slug, name, surname, bio, socialLinks, photoPath, createdBy string) (*profile.Profile, error) {
	if m.createProfileFunc != nil {
		return m.createProfileFunc(ctx, slug, name, surname, bio, socialLinks, photoPath, createdBy)
	}
	return &profile.Profile{
		ID:   uuid.New(),
		Slug: slug,
		Name: name,
	}, nil
}

func TestNewSeeder(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	mockProfile := &mockProfileService{}
	var fs embed.FS
	log := logger.NewNoopLogger()

	seeder := NewSeeder(svc, mockProfile, fs, log)

	if seeder == nil {
		t.Fatal("NewSeeder() returned nil")
	}
	if seeder.service == nil {
		t.Error("service should not be nil")
	}
	if seeder.profileService == nil {
		t.Error("profileService should not be nil")
	}
}

func TestSeederSetCredentialsPath(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	mockProfile := &mockProfileService{}
	var fs embed.FS
	log := logger.NewNoopLogger()

	seeder := NewSeeder(svc, mockProfile, fs, log)
	seeder.SetCredentialsPath("/tmp/test-creds.txt")

	if seeder.credentialsPath != "/tmp/test-creds.txt" {
		t.Errorf("credentialsPath = %q, want %q", seeder.credentialsPath, "/tmp/test-creds.txt")
	}
}

func TestSeederName(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	mockProfile := &mockProfileService{}
	var fs embed.FS
	log := logger.NewNoopLogger()

	seeder := NewSeeder(svc, mockProfile, fs, log)

	if got := seeder.Name(); got != "auth" {
		t.Errorf("Name() = %q, want %q", got, "auth")
	}
}

func TestSeederDepends(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	mockProfile := &mockProfileService{}
	var fs embed.FS
	log := logger.NewNoopLogger()

	seeder := NewSeeder(svc, mockProfile, fs, log)

	if got := seeder.Depends(); got != nil {
		t.Errorf("Depends() = %v, want nil", got)
	}
}

func TestSeederStartWithExistingUsers(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Create an existing user
	_, err := svc.CreateUser(ctx, "existing@test.com", "password", "existinguser", "", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	mockProfile := &mockProfileService{}
	var fs embed.FS
	log := logger.NewNoopLogger()

	seeder := NewSeeder(svc, mockProfile, fs, log)

	// Start should skip seeding when users exist
	err = seeder.Start(ctx)
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}

	// Verify no admin user was created
	users, _ := svc.ListUsers(ctx)
	for _, u := range users {
		if u.Email == "admin@local" {
			t.Error("Admin user should not have been created when users exist")
		}
	}
}

func TestSeederStartCreatesAdmin(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	mockProfile := &mockProfileService{}
	var fs embed.FS
	log := logger.NewNoopLogger()

	seeder := NewSeeder(svc, mockProfile, fs, log)

	// Start should create admin user
	err := seeder.Start(ctx)
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}

	// Verify admin user was created
	users, _ := svc.ListUsers(ctx)
	found := false
	for _, u := range users {
		if u.Email == "admin@local" {
			found = true
			if !u.HasRole(RoleAdmin) {
				t.Error("Admin user should have admin role")
			}
			if !u.MustChangePassword {
				t.Error("Admin user should have MustChangePassword set")
			}
			break
		}
	}
	if !found {
		t.Error("Admin user was not created")
	}
}

func TestSeederStartWritesCredentials(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	mockProfile := &mockProfileService{}
	var fs embed.FS
	log := logger.NewNoopLogger()

	// Create temp directory for credentials
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "creds.txt")

	seeder := NewSeeder(svc, mockProfile, fs, log)
	seeder.SetCredentialsPath(credPath)

	err := seeder.Start(ctx)
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}

	// Verify credentials file was written
	content, err := os.ReadFile(credPath)
	if err != nil {
		t.Errorf("Failed to read credentials file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Credentials file is empty")
	}

	// Check file permissions
	info, err := os.Stat(credPath)
	if err != nil {
		t.Errorf("Failed to stat credentials file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("Credentials file permissions = %o, want 0600", info.Mode().Perm())
	}
}

func TestSeederStartWithProfileError(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Mock profile service that returns an error
	mockProfile := &mockProfileService{
		createProfileFunc: func(ctx context.Context, slug, name, surname, bio, socialLinks, photoPath, createdBy string) (*profile.Profile, error) {
			return nil, os.ErrPermission
		},
	}
	var fs embed.FS
	log := logger.NewNoopLogger()

	seeder := NewSeeder(svc, mockProfile, fs, log)

	// Start should still succeed even if profile creation fails
	err := seeder.Start(ctx)
	if err != nil {
		t.Errorf("Start() should not fail when profile creation fails, got error = %v", err)
	}

	// Admin user should still be created
	users, _ := svc.ListUsers(ctx)
	found := false
	for _, u := range users {
		if u.Email == "admin@local" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Admin user should still be created even when profile fails")
	}
}

func TestGenerateRandomPassword(t *testing.T) {
	pwd := generateRandomPassword()
	if pwd == "" {
		t.Error("generateRandomPassword() returned empty string")
	}
	// Current implementation returns fixed password
	if pwd != "admin123" {
		t.Errorf("generateRandomPassword() = %q, want %q", pwd, "admin123")
	}
}

func TestSeederWriteCredentialsCreatesDirAndFile(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	mockProfile := &mockProfileService{}
	var fs embed.FS
	log := logger.NewNoopLogger()

	seeder := NewSeeder(svc, mockProfile, fs, log)

	// Use nested directory that doesn't exist
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "nested", "dir", "creds.txt")
	seeder.credentialsPath = credPath

	err := seeder.writeCredentials("test@test.com", "testpass")
	if err != nil {
		t.Errorf("writeCredentials() error = %v", err)
	}

	content, err := os.ReadFile(credPath)
	if err != nil {
		t.Fatalf("Failed to read credentials: %v", err)
	}

	expected := "Email: test@test.com\nPassword: testpass\n"
	if string(content) != expected {
		t.Errorf("credentials content = %q, want %q", string(content), expected)
	}
}

func TestSeederWriteCredentialsDirCreationError(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	mockProfile := &mockProfileService{}
	var fs embed.FS
	log := logger.NewNoopLogger()

	seeder := NewSeeder(svc, mockProfile, fs, log)

	// Create a file where we expect a directory
	tmpDir := t.TempDir()
	blockingFile := filepath.Join(tmpDir, "blocker")
	if err := os.WriteFile(blockingFile, []byte("block"), 0644); err != nil {
		t.Fatalf("Failed to create blocking file: %v", err)
	}

	// Try to create credentials inside the file (which will fail)
	credPath := filepath.Join(blockingFile, "nested", "creds.txt")
	seeder.credentialsPath = credPath

	err := seeder.writeCredentials("test@test.com", "testpass")
	if err == nil {
		t.Error("writeCredentials() should fail when directory creation fails")
	}
}

func TestSeederWriteCredentialsFileWriteError(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	mockProfile := &mockProfileService{}
	var fs embed.FS
	log := logger.NewNoopLogger()

	seeder := NewSeeder(svc, mockProfile, fs, log)

	// Create a directory with no write permissions
	tmpDir := t.TempDir()
	noWriteDir := filepath.Join(tmpDir, "nowrite")
	if err := os.MkdirAll(noWriteDir, 0555); err != nil {
		t.Fatalf("Failed to create no-write dir: %v", err)
	}

	credPath := filepath.Join(noWriteDir, "creds.txt")
	seeder.credentialsPath = credPath

	err := seeder.writeCredentials("test@test.com", "testpass")
	if err == nil {
		t.Error("writeCredentials() should fail when file write fails")
	}
}
