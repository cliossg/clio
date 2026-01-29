package ssg

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/cliossg/clio/internal/feat/profile"
	"github.com/cliossg/clio/pkg/cl/logger"
	"github.com/google/uuid"
)

type mockProfileService struct {
	db         *sql.DB
	createFunc func(ctx context.Context, slug, name, surname, bio, socialLinks, photoPath, createdBy string) (*profile.Profile, error)
}

func (m *mockProfileService) CreateProfile(ctx context.Context, slug, name, surname, bio, socialLinks, photoPath, createdBy string) (*profile.Profile, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, slug, name, surname, bio, socialLinks, photoPath, createdBy)
	}

	// Create a real profile in the database
	p := &profile.Profile{
		ID:   uuid.New(),
		Slug: slug,
		Name: name,
	}
	if m.db != nil {
		_, err := m.db.Exec(`INSERT INTO profile (id, short_id, slug, name, surname, bio, social_links, photo_path, created_by, updated_by, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
			p.ID.String(), uuid.New().String()[:8], slug, name, surname, bio, socialLinks, photoPath, uuid.New().String(), uuid.New().String())
		if err != nil {
			return nil, err
		}
	}
	return p, nil
}

func TestSeederNewSeeder(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	mockProfile := &mockProfileService{}
	log := logger.NewNoopLogger()

	seeder := NewSeeder(svc, mockProfile, log)

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

func TestSeederName(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	seeder := NewSeeder(svc, &mockProfileService{}, logger.NewNoopLogger())

	if got := seeder.Name(); got != "ssg" {
		t.Errorf("Name() = %q, want %q", got, "ssg")
	}
}

func TestSeederDepends(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	seeder := NewSeeder(svc, &mockProfileService{}, logger.NewNoopLogger())

	deps := seeder.Depends()
	if len(deps) != 1 || deps[0] != "auth" {
		t.Errorf("Depends() = %v, want [auth]", deps)
	}
}

func TestSeederStartWithExistingSites(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Create an existing site
	site := NewSite("Existing Site", "existing-site", "blog")
	if err := svc.CreateSite(ctx, site); err != nil {
		t.Fatalf("CreateSite failed: %v", err)
	}

	mockProfile := &mockProfileService{}
	seeder := NewSeeder(svc, mockProfile, logger.NewNoopLogger())

	// Start should skip seeding when sites exist
	err := seeder.Start(ctx)
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}

	// Verify no demo site was created
	sites, _ := svc.ListSites(ctx)
	for _, s := range sites {
		if s.Slug == "demo" {
			t.Error("Demo site should not have been created when sites exist")
		}
	}
}

func TestSeederStartCreatesDemoSite(t *testing.T) {
	svc, db, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	mockProfile := &mockProfileService{db: db}
	seeder := NewSeeder(svc, mockProfile, logger.NewNoopLogger())

	// Start should create demo site
	err := seeder.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Verify demo site was created
	site, err := svc.GetSiteBySlug(ctx, "demo")
	if err != nil {
		t.Errorf("Demo site should have been created: %v", err)
	}
	if site.Name != "Demo" {
		t.Errorf("Site name = %q, want %q", site.Name, "Demo")
	}

	// Verify sections were created
	sections, err := svc.GetSections(ctx, site.ID)
	if err != nil {
		t.Errorf("GetSections() error = %v", err)
	}
	if len(sections) < 4 {
		t.Errorf("Expected at least 4 sections (main + 3), got %d", len(sections))
	}

	// Verify settings were created
	settings, err := svc.GetSettings(ctx, site.ID)
	if err != nil {
		t.Errorf("GetSettings() error = %v", err)
	}
	if len(settings) == 0 {
		t.Error("Expected settings to be created")
	}

	// Verify contributors were created
	contributors, err := svc.GetContributors(ctx, site.ID)
	if err != nil {
		t.Errorf("GetContributors() error = %v", err)
	}
	if len(contributors) != 2 {
		t.Errorf("Expected 2 contributors, got %d", len(contributors))
	}

	// Verify content was created
	contents, err := svc.GetAllContentWithMeta(ctx, site.ID)
	if err != nil {
		t.Errorf("GetAllContentWithMeta() error = %v", err)
	}
	if len(contents) < 10 {
		t.Errorf("Expected at least 10 content items, got %d", len(contents))
	}
}

func TestSeederStartWithProfileError(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Mock profile service that returns an error
	mockProfile := &mockProfileService{
		createFunc: func(ctx context.Context, slug, name, surname, bio, socialLinks, photoPath, createdBy string) (*profile.Profile, error) {
			return nil, errors.New("profile service error")
		},
	}
	seeder := NewSeeder(svc, mockProfile, logger.NewNoopLogger())

	// Start should fail when profile creation fails
	err := seeder.Start(ctx)
	if err == nil {
		t.Error("Start() should fail when profile creation fails")
	}
}

func TestSeederStartWithCancelledContext(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mockProfile := &mockProfileService{}
	seeder := NewSeeder(svc, mockProfile, logger.NewNoopLogger())

	// Start should fail with cancelled context
	err := seeder.Start(ctx)
	if err == nil {
		t.Error("Start() should fail with cancelled context")
	}
}

func TestSeederStartVerifySections(t *testing.T) {
	svc, db, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	mockProfile := &mockProfileService{db: db}
	seeder := NewSeeder(svc, mockProfile, logger.NewNoopLogger())

	err := seeder.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	site, _ := svc.GetSiteBySlug(ctx, "demo")
	sections, _ := svc.GetSections(ctx, site.ID)

	// Verify specific section names exist
	sectionNames := make(map[string]bool)
	for _, s := range sections {
		sectionNames[s.Name] = true
	}

	expected := []string{"main", "Coding", "Essays", "Food"}
	for _, name := range expected {
		if !sectionNames[name] {
			t.Errorf("Expected section %q to exist", name)
		}
	}
}

func TestSeederStartVerifySettings(t *testing.T) {
	svc, db, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	mockProfile := &mockProfileService{db: db}
	seeder := NewSeeder(svc, mockProfile, logger.NewNoopLogger())

	err := seeder.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	site, _ := svc.GetSiteBySlug(ctx, "demo")
	settings, _ := svc.GetSettings(ctx, site.ID)

	// Verify specific settings exist
	settingNames := make(map[string]bool)
	for _, s := range settings {
		settingNames[s.Name] = true
	}

	expected := []string{"Site description", "Hero image", "Index max items", "Blocks enabled"}
	for _, name := range expected {
		if !settingNames[name] {
			t.Errorf("Expected setting %q to exist", name)
		}
	}
}

func TestSeederStartVerifyContributors(t *testing.T) {
	svc, db, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	mockProfile := &mockProfileService{db: db}
	seeder := NewSeeder(svc, mockProfile, logger.NewNoopLogger())

	err := seeder.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	site, _ := svc.GetSiteBySlug(ctx, "demo")
	contributors, _ := svc.GetContributors(ctx, site.ID)

	// Verify contributor handles
	handles := make(map[string]bool)
	for _, c := range contributors {
		handles[c.Handle] = true
	}

	expected := []string{"johndoe", "janesmith"}
	for _, handle := range expected {
		if !handles[handle] {
			t.Errorf("Expected contributor with handle %q to exist", handle)
		}
	}

	// Verify contributors have profiles
	for _, c := range contributors {
		if c.ProfileID == nil {
			t.Errorf("Contributor %s should have a profile", c.Handle)
		}
	}
}

func TestSeederStartVerifyContent(t *testing.T) {
	svc, db, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	mockProfile := &mockProfileService{db: db}
	seeder := NewSeeder(svc, mockProfile, logger.NewNoopLogger())

	err := seeder.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	site, _ := svc.GetSiteBySlug(ctx, "demo")
	contents, _ := svc.GetAllContentWithMeta(ctx, site.ID)

	// Check for variety of content types
	hasPages := false
	hasArticles := false
	hasContentWithContributor := false

	for _, c := range contents {
		if c.Kind == "page" {
			hasPages = true
		}
		if c.Kind == "article" {
			hasArticles = true
		}
		if c.ContributorID != nil {
			hasContentWithContributor = true
		}
	}

	if !hasPages {
		t.Error("Expected page content to be created")
	}
	if !hasArticles {
		t.Error("Expected article content to be created")
	}
	if !hasContentWithContributor {
		t.Error("Expected some content to have contributors")
	}
}

func TestSeederStartVerifyTags(t *testing.T) {
	svc, db, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	mockProfile := &mockProfileService{db: db}
	seeder := NewSeeder(svc, mockProfile, logger.NewNoopLogger())

	err := seeder.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	site, _ := svc.GetSiteBySlug(ctx, "demo")
	tags, err := svc.GetTags(ctx, site.ID)
	if err != nil {
		t.Fatalf("GetTags() error = %v", err)
	}

	if len(tags) == 0 {
		t.Error("Expected tags to be created")
	}

	// Verify some expected tag names exist
	tagNames := make(map[string]bool)
	for _, tag := range tags {
		tagNames[tag.Name] = true
	}

	expectedTags := []string{"golang", "tutorial", "beginner"}
	found := 0
	for _, expected := range expectedTags {
		if tagNames[expected] {
			found++
		}
	}

	if found == 0 {
		t.Error("Expected at least some known tags to be created")
	}
}

func TestSeederStartVerifyContentDetails(t *testing.T) {
	svc, db, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	mockProfile := &mockProfileService{db: db}
	seeder := NewSeeder(svc, mockProfile, logger.NewNoopLogger())

	err := seeder.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	site, _ := svc.GetSiteBySlug(ctx, "demo")
	contents, _ := svc.GetAllContentWithMeta(ctx, site.ID)

	// Check that content has expected fields populated
	hasContentWithSummary := false
	hasContentWithBody := false
	hasPublishedContent := false

	for _, c := range contents {
		if c.Summary != "" {
			hasContentWithSummary = true
		}
		if c.Body != "" {
			hasContentWithBody = true
		}
		if c.PublishedAt != nil {
			hasPublishedContent = true
		}
	}

	if !hasContentWithSummary {
		t.Error("Expected at least some content to have summary")
	}
	if !hasContentWithBody {
		t.Error("Expected at least some content to have body")
	}
	if !hasPublishedContent {
		t.Error("Expected at least some content to be published")
	}
}
