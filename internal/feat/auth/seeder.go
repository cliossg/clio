package auth

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cliossg/clio/internal/feat/profile"
	"github.com/cliossg/clio/pkg/cl/logger"
)

type SeederProfileService interface {
	CreateProfile(ctx context.Context, slug, name, surname, bio, socialLinks, photoPath, createdBy string) (*profile.Profile, error)
}

// Seeder handles seeding auth-related data.
type Seeder struct {
	service         Service
	profileService  SeederProfileService
	assetsFS        embed.FS
	log             logger.Logger
	credentialsPath string
}

// NewSeeder creates a new auth seeder.
func NewSeeder(service Service, profileService SeederProfileService, assetsFS embed.FS, log logger.Logger) *Seeder {
	return &Seeder{
		service:        service,
		profileService: profileService,
		assetsFS:       assetsFS,
		log:            log,
	}
}

// SetCredentialsPath sets the path where credentials will be written.
func (s *Seeder) SetCredentialsPath(path string) {
	s.credentialsPath = path
}

// Start seeds the initial admin user if no users exist.
func (s *Seeder) Start(ctx context.Context) error {
	users, err := s.service.ListUsers(ctx)
	if err != nil {
		return fmt.Errorf("cannot list users: %w", err)
	}

	if len(users) > 0 {
		s.log.Info("Users already exist, skipping auth seeding")
		return nil
	}

	// Create default admin user
	email := "admin@local"
	password := generateRandomPassword()
	name := "admin"

	user, err := s.service.CreateUser(ctx, email, password, name, RoleAdmin, true)
	if err != nil {
		return fmt.Errorf("cannot create admin user: %w", err)
	}

	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	bio := "Site administrator and content curator."
	socialLinks := `[{"Platform":"GitHub","URL":"https://github.com/admin"},{"Platform":"X","URL":"https://x.com/admin"}]`
	userProfile, err := s.profileService.CreateProfile(ctx, slug, "Site", "Admin", bio, socialLinks, "", user.ID.String())
	if err != nil {
		s.log.Errorf("Cannot create profile for admin user: %v", err)
	} else {
		s.service.SetUserProfile(ctx, user.ID, userProfile.ID)
	}

	s.log.Infof("Created admin user: %s", user.Email)

	// Write credentials to file if path is set
	if s.credentialsPath != "" {
		if err := s.writeCredentials(email, password); err != nil {
			s.log.Errorf("Cannot write credentials file: %v", err)
		} else {
			s.log.Infof("Credentials written to: %s", s.credentialsPath)
		}
	} else {
		// Print credentials to log
		s.log.Infof("Admin credentials - Email: %s, Password: %s", email, password)
	}

	return nil
}

func (s *Seeder) writeCredentials(email, password string) error {
	dir := filepath.Dir(s.credentialsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("cannot create directory: %w", err)
	}

	content := fmt.Sprintf("Email: %s\nPassword: %s\n", email, password)
	if err := os.WriteFile(s.credentialsPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("cannot write file: %w", err)
	}

	return nil
}

// Name returns the seeder name for logging.
func (s *Seeder) Name() string {
	return "auth"
}

// Depends returns the names of seeders this one depends on.
func (s *Seeder) Depends() []string {
	return nil
}

// generateRandomPassword generates a simple random password.
func generateRandomPassword() string {
	// For simplicity, use a fixed password for now
	// In production, use crypto/rand
	return "admin123"
}

// ErrUserExists is returned when trying to create a user that already exists.
var ErrUserExists = errors.New("user already exists")
