package ssg

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

	cfg := &config.Config{}
	svc := NewService(&testutil.TestDBProvider{DB: db}, nil, cfg, newTestLogger())
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

func createTestSite(t *testing.T, svc Service, name, slug string) *Site {
	t.Helper()
	site := NewSite(name, slug, "blog")
	site.CreatedBy = uuid.New()
	site.UpdatedBy = site.CreatedBy
	if err := svc.CreateSite(context.Background(), site); err != nil {
		t.Fatalf("Failed to create test site: %v", err)
	}
	return site
}

func TestServiceCreateSite(t *testing.T) {
	tests := []struct {
		name    string
		site    *Site
		wantErr bool
	}{
		{
			name: "valid site",
			site: func() *Site {
				s := NewSite("Test Site", "test-site", "blog")
				s.CreatedBy = uuid.New()
				s.UpdatedBy = s.CreatedBy
				return s
			}(),
			wantErr: false,
		},
		{
			name: "site with structured mode",
			site: func() *Site {
				s := NewSite("Docs Site", "docs-site", "structured")
				s.CreatedBy = uuid.New()
				s.UpdatedBy = s.CreatedBy
				return s
			}(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, cleanup := setupTestService(t)
			defer cleanup()

			err := svc.CreateSite(context.Background(), tt.site)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSite() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServiceCreateSiteDuplicateSlug(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	createTestSite(t, svc, "Site 1", "same-slug")

	site2 := NewSite("Site 2", "same-slug", "blog")
	site2.CreatedBy = uuid.New()
	site2.UpdatedBy = site2.CreatedBy

	if err := svc.CreateSite(ctx, site2); err == nil {
		t.Error("Expected error for duplicate slug, got nil")
	}
}

func TestServiceGetSite(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	created := createTestSite(t, svc, "Get Site", "get-site")

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr error
	}{
		{
			name:    "existing site",
			id:      created.ID,
			wantErr: nil,
		},
		{
			name:    "non existent site",
			id:      uuid.New(),
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			site, err := svc.GetSite(ctx, tt.id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetSite() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("GetSite() unexpected error = %v", err)
				return
			}
			if site.ID != tt.id {
				t.Error("GetSite() returned wrong site")
			}
		})
	}
}

func TestServiceGetSiteBySlug(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	createTestSite(t, svc, "Slug Site", "slug-site")

	tests := []struct {
		name    string
		slug    string
		wantErr error
	}{
		{
			name:    "existing slug",
			slug:    "slug-site",
			wantErr: nil,
		},
		{
			name:    "non existent slug",
			slug:    "not-found",
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.GetSiteBySlug(ctx, tt.slug)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetSiteBySlug() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("GetSiteBySlug() unexpected error = %v", err)
			}
		})
	}
}

func TestServiceListSites(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	sites, err := svc.ListSites(ctx)
	if err != nil {
		t.Fatalf("ListSites() error = %v", err)
	}
	if len(sites) != 0 {
		t.Errorf("ListSites() returned %d sites, want 0", len(sites))
	}

	for i := 0; i < 3; i++ {
		createTestSite(t, svc, "List Site "+string(rune('A'+i)), "list-site-"+string(rune('a'+i)))
	}

	sites, err = svc.ListSites(ctx)
	if err != nil {
		t.Fatalf("ListSites() error = %v", err)
	}
	if len(sites) != 3 {
		t.Errorf("ListSites() returned %d sites, want 3", len(sites))
	}
}

func TestServiceUpdateSite(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Update Site", "update-site")

	site.Name = "Updated Site Name"
	site.UpdatedAt = time.Now()

	if err := svc.UpdateSite(ctx, site); err != nil {
		t.Errorf("UpdateSite() error = %v", err)
	}

	updated, _ := svc.GetSite(ctx, site.ID)
	if updated.Name != "Updated Site Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "Updated Site Name")
	}
}

func TestServiceDeleteSite(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Delete Site", "delete-site")

	if err := svc.DeleteSite(ctx, site.ID); err != nil {
		t.Errorf("DeleteSite() error = %v", err)
	}

	_, err := svc.GetSite(ctx, site.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Error("Site should have been deleted")
	}
}

func TestServiceCreateSection(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Section Site", "section-site")

	section := NewSection(site.ID, "Blog", "Blog section", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy

	if err := svc.CreateSection(ctx, section); err != nil {
		t.Errorf("CreateSection() error = %v", err)
	}

	got, err := svc.GetSection(ctx, section.ID)
	if err != nil {
		t.Errorf("GetSection() error = %v", err)
	}
	if got.Name != "Blog" {
		t.Errorf("Name = %q, want %q", got.Name, "Blog")
	}
}

func TestServiceGetSection(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Get Section Site", "get-section-site")

	section := NewSection(site.ID, "Test Section", "Description", "/test")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr error
	}{
		{
			name:    "existing section",
			id:      section.ID,
			wantErr: nil,
		},
		{
			name:    "non existent section",
			id:      uuid.New(),
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.GetSection(ctx, tt.id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetSection() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestServiceGetSections(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Sections Site", "sections-site")

	for i := 0; i < 3; i++ {
		section := NewSection(site.ID, "Section "+string(rune('A'+i)), "", "/section-"+string(rune('a'+i)))
		section.CreatedBy = uuid.New()
		section.UpdatedBy = section.CreatedBy
		svc.CreateSection(ctx, section)
	}

	sections, err := svc.GetSections(ctx, site.ID)
	if err != nil {
		t.Fatalf("GetSections() error = %v", err)
	}
	if len(sections) != 3 {
		t.Errorf("GetSections() returned %d sections, want 3", len(sections))
	}
}

func TestServiceUpdateSection(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Update Section Site", "update-section-site")

	section := NewSection(site.ID, "Original", "Original desc", "/original")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	section.Name = "Updated"
	section.UpdatedAt = time.Now()

	if err := svc.UpdateSection(ctx, section); err != nil {
		t.Errorf("UpdateSection() error = %v", err)
	}
}

func TestServiceDeleteSection(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Delete Section Site", "delete-section-site")

	section := NewSection(site.ID, "To Delete", "", "/delete")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	if err := svc.DeleteSection(ctx, section.ID); err != nil {
		t.Errorf("DeleteSection() error = %v", err)
	}

	_, err := svc.GetSection(ctx, section.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Error("Section should have been deleted")
	}
}

func TestServiceCreateContent(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Content Site", "content-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	content := NewContent(site.ID, section.ID, "Test Post", "Post body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy

	if err := svc.CreateContent(ctx, content); err != nil {
		t.Errorf("CreateContent() error = %v", err)
	}
}

func TestServiceGetContent(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Get Content Site", "get-content-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	content := NewContent(site.ID, section.ID, "Test Post", "Body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr error
	}{
		{
			name:    "existing content",
			id:      content.ID,
			wantErr: nil,
		},
		{
			name:    "non existent content",
			id:      uuid.New(),
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.GetContent(ctx, tt.id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetContent() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestServiceGetContentWithMeta(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Meta Content Site", "meta-content-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	content := NewContent(site.ID, section.ID, "Meta Post", "Body with meta")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	meta := NewMeta(site.ID, content.ID)
	meta.Description = "Test description"
	meta.Keywords = "test,keywords"
	meta.CreatedBy = uuid.New()
	meta.UpdatedBy = meta.CreatedBy
	svc.CreateMeta(ctx, meta)

	got, err := svc.GetContentWithMeta(ctx, content.ID)
	if err != nil {
		t.Errorf("GetContentWithMeta() error = %v", err)
	}
	if got.Meta == nil {
		t.Error("Meta should not be nil")
	}
}

func TestServiceGetContentWithPagination(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Pagination Site", "pagination-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	for i := 0; i < 10; i++ {
		content := NewContent(site.ID, section.ID, "Post "+string(rune('A'+i)), "Body")
		content.CreatedBy = uuid.New()
		content.UpdatedBy = content.CreatedBy
		svc.CreateContent(ctx, content)
	}

	tests := []struct {
		name      string
		offset    int
		limit     int
		search    string
		wantCount int
		wantTotal int
	}{
		{
			name:      "first page",
			offset:    0,
			limit:     5,
			search:    "",
			wantCount: 5,
			wantTotal: 10,
		},
		{
			name:      "second page",
			offset:    5,
			limit:     5,
			search:    "",
			wantCount: 5,
			wantTotal: 10,
		},
		{
			name:      "beyond data",
			offset:    20,
			limit:     5,
			search:    "",
			wantCount: 0,
			wantTotal: 10,
		},
		{
			name:      "with search",
			offset:    0,
			limit:     10,
			search:    "Post A",
			wantCount: 1,
			wantTotal: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contents, total, err := svc.GetContentWithPagination(ctx, site.ID, tt.offset, tt.limit, tt.search)
			if err != nil {
				t.Errorf("GetContentWithPagination() error = %v", err)
				return
			}
			if len(contents) != tt.wantCount {
				t.Errorf("Got %d contents, want %d", len(contents), tt.wantCount)
			}
			if total != tt.wantTotal {
				t.Errorf("Total = %d, want %d", total, tt.wantTotal)
			}
		})
	}
}

func TestServiceUpdateContent(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Update Content Site", "update-content-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	content := NewContent(site.ID, section.ID, "Original Title", "Original body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	content.Heading = "Updated Title"
	content.Body = "Updated body"
	content.UpdatedAt = time.Now()

	if err := svc.UpdateContent(ctx, content); err != nil {
		t.Errorf("UpdateContent() error = %v", err)
	}

	updated, _ := svc.GetContent(ctx, content.ID)
	if updated.Heading != "Updated Title" {
		t.Errorf("Heading = %q, want %q", updated.Heading, "Updated Title")
	}
}

func TestServiceDeleteContent(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Delete Content Site", "delete-content-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	content := NewContent(site.ID, section.ID, "To Delete", "Body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	if err := svc.DeleteContent(ctx, content.ID); err != nil {
		t.Errorf("DeleteContent() error = %v", err)
	}

	_, err := svc.GetContent(ctx, content.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Error("Content should have been deleted")
	}
}

func TestServiceCreateLayout(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Layout Site", "layout-site")

	layout := NewLayout(site.ID, "Default", "Default layout")
	layout.Code = "<html>{{.Content}}</html>"
	layout.CreatedBy = uuid.New()
	layout.UpdatedBy = layout.CreatedBy

	if err := svc.CreateLayout(ctx, layout); err != nil {
		t.Errorf("CreateLayout() error = %v", err)
	}
}

func TestServiceGetLayout(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Get Layout Site", "get-layout-site")

	layout := NewLayout(site.ID, "Test Layout", "Description")
	layout.CreatedBy = uuid.New()
	layout.UpdatedBy = layout.CreatedBy
	svc.CreateLayout(ctx, layout)

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr error
	}{
		{
			name:    "existing layout",
			id:      layout.ID,
			wantErr: nil,
		},
		{
			name:    "non existent layout",
			id:      uuid.New(),
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.GetLayout(ctx, tt.id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetLayout() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestServiceGetLayouts(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Layouts Site", "layouts-site")

	for i := 0; i < 3; i++ {
		layout := NewLayout(site.ID, "Layout "+string(rune('A'+i)), "")
		layout.CreatedBy = uuid.New()
		layout.UpdatedBy = layout.CreatedBy
		svc.CreateLayout(ctx, layout)
	}

	layouts, err := svc.GetLayouts(ctx, site.ID)
	if err != nil {
		t.Fatalf("GetLayouts() error = %v", err)
	}
	if len(layouts) != 3 {
		t.Errorf("GetLayouts() returned %d layouts, want 3", len(layouts))
	}
}

func TestServiceUpdateLayout(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Update Layout Site", "update-layout-site")

	layout := NewLayout(site.ID, "Original", "Original desc")
	layout.CreatedBy = uuid.New()
	layout.UpdatedBy = layout.CreatedBy
	svc.CreateLayout(ctx, layout)

	layout.Name = "Updated"
	layout.UpdatedAt = time.Now()

	if err := svc.UpdateLayout(ctx, layout); err != nil {
		t.Errorf("UpdateLayout() error = %v", err)
	}
}

func TestServiceDeleteLayout(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Delete Layout Site", "delete-layout-site")

	layout := NewLayout(site.ID, "To Delete", "")
	layout.CreatedBy = uuid.New()
	layout.UpdatedBy = layout.CreatedBy
	svc.CreateLayout(ctx, layout)

	if err := svc.DeleteLayout(ctx, layout.ID); err != nil {
		t.Errorf("DeleteLayout() error = %v", err)
	}

	_, err := svc.GetLayout(ctx, layout.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Error("Layout should have been deleted")
	}
}

func TestServiceCreateTag(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Tag Site", "tag-site")

	tag := NewTag(site.ID, "Technology")
	tag.CreatedBy = uuid.New()
	tag.UpdatedBy = tag.CreatedBy

	if err := svc.CreateTag(ctx, tag); err != nil {
		t.Errorf("CreateTag() error = %v", err)
	}
}

func TestServiceGetTag(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Get Tag Site", "get-tag-site")

	tag := NewTag(site.ID, "Test Tag")
	tag.CreatedBy = uuid.New()
	tag.UpdatedBy = tag.CreatedBy
	svc.CreateTag(ctx, tag)

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr error
	}{
		{
			name:    "existing tag",
			id:      tag.ID,
			wantErr: nil,
		},
		{
			name:    "non existent tag",
			id:      uuid.New(),
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.GetTag(ctx, tt.id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetTag() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestServiceGetTagByName(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Tag Name Site", "tag-name-site")

	tag := NewTag(site.ID, "JavaScript")
	tag.CreatedBy = uuid.New()
	tag.UpdatedBy = tag.CreatedBy
	svc.CreateTag(ctx, tag)

	tests := []struct {
		name    string
		tagName string
		wantErr error
	}{
		{
			name:    "existing tag name",
			tagName: "JavaScript",
			wantErr: nil,
		},
		{
			name:    "non existent tag name",
			tagName: "Python",
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.GetTagByName(ctx, site.ID, tt.tagName)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetTagByName() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestServiceGetTags(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Tags Site", "tags-site")

	tagNames := []string{"Go", "Rust", "Python"}
	for _, name := range tagNames {
		tag := NewTag(site.ID, name)
		tag.CreatedBy = uuid.New()
		tag.UpdatedBy = tag.CreatedBy
		svc.CreateTag(ctx, tag)
	}

	tags, err := svc.GetTags(ctx, site.ID)
	if err != nil {
		t.Fatalf("GetTags() error = %v", err)
	}
	if len(tags) != 3 {
		t.Errorf("GetTags() returned %d tags, want 3", len(tags))
	}
}

func TestServiceAddTagToContent(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Add Tag Site", "add-tag-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	content := NewContent(site.ID, section.ID, "Tagged Post", "Body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	tag := NewTag(site.ID, "Existing")
	tag.CreatedBy = uuid.New()
	tag.UpdatedBy = tag.CreatedBy
	svc.CreateTag(ctx, tag)

	if err := svc.AddTagToContent(ctx, content.ID, "Existing", site.ID); err != nil {
		t.Errorf("AddTagToContent() error = %v", err)
	}

	if err := svc.AddTagToContent(ctx, content.ID, "NewTag", site.ID); err != nil {
		t.Errorf("AddTagToContent() with new tag error = %v", err)
	}

	tags, _ := svc.GetTagsForContent(ctx, content.ID)
	if len(tags) != 2 {
		t.Errorf("Content should have 2 tags, got %d", len(tags))
	}
}

func TestServiceAddTagToContentByID(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Add Tag ID Site", "add-tag-id-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	content := NewContent(site.ID, section.ID, "Tagged Post", "Body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	tag := NewTag(site.ID, "ByID")
	tag.CreatedBy = uuid.New()
	tag.UpdatedBy = tag.CreatedBy
	svc.CreateTag(ctx, tag)

	if err := svc.AddTagToContentByID(ctx, content.ID, tag.ID); err != nil {
		t.Errorf("AddTagToContentByID() error = %v", err)
	}
}

func TestServiceRemoveTagFromContent(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Remove Tag Site", "remove-tag-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	content := NewContent(site.ID, section.ID, "Tagged Post", "Body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	tag := NewTag(site.ID, "ToRemove")
	tag.CreatedBy = uuid.New()
	tag.UpdatedBy = tag.CreatedBy
	svc.CreateTag(ctx, tag)
	svc.AddTagToContentByID(ctx, content.ID, tag.ID)

	if err := svc.RemoveTagFromContent(ctx, content.ID, tag.ID); err != nil {
		t.Errorf("RemoveTagFromContent() error = %v", err)
	}

	tags, _ := svc.GetTagsForContent(ctx, content.ID)
	if len(tags) != 0 {
		t.Errorf("Content should have 0 tags, got %d", len(tags))
	}
}

func TestServiceRemoveAllTagsFromContent(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Remove All Tags Site", "remove-all-tags-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	content := NewContent(site.ID, section.ID, "Tagged Post", "Body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	for i := 0; i < 3; i++ {
		svc.AddTagToContent(ctx, content.ID, "Tag"+string(rune('A'+i)), site.ID)
	}

	if err := svc.RemoveAllTagsFromContent(ctx, content.ID); err != nil {
		t.Errorf("RemoveAllTagsFromContent() error = %v", err)
	}

	tags, _ := svc.GetTagsForContent(ctx, content.ID)
	if len(tags) != 0 {
		t.Errorf("Content should have 0 tags, got %d", len(tags))
	}
}

func TestServiceCreateSetting(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Setting Site", "setting-site")

	setting := NewSetting(site.ID, "site_title", "My Site")
	setting.CreatedBy = uuid.New()
	setting.UpdatedBy = setting.CreatedBy

	if err := svc.CreateSetting(ctx, setting); err != nil {
		t.Errorf("CreateSetting() error = %v", err)
	}
}

func TestServiceGetSetting(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Get Setting Site", "get-setting-site")

	setting := NewSetting(site.ID, "test_param", "test_value")
	setting.CreatedBy = uuid.New()
	setting.UpdatedBy = setting.CreatedBy
	svc.CreateSetting(ctx, setting)

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr error
	}{
		{
			name:    "existing setting",
			id:      setting.ID,
			wantErr: nil,
		},
		{
			name:    "non existent setting",
			id:      uuid.New(),
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.GetSetting(ctx, tt.id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetSetting() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestServiceGetSettingByName(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Setting Name Site", "setting-name-site")

	setting := NewSetting(site.ID, "site_title", "My Site")
	setting.CreatedBy = uuid.New()
	setting.UpdatedBy = setting.CreatedBy
	svc.CreateSetting(ctx, setting)

	tests := []struct {
		name      string
		paramName string
		wantErr   error
	}{
		{
			name:      "existing param",
			paramName: "site_title",
			wantErr:   nil,
		},
		{
			name:      "non existent param",
			paramName: "not_found",
			wantErr:   ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.GetSettingByName(ctx, site.ID, tt.paramName)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetSettingByName() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestServiceGetSettingByRefKey(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Setting RefKey Site", "setting-refkey-site")

	setting := NewSetting(site.ID, "API Token", "secret123")
	setting.RefKey = "api_token"
	setting.CreatedBy = uuid.New()
	setting.UpdatedBy = setting.CreatedBy
	svc.CreateSetting(ctx, setting)

	tests := []struct {
		name    string
		refKey  string
		wantErr error
	}{
		{
			name:    "existing refKey",
			refKey:  "api_token",
			wantErr: nil,
		},
		{
			name:    "non existent refKey",
			refKey:  "not_found",
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.GetSettingByRefKey(ctx, site.ID, tt.refKey)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetSettingByRefKey() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestServiceGetSettings(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Settings Site", "settings-site")

	for i := 0; i < 3; i++ {
		setting := NewSetting(site.ID, "param_"+string(rune('a'+i)), "value")
		setting.CreatedBy = uuid.New()
		setting.UpdatedBy = setting.CreatedBy
		svc.CreateSetting(ctx, setting)
	}

	settings, err := svc.GetSettings(ctx, site.ID)
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	if len(settings) != 3 {
		t.Errorf("GetSettings() returned %d settings, want 3", len(settings))
	}
}

func TestServiceCreateImage(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Image Site", "image-site")

	image := NewImage(site.ID, "photo.jpg", "/images/photo.jpg")
	image.AltText = "A photo"
	image.Width = 800
	image.Height = 600
	image.CreatedBy = uuid.New()
	image.UpdatedBy = image.CreatedBy

	if err := svc.CreateImage(ctx, image); err != nil {
		t.Errorf("CreateImage() error = %v", err)
	}
}

func TestServiceGetImage(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Get Image Site", "get-image-site")

	image := NewImage(site.ID, "test.jpg", "/images/test.jpg")
	image.CreatedBy = uuid.New()
	image.UpdatedBy = image.CreatedBy
	svc.CreateImage(ctx, image)

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr error
	}{
		{
			name:    "existing image",
			id:      image.ID,
			wantErr: nil,
		},
		{
			name:    "non existent image",
			id:      uuid.New(),
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.GetImage(ctx, tt.id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetImage() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestServiceGetImages(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Images Site", "images-site")

	for i := 0; i < 3; i++ {
		image := NewImage(site.ID, "image"+string(rune('a'+i))+".jpg", "/images/image"+string(rune('a'+i))+".jpg")
		image.CreatedBy = uuid.New()
		image.UpdatedBy = image.CreatedBy
		svc.CreateImage(ctx, image)
	}

	images, err := svc.GetImages(ctx, site.ID)
	if err != nil {
		t.Fatalf("GetImages() error = %v", err)
	}
	if len(images) != 3 {
		t.Errorf("GetImages() returned %d images, want 3", len(images))
	}
}

func TestServiceLinkImageToContent(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Link Image Site", "link-image-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	content := NewContent(site.ID, section.ID, "Image Post", "Body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	image := NewImage(site.ID, "header.jpg", "/images/header.jpg")
	image.CreatedBy = uuid.New()
	image.UpdatedBy = image.CreatedBy
	svc.CreateImage(ctx, image)

	if err := svc.LinkImageToContent(ctx, content.ID, image.ID, true); err != nil {
		t.Errorf("LinkImageToContent() error = %v", err)
	}

	images, err := svc.GetContentImagesWithDetails(ctx, content.ID)
	if err != nil {
		t.Errorf("GetContentImagesWithDetails() error = %v", err)
	}
	if len(images) != 1 {
		t.Errorf("Expected 1 image, got %d", len(images))
	}
	if !images[0].IsHeader {
		t.Error("Image should be marked as header")
	}
}

func TestServiceUnlinkImageFromContent(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Unlink Image Site", "unlink-image-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	content := NewContent(site.ID, section.ID, "Image Post", "Body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	image := NewImage(site.ID, "photo.jpg", "/images/photo.jpg")
	image.CreatedBy = uuid.New()
	image.UpdatedBy = image.CreatedBy
	svc.CreateImage(ctx, image)

	svc.LinkImageToContent(ctx, content.ID, image.ID, false)

	images, _ := svc.GetContentImagesWithDetails(ctx, content.ID)
	if len(images) != 1 {
		t.Fatalf("Expected 1 image before unlink")
	}

	if err := svc.UnlinkImageFromContent(ctx, images[0].ContentImageID); err != nil {
		t.Errorf("UnlinkImageFromContent() error = %v", err)
	}

	images, _ = svc.GetContentImagesWithDetails(ctx, content.ID)
	if len(images) != 0 {
		t.Errorf("Expected 0 images after unlink, got %d", len(images))
	}
}

func TestServiceCreateMeta(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Meta Site", "meta-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	content := NewContent(site.ID, section.ID, "Meta Post", "Body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	meta := NewMeta(site.ID, content.ID)
	meta.Description = "Test description"
	meta.Keywords = "test,keywords"
	meta.CreatedBy = uuid.New()
	meta.UpdatedBy = meta.CreatedBy

	if err := svc.CreateMeta(ctx, meta); err != nil {
		t.Errorf("CreateMeta() error = %v", err)
	}
}

func TestServiceGetMetaByContentID(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Get Meta Site", "get-meta-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	content := NewContent(site.ID, section.ID, "Meta Post", "Body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	meta, err := svc.GetMetaByContentID(ctx, content.ID)
	if err != nil {
		t.Errorf("GetMetaByContentID() error = %v", err)
	}
	if meta != nil {
		t.Error("Expected nil meta for content without meta")
	}

	newMeta := NewMeta(site.ID, content.ID)
	newMeta.Description = "Description"
	newMeta.CreatedBy = uuid.New()
	newMeta.UpdatedBy = newMeta.CreatedBy
	svc.CreateMeta(ctx, newMeta)

	meta, err = svc.GetMetaByContentID(ctx, content.ID)
	if err != nil {
		t.Errorf("GetMetaByContentID() error = %v", err)
	}
	if meta == nil {
		t.Error("Expected meta, got nil")
	}
}

func TestServiceUpdateMeta(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Update Meta Site", "update-meta-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	content := NewContent(site.ID, section.ID, "Meta Post", "Body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	meta := NewMeta(site.ID, content.ID)
	meta.Description = "Original"
	meta.CreatedBy = uuid.New()
	meta.UpdatedBy = meta.CreatedBy
	svc.CreateMeta(ctx, meta)

	meta.Description = "Updated"
	meta.UpdatedAt = time.Now()

	if err := svc.UpdateMeta(ctx, meta); err != nil {
		t.Errorf("UpdateMeta() error = %v", err)
	}
}

func TestServiceCreateContributor(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Contributor Site", "contributor-site")

	contributor := NewContributor(site.ID, "johndoe", "John", "Doe")
	contributor.Bio = "Software developer"
	contributor.SocialLinks = []SocialLink{
		{Platform: "twitter", Handle: "@johndoe"},
	}
	contributor.CreatedBy = uuid.New()
	contributor.UpdatedBy = contributor.CreatedBy

	if err := svc.CreateContributor(ctx, contributor); err != nil {
		t.Errorf("CreateContributor() error = %v", err)
	}
}

func TestServiceGetContributor(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Get Contributor Site", "get-contributor-site")

	contributor := NewContributor(site.ID, "janedoe", "Jane", "Doe")
	contributor.CreatedBy = uuid.New()
	contributor.UpdatedBy = contributor.CreatedBy
	svc.CreateContributor(ctx, contributor)

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr error
	}{
		{
			name:    "existing contributor",
			id:      contributor.ID,
			wantErr: nil,
		},
		{
			name:    "non existent contributor",
			id:      uuid.New(),
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.GetContributor(ctx, tt.id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetContributor() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestServiceGetContributors(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Contributors Site", "contributors-site")

	for i := 0; i < 3; i++ {
		contributor := NewContributor(site.ID, "user"+string(rune('a'+i)), "User", string(rune('A'+i)))
		contributor.CreatedBy = uuid.New()
		contributor.UpdatedBy = contributor.CreatedBy
		svc.CreateContributor(ctx, contributor)
	}

	contributors, err := svc.GetContributors(ctx, site.ID)
	if err != nil {
		t.Fatalf("GetContributors() error = %v", err)
	}
	if len(contributors) != 3 {
		t.Errorf("GetContributors() returned %d contributors, want 3", len(contributors))
	}
}

func TestServiceUpdateContributor(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Update Contributor Site", "update-contributor-site")

	contributor := NewContributor(site.ID, "updateuser", "Original", "Name")
	contributor.CreatedBy = uuid.New()
	contributor.UpdatedBy = contributor.CreatedBy
	svc.CreateContributor(ctx, contributor)

	contributor.Name = "Updated"
	contributor.Bio = "New bio"
	contributor.UpdatedAt = time.Now()

	if err := svc.UpdateContributor(ctx, contributor); err != nil {
		t.Errorf("UpdateContributor() error = %v", err)
	}
}

func TestServiceDeleteContributor(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Delete Contributor Site", "delete-contributor-site")

	contributor := NewContributor(site.ID, "deleteuser", "Delete", "Me")
	contributor.CreatedBy = uuid.New()
	contributor.UpdatedBy = contributor.CreatedBy
	svc.CreateContributor(ctx, contributor)

	if err := svc.DeleteContributor(ctx, contributor.ID); err != nil {
		t.Errorf("DeleteContributor() error = %v", err)
	}

	_, err := svc.GetContributor(ctx, contributor.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Error("Contributor should have been deleted")
	}
}

func TestServiceSetContributorProfile(t *testing.T) {
	svc, db, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Profile Contributor Site", "profile-contributor-site")

	contributor := NewContributor(site.ID, "profileuser", "Profile", "User")
	contributor.CreatedBy = uuid.New()
	contributor.UpdatedBy = contributor.CreatedBy
	svc.CreateContributor(ctx, contributor)

	// Create a real profile in the database
	profileID := uuid.New()
	_, err := db.Exec(`INSERT INTO profile (id, short_id, slug, name, created_by, updated_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		profileID.String(), "abc12345", "test-profile", "Test Profile", contributor.CreatedBy.String(), contributor.CreatedBy.String())
	if err != nil {
		t.Fatalf("Failed to create test profile: %v", err)
	}

	if err := svc.SetContributorProfile(ctx, contributor.ID, profileID, "admin"); err != nil {
		t.Errorf("SetContributorProfile() error = %v", err)
	}
}

func TestServiceUpdateTag(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Update Tag Site", "update-tag-site")

	tag := NewTag(site.ID, "Original Tag")
	tag.CreatedBy = uuid.New()
	tag.UpdatedBy = tag.CreatedBy
	svc.CreateTag(ctx, tag)

	tag.Name = "Updated Tag"
	tag.Slug = Slugify("Updated Tag")
	tag.UpdatedAt = time.Now()

	if err := svc.UpdateTag(ctx, tag); err != nil {
		t.Errorf("UpdateTag() error = %v", err)
	}
}

func TestServiceDeleteTag(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Delete Tag Site", "delete-tag-site")

	tag := NewTag(site.ID, "To Delete")
	tag.CreatedBy = uuid.New()
	tag.UpdatedBy = tag.CreatedBy
	svc.CreateTag(ctx, tag)

	if err := svc.DeleteTag(ctx, tag.ID); err != nil {
		t.Errorf("DeleteTag() error = %v", err)
	}

	_, err := svc.GetTag(ctx, tag.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Error("Tag should have been deleted")
	}
}

func TestServiceUpdateSetting(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Update Setting Site", "update-setting-site")

	setting := NewSetting(site.ID, "original_param", "original_value")
	setting.CreatedBy = uuid.New()
	setting.UpdatedBy = setting.CreatedBy
	svc.CreateSetting(ctx, setting)

	setting.Value = "updated_value"
	setting.UpdatedAt = time.Now()

	if err := svc.UpdateSetting(ctx, setting); err != nil {
		t.Errorf("UpdateSetting() error = %v", err)
	}
}

func TestServiceDeleteSetting(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Delete Setting Site", "delete-setting-site")

	setting := NewSetting(site.ID, "to_delete", "value")
	setting.CreatedBy = uuid.New()
	setting.UpdatedBy = setting.CreatedBy
	svc.CreateSetting(ctx, setting)

	if err := svc.DeleteSetting(ctx, setting.ID); err != nil {
		t.Errorf("DeleteSetting() error = %v", err)
	}

	_, err := svc.GetSetting(ctx, setting.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Error("Setting should have been deleted")
	}
}

func TestServiceUpdateImage(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Update Image Site", "update-image-site")

	image := NewImage(site.ID, "original.jpg", "/images/original.jpg")
	image.CreatedBy = uuid.New()
	image.UpdatedBy = image.CreatedBy
	svc.CreateImage(ctx, image)

	image.FileName = "updated.jpg"
	image.FilePath = "/images/updated.jpg"
	image.AltText = "Updated alt"
	image.UpdatedAt = time.Now()

	if err := svc.UpdateImage(ctx, image); err != nil {
		t.Errorf("UpdateImage() error = %v", err)
	}
}

func TestServiceDeleteImage(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Delete Image Site", "delete-image-site")

	image := NewImage(site.ID, "todelete.jpg", "/images/todelete.jpg")
	image.CreatedBy = uuid.New()
	image.UpdatedBy = image.CreatedBy
	svc.CreateImage(ctx, image)

	if err := svc.DeleteImage(ctx, image.ID); err != nil {
		t.Errorf("DeleteImage() error = %v", err)
	}

	_, err := svc.GetImage(ctx, image.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Error("Image should have been deleted")
	}
}

func TestServiceSectionImageOperations(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Section Image Site", "section-image-site")

	section := NewSection(site.ID, "Test Section", "", "/test")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	image := NewImage(site.ID, "section-img.jpg", "/images/section-img.jpg")
	image.CreatedBy = uuid.New()
	image.UpdatedBy = image.CreatedBy
	svc.CreateImage(ctx, image)

	// Link image to section
	if err := svc.LinkImageToSection(ctx, section.ID, image.ID, true); err != nil {
		t.Errorf("LinkImageToSection() error = %v", err)
	}

	// Get section images
	images, err := svc.GetSectionImagesWithDetails(ctx, section.ID)
	if err != nil {
		t.Errorf("GetSectionImagesWithDetails() error = %v", err)
	}
	if len(images) != 1 {
		t.Errorf("Expected 1 image, got %d", len(images))
	}

	// Get section image details
	if len(images) > 0 {
		details, err := svc.GetSectionImageDetails(ctx, images[0].SectionImageID)
		if err != nil {
			t.Errorf("GetSectionImageDetails() error = %v", err)
		}
		if details == nil {
			t.Error("Expected details, got nil")
		}

		// Unlink image from section
		if err := svc.UnlinkImageFromSection(ctx, images[0].SectionImageID); err != nil {
			t.Errorf("UnlinkImageFromSection() error = %v", err)
		}
	}

	// Verify unlink
	images, _ = svc.GetSectionImagesWithDetails(ctx, section.ID)
	if len(images) != 0 {
		t.Errorf("Expected 0 images after unlink, got %d", len(images))
	}
}

func TestServiceGetContentImageDetails(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Content Image Details Site", "content-image-details-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	content := NewContent(site.ID, section.ID, "Image Post", "Body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	image := NewImage(site.ID, "detail.jpg", "/images/detail.jpg")
	image.CreatedBy = uuid.New()
	image.UpdatedBy = image.CreatedBy
	svc.CreateImage(ctx, image)

	svc.LinkImageToContent(ctx, content.ID, image.ID, false)

	images, _ := svc.GetContentImagesWithDetails(ctx, content.ID)
	if len(images) != 1 {
		t.Fatalf("Expected 1 image")
	}

	details, err := svc.GetContentImageDetails(ctx, images[0].ContentImageID)
	if err != nil {
		t.Errorf("GetContentImageDetails() error = %v", err)
	}
	if details == nil {
		t.Error("Expected details, got nil")
	}
	if details.FilePath != "/images/detail.jpg" {
		t.Errorf("FilePath = %q, want %q", details.FilePath, "/images/detail.jpg")
	}
}

func TestServiceUnlinkHeaderImageFromContent(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Unlink Header Site", "unlink-header-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	content := NewContent(site.ID, section.ID, "Header Post", "Body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	// Add a header image
	headerImage := NewImage(site.ID, "header.jpg", "/images/header.jpg")
	headerImage.CreatedBy = uuid.New()
	headerImage.UpdatedBy = headerImage.CreatedBy
	svc.CreateImage(ctx, headerImage)
	svc.LinkImageToContent(ctx, content.ID, headerImage.ID, true)

	// Add a non-header image
	otherImage := NewImage(site.ID, "other.jpg", "/images/other.jpg")
	otherImage.CreatedBy = uuid.New()
	otherImage.UpdatedBy = otherImage.CreatedBy
	svc.CreateImage(ctx, otherImage)
	svc.LinkImageToContent(ctx, content.ID, otherImage.ID, false)

	// Unlink header
	if err := svc.UnlinkHeaderImageFromContent(ctx, content.ID); err != nil {
		t.Errorf("UnlinkHeaderImageFromContent() error = %v", err)
	}

	// Verify only non-header remains
	images, _ := svc.GetContentImagesWithDetails(ctx, content.ID)
	if len(images) != 1 {
		t.Errorf("Expected 1 image after unlink, got %d", len(images))
	}
	if len(images) > 0 && images[0].IsHeader {
		t.Error("Remaining image should not be header")
	}
}

func TestServiceGetAllContentWithMeta(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "All Content Meta Site", "all-content-meta-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	for i := 0; i < 3; i++ {
		content := NewContent(site.ID, section.ID, "Post "+string(rune('A'+i)), "Body")
		content.CreatedBy = uuid.New()
		content.UpdatedBy = content.CreatedBy
		svc.CreateContent(ctx, content)
	}

	contents, err := svc.GetAllContentWithMeta(ctx, site.ID)
	if err != nil {
		t.Errorf("GetAllContentWithMeta() error = %v", err)
	}
	if len(contents) != 3 {
		t.Errorf("Expected 3 contents, got %d", len(contents))
	}
}

func TestServiceBuildUserAuthorsMap(t *testing.T) {
	svc, db, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Create a profile first
	userID := uuid.New()
	profileID := uuid.New()
	_, err := db.Exec(`INSERT INTO profile (id, short_id, slug, name, surname, bio, photo_path, created_by, updated_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		profileID.String(), "p123", "author-user", "Author", "User", "Bio text", "/photos/author.jpg", userID.String(), userID.String())
	if err != nil {
		t.Fatalf("Failed to create profile: %v", err)
	}

	// Create a user with profile_id reference
	_, err = db.Exec(`INSERT INTO user (id, short_id, email, password_hash, name, status, roles, profile_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		userID.String(), "u123", "author@test.com", "hash", "authoruser", "active", "editor", profileID.String())
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	site := createTestSite(t, svc, "Authors Site", "authors-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	// Content with user author
	content := NewContent(site.ID, section.ID, "Post 1", "Body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	content.AuthorUsername = "authoruser"
	svc.CreateContent(ctx, content)

	// Content with non-existent user
	content2 := NewContent(site.ID, section.ID, "Post 2", "Body")
	content2.CreatedBy = uuid.New()
	content2.UpdatedBy = content2.CreatedBy
	content2.AuthorUsername = "unknownuser"
	svc.CreateContent(ctx, content2)

	// Contributor
	contributor := NewContributor(site.ID, "contrib1", "Contrib", "One")
	contributor.CreatedBy = uuid.New()
	contributor.UpdatedBy = contributor.CreatedBy
	svc.CreateContributor(ctx, contributor)

	// Content with contributor handle (should be excluded)
	content3 := NewContent(site.ID, section.ID, "Post 3", "Body")
	content3.CreatedBy = uuid.New()
	content3.UpdatedBy = content3.CreatedBy
	content3.AuthorUsername = "contrib1"
	svc.CreateContent(ctx, content3)

	contents := []*Content{content, content2, content3}
	contributors := []*Contributor{contributor}

	authorsMap := svc.BuildUserAuthorsMap(ctx, contents, contributors)

	// Should have 2 entries: authoruser (with profile) and unknownuser (fallback)
	if len(authorsMap) != 2 {
		t.Errorf("Expected 2 authors in map, got %d", len(authorsMap))
	}

	// Check authoruser has profile info
	author, ok := authorsMap["authoruser"]
	if !ok {
		t.Error("Expected authoruser in map")
	} else {
		if author.Name != "Author" {
			t.Errorf("Expected name 'Author', got %q", author.Name)
		}
		if author.Surname != "User" {
			t.Errorf("Expected surname 'User', got %q", author.Surname)
		}
		if author.Bio != "Bio text" {
			t.Errorf("Expected bio 'Bio text', got %q", author.Bio)
		}
	}

	// Check unknownuser has fallback
	unknown, ok := authorsMap["unknownuser"]
	if !ok {
		t.Error("Expected unknownuser in map")
	} else {
		if unknown.Name != "unknownuser" {
			t.Errorf("Expected fallback name 'unknownuser', got %q", unknown.Name)
		}
	}

	// contrib1 should NOT be in map (it's a contributor handle)
	if _, ok := authorsMap["contrib1"]; ok {
		t.Error("contrib1 should not be in map as it's a contributor handle")
	}
}

func TestServiceBuildUserAuthorsMapEmpty(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	authorsMap := svc.BuildUserAuthorsMap(ctx, nil, nil)

	if len(authorsMap) != 0 {
		t.Errorf("Expected empty map, got %d entries", len(authorsMap))
	}
}

func TestServiceBuildUserAuthorsMapNoUserAuthors(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "No Authors Site", "no-authors-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	// Content without author username
	content := NewContent(site.ID, section.ID, "Post", "Body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	contents := []*Content{content}
	authorsMap := svc.BuildUserAuthorsMap(ctx, contents, nil)

	if len(authorsMap) != 0 {
		t.Errorf("Expected empty map, got %d entries", len(authorsMap))
	}
}

func TestServiceAddTagToContentCreatesTag(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Tag Create Site", "tag-create-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	content := NewContent(site.ID, section.ID, "Post", "Body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	// Add non-existent tag by name - should create the tag
	err := svc.AddTagToContent(ctx, content.ID, "newtag", site.ID)
	if err != nil {
		t.Errorf("AddTagToContent() should create tag if not exists, got error: %v", err)
	}

	// Verify tag was created
	tags, err := svc.GetTagsForContent(ctx, content.ID)
	if err != nil {
		t.Errorf("GetTagsForContent() error = %v", err)
	}
	if len(tags) != 1 {
		t.Errorf("Expected 1 tag, got %d", len(tags))
	}
	if len(tags) > 0 && tags[0].Name != "newtag" {
		t.Errorf("Expected tag name 'newtag', got %q", tags[0].Name)
	}
}

func TestServiceContributorCRUD(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Contrib CRUD Site", "contrib-crud-site")

	// Create contributor with all fields
	contributor := NewContributor(site.ID, "testcontrib", "Test", "Contributor")
	contributor.Bio = "A test contributor"
	contributor.Role = "author"
	contributor.PhotoPath = "/photos/test.jpg"
	contributor.CreatedBy = uuid.New()
	contributor.UpdatedBy = contributor.CreatedBy

	err := svc.CreateContributor(ctx, contributor)
	if err != nil {
		t.Fatalf("CreateContributor() error = %v", err)
	}

	// Get contributor
	got, err := svc.GetContributor(ctx, contributor.ID)
	if err != nil {
		t.Errorf("GetContributor() error = %v", err)
	}
	if got.Handle != "testcontrib" {
		t.Errorf("Handle = %q, want %q", got.Handle, "testcontrib")
	}
	if got.Bio != "A test contributor" {
		t.Errorf("Bio = %q, want %q", got.Bio, "A test contributor")
	}

	// Update contributor
	got.Name = "Updated"
	got.Surname = "Name"
	got.Bio = "Updated bio"
	err = svc.UpdateContributor(ctx, got)
	if err != nil {
		t.Errorf("UpdateContributor() error = %v", err)
	}

	// Verify update
	updated, err := svc.GetContributor(ctx, contributor.ID)
	if err != nil {
		t.Errorf("GetContributor() after update error = %v", err)
	}
	if updated.Name != "Updated" {
		t.Errorf("Name = %q, want %q", updated.Name, "Updated")
	}
	if updated.Bio != "Updated bio" {
		t.Errorf("Bio = %q, want %q", updated.Bio, "Updated bio")
	}

	// List contributors
	contributors, err := svc.GetContributors(ctx, site.ID)
	if err != nil {
		t.Errorf("GetContributors() error = %v", err)
	}
	if len(contributors) != 1 {
		t.Errorf("Expected 1 contributor, got %d", len(contributors))
	}

	// Delete contributor
	err = svc.DeleteContributor(ctx, contributor.ID)
	if err != nil {
		t.Errorf("DeleteContributor() error = %v", err)
	}

	// Verify deletion
	_, err = svc.GetContributor(ctx, contributor.ID)
	if err == nil {
		t.Error("Expected error when getting deleted contributor")
	}
}

func TestServiceGetContributorNotFound(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	_, err := svc.GetContributor(ctx, uuid.New())
	if err == nil {
		t.Error("Expected error when getting non-existent contributor")
	}
}

func TestServiceMultipleContributors(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Multi Contrib Site", "multi-contrib-site")

	// Create multiple contributors
	for i := 0; i < 3; i++ {
		contributor := NewContributor(site.ID, "contrib"+string(rune('1'+i)), "Contrib", string(rune('A'+i)))
		contributor.CreatedBy = uuid.New()
		contributor.UpdatedBy = contributor.CreatedBy
		if err := svc.CreateContributor(ctx, contributor); err != nil {
			t.Fatalf("CreateContributor() error = %v", err)
		}
	}

	contributors, err := svc.GetContributors(ctx, site.ID)
	if err != nil {
		t.Errorf("GetContributors() error = %v", err)
	}
	if len(contributors) != 3 {
		t.Errorf("Expected 3 contributors, got %d", len(contributors))
	}
}

func TestServiceCreateContentWithContributor(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Content Contrib Site", "content-contrib-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	// Create contributor
	contributor := NewContributor(site.ID, "author1", "Author", "One")
	contributor.CreatedBy = uuid.New()
	contributor.UpdatedBy = contributor.CreatedBy
	svc.CreateContributor(ctx, contributor)

	// Create content with contributor
	content := NewContent(site.ID, section.ID, "Authored Post", "Body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	content.ContributorID = &contributor.ID
	content.ContributorHandle = contributor.Handle

	err := svc.CreateContent(ctx, content)
	if err != nil {
		t.Fatalf("CreateContent() with contributor error = %v", err)
	}

	// Get content with meta (which includes contributor info)
	got, err := svc.GetContentWithMeta(ctx, content.ID)
	if err != nil {
		t.Errorf("GetContentWithMeta() error = %v", err)
	}
	if got.ContributorID == nil {
		t.Error("ContributorID should not be nil")
	} else if *got.ContributorID != contributor.ID {
		t.Errorf("ContributorID = %v, want %v", *got.ContributorID, contributor.ID)
	}
}

func TestServiceUpdateContentFields(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Update Content Site", "update-content-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	content := NewContent(site.ID, section.ID, "Original Title", "Original Body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	// Update multiple fields
	content.Heading = "Updated Title"
	content.Body = "Updated Body"
	content.Summary = "A summary"
	content.Draft = false
	content.Featured = true
	content.Series = "test-series"
	content.SeriesOrder = 1
	now := time.Now()
	content.PublishedAt = &now

	err := svc.UpdateContent(ctx, content)
	if err != nil {
		t.Fatalf("UpdateContent() error = %v", err)
	}

	// Verify all updates
	updated, err := svc.GetContent(ctx, content.ID)
	if err != nil {
		t.Errorf("GetContent() error = %v", err)
	}
	if updated.Heading != "Updated Title" {
		t.Errorf("Heading = %q, want %q", updated.Heading, "Updated Title")
	}
	if updated.Summary != "A summary" {
		t.Errorf("Summary = %q, want %q", updated.Summary, "A summary")
	}
	if updated.Featured != true {
		t.Error("Featured should be true")
	}
	if updated.Series != "test-series" {
		t.Errorf("Series = %q, want %q", updated.Series, "test-series")
	}
}

func TestServiceContributorWithSocialLinks(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Social Links Site", "social-links-site")

	// Create contributor with social links
	contributor := NewContributor(site.ID, "socialuser", "Social", "User")
	contributor.SocialLinks = []SocialLink{
		{Platform: "twitter", Handle: "@socialuser", URL: "https://twitter.com/socialuser"},
		{Platform: "github", Handle: "socialuser", URL: "https://github.com/socialuser"},
		{Platform: "linkedin", Handle: "socialuser", URL: "https://linkedin.com/in/socialuser"},
	}
	contributor.CreatedBy = uuid.New()
	contributor.UpdatedBy = contributor.CreatedBy

	err := svc.CreateContributor(ctx, contributor)
	if err != nil {
		t.Fatalf("CreateContributor() error = %v", err)
	}

	// Get contributor and verify social links
	got, err := svc.GetContributor(ctx, contributor.ID)
	if err != nil {
		t.Errorf("GetContributor() error = %v", err)
	}
	if len(got.SocialLinks) != 3 {
		t.Errorf("Expected 3 social links, got %d", len(got.SocialLinks))
	}
	if got.SocialLinks[0].Platform != "twitter" {
		t.Errorf("First social link platform = %q, want %q", got.SocialLinks[0].Platform, "twitter")
	}
}

func TestServiceContributorWithProfile(t *testing.T) {
	svc, db, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Contrib Profile Site", "contrib-profile-site")

	// Create a profile first
	profileID := uuid.New()
	creatorID := uuid.New()
	_, err := db.Exec(`INSERT INTO profile (id, short_id, slug, name, surname, bio, photo_path, social_links, created_by, updated_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		profileID.String(), "prof1234", "contrib-profile", "Profile", "Name", "Profile bio", "/photos/profile.jpg",
		`[{"platform":"twitter","handle":"@profile"}]`,
		creatorID.String(), creatorID.String())
	if err != nil {
		t.Fatalf("Failed to create profile: %v", err)
	}

	// Create contributor
	contributor := NewContributor(site.ID, "profcontrib", "Prof", "Contrib")
	contributor.CreatedBy = creatorID
	contributor.UpdatedBy = creatorID
	err = svc.CreateContributor(ctx, contributor)
	if err != nil {
		t.Fatalf("CreateContributor() error = %v", err)
	}

	// Set contributor profile
	err = svc.SetContributorProfile(ctx, contributor.ID, profileID, creatorID.String())
	if err != nil {
		t.Fatalf("SetContributorProfile() error = %v", err)
	}

	// List contributors (this uses contributorWithProfileFromSQLC)
	contributors, err := svc.GetContributors(ctx, site.ID)
	if err != nil {
		t.Errorf("GetContributors() error = %v", err)
	}
	if len(contributors) != 1 {
		t.Fatalf("Expected 1 contributor, got %d", len(contributors))
	}

	// Check that profile data was loaded
	c := contributors[0]
	if c.ProfileID == nil {
		t.Error("ProfileID should not be nil")
	}
	if c.PhotoPath != "/photos/profile.jpg" {
		t.Errorf("PhotoPath = %q, want %q", c.PhotoPath, "/photos/profile.jpg")
	}
}

func TestServiceGetContentWithMetaNotFound(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	_, err := svc.GetContentWithMeta(ctx, uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetContentWithMeta() for non-existent content should return ErrNotFound, got: %v", err)
	}
}

func TestServiceGetContentWithMetaWithTags(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Meta Tags Site", "meta-tags-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	content := NewContent(site.ID, section.ID, "Tagged Content", "Body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	// Add tags
	svc.AddTagToContent(ctx, content.ID, "tag1", site.ID)
	svc.AddTagToContent(ctx, content.ID, "tag2", site.ID)

	// Get with meta (should include tags)
	got, err := svc.GetContentWithMeta(ctx, content.ID)
	if err != nil {
		t.Fatalf("GetContentWithMeta() error = %v", err)
	}
	if len(got.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(got.Tags))
	}
}

func TestServiceUpdateContentWithContributor(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Update Contrib Site", "update-contrib-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	// Create contributor
	contributor := NewContributor(site.ID, "author", "Author", "Name")
	contributor.CreatedBy = uuid.New()
	contributor.UpdatedBy = contributor.CreatedBy
	svc.CreateContributor(ctx, contributor)

	// Create content without contributor
	content := NewContent(site.ID, section.ID, "Post", "Body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	// Update content to add contributor
	content.ContributorID = &contributor.ID
	content.ContributorHandle = contributor.Handle
	content.UpdatedAt = time.Now()

	err := svc.UpdateContent(ctx, content)
	if err != nil {
		t.Fatalf("UpdateContent() error = %v", err)
	}

	// Verify
	updated, err := svc.GetContentWithMeta(ctx, content.ID)
	if err != nil {
		t.Errorf("GetContentWithMeta() error = %v", err)
	}
	if updated.ContributorID == nil {
		t.Error("ContributorID should not be nil")
	}
	if updated.ContributorHandle != "author" {
		t.Errorf("ContributorHandle = %q, want %q", updated.ContributorHandle, "author")
	}
}

func TestServiceContributorSocialLinksUpdate(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Update Social Site", "update-social-site")

	// Create contributor without social links
	contributor := NewContributor(site.ID, "nosocial", "No", "Social")
	contributor.CreatedBy = uuid.New()
	contributor.UpdatedBy = contributor.CreatedBy
	svc.CreateContributor(ctx, contributor)

	// Update with social links
	contributor.SocialLinks = []SocialLink{
		{Platform: "website", Handle: "", URL: "https://example.com"},
	}
	contributor.UpdatedAt = time.Now()

	err := svc.UpdateContributor(ctx, contributor)
	if err != nil {
		t.Fatalf("UpdateContributor() error = %v", err)
	}

	// Verify
	updated, err := svc.GetContributor(ctx, contributor.ID)
	if err != nil {
		t.Errorf("GetContributor() error = %v", err)
	}
	if len(updated.SocialLinks) != 1 {
		t.Errorf("Expected 1 social link, got %d", len(updated.SocialLinks))
	}
}

func TestServiceUnlinkImageFromContentNotFound(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Try to unlink non-existent content image - should not error (SQL DELETE)
	err := svc.UnlinkImageFromContent(ctx, uuid.New())
	if err != nil {
		// Note: SQLite DELETE doesn't error on non-existent rows
		t.Logf("UnlinkImageFromContent() returned error for non-existent: %v", err)
	}
}

func TestServiceUnlinkImageFromSectionNotFound(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Try to unlink non-existent section image - should not error (SQL DELETE)
	err := svc.UnlinkImageFromSection(ctx, uuid.New())
	if err != nil {
		t.Logf("UnlinkImageFromSection() returned error for non-existent: %v", err)
	}
}

func TestServiceCreateContributorWithProfileID(t *testing.T) {
	svc, db, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Contrib PID Site", "contrib-pid-site")

	// Create a profile first
	profileID := uuid.New()
	creatorID := uuid.New()
	_, err := db.Exec(`INSERT INTO profile (id, short_id, slug, name, created_by, updated_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		profileID.String(), "pid12345", "pid-profile", "PID Profile", creatorID.String(), creatorID.String())
	if err != nil {
		t.Fatalf("Failed to create profile: %v", err)
	}

	// Create contributor with profile ID
	contributor := NewContributor(site.ID, "pidcontrib", "PID", "Contrib")
	contributor.ProfileID = &profileID
	contributor.CreatedBy = creatorID
	contributor.UpdatedBy = creatorID

	err = svc.CreateContributor(ctx, contributor)
	if err != nil {
		t.Fatalf("CreateContributor() error = %v", err)
	}

	// Verify profile ID was saved
	got, err := svc.GetContributor(ctx, contributor.ID)
	if err != nil {
		t.Errorf("GetContributor() error = %v", err)
	}
	if got.ProfileID == nil {
		t.Error("ProfileID should not be nil")
	} else if *got.ProfileID != profileID {
		t.Errorf("ProfileID = %v, want %v", *got.ProfileID, profileID)
	}
}

func TestServiceDeleteContributorVerify(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Del Contrib Site", "del-contrib-site")

	contributor := NewContributor(site.ID, "delme", "Del", "Me")
	contributor.CreatedBy = uuid.New()
	contributor.UpdatedBy = contributor.CreatedBy
	svc.CreateContributor(ctx, contributor)

	// Verify exists
	_, err := svc.GetContributor(ctx, contributor.ID)
	if err != nil {
		t.Fatalf("Contributor should exist: %v", err)
	}

	// Delete
	err = svc.DeleteContributor(ctx, contributor.ID)
	if err != nil {
		t.Fatalf("DeleteContributor() error = %v", err)
	}

	// Verify deleted
	_, err = svc.GetContributor(ctx, contributor.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound after delete, got: %v", err)
	}
}

func TestServiceContributorEmptySocialLinks(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Empty Social Site", "empty-social-site")

	// Create contributor with empty social links
	contributor := NewContributor(site.ID, "emptylinks", "Empty", "Links")
	contributor.SocialLinks = []SocialLink{}
	contributor.CreatedBy = uuid.New()
	contributor.UpdatedBy = contributor.CreatedBy

	err := svc.CreateContributor(ctx, contributor)
	if err != nil {
		t.Fatalf("CreateContributor() error = %v", err)
	}

	// Get and verify
	got, err := svc.GetContributor(ctx, contributor.ID)
	if err != nil {
		t.Errorf("GetContributor() error = %v", err)
	}
	if got.SocialLinks == nil {
		// Empty slice is acceptable
	}
}

func TestServiceSiteWithDefaultLayout(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Layout Site", "layout-site")

	// Create a layout
	layout := NewLayout(site.ID, "Default Layout", "The default layout")
	layout.Code = "<html>{{.Content}}</html>"
	layout.CreatedBy = uuid.New()
	layout.UpdatedBy = layout.CreatedBy
	svc.CreateLayout(ctx, layout)

	// Update site with default layout
	site.DefaultLayoutID = layout.ID
	site.DefaultLayoutName = layout.Name
	site.UpdatedAt = time.Now()

	err := svc.UpdateSite(ctx, site)
	if err != nil {
		t.Fatalf("UpdateSite() error = %v", err)
	}

	// Get site and verify default layout fields
	got, err := svc.GetSite(ctx, site.ID)
	if err != nil {
		t.Errorf("GetSite() error = %v", err)
	}
	if got.DefaultLayoutID != layout.ID {
		t.Errorf("DefaultLayoutID = %v, want %v", got.DefaultLayoutID, layout.ID)
	}
	if got.DefaultLayoutName != layout.Name {
		t.Errorf("DefaultLayoutName = %q, want %q", got.DefaultLayoutName, layout.Name)
	}
}

func TestServiceContentWithHeaderImage(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Header Img Site", "header-img-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	content := NewContent(site.ID, section.ID, "Header Image Post", "Body with header image")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	// Create and link header image
	image := NewImage(site.ID, "header.jpg", "posts/header.jpg")
	image.AltText = "Header image alt text"
	image.CreatedBy = uuid.New()
	image.UpdatedBy = image.CreatedBy
	svc.CreateImage(ctx, image)
	svc.LinkImageToContent(ctx, content.ID, image.ID, true)

	// Get all content with meta (tests contentWithMetaFromSQLCAll with header image)
	contents, err := svc.GetAllContentWithMeta(ctx, site.ID)
	if err != nil {
		t.Fatalf("GetAllContentWithMeta() error = %v", err)
	}
	if len(contents) != 1 {
		t.Fatalf("Expected 1 content, got %d", len(contents))
	}

	c := contents[0]
	if c.HeaderImageURL == "" {
		t.Error("HeaderImageURL should not be empty")
	}
	if c.HeaderImageAlt != "Header image alt text" {
		t.Errorf("HeaderImageAlt = %q, want %q", c.HeaderImageAlt, "Header image alt text")
	}
}

func TestServiceMetaAllFields(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Meta All Fields Site", "meta-all-fields-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	content := NewContent(site.ID, section.ID, "Full Meta Post", "Body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	// Create meta with all fields
	meta := NewMeta(site.ID, content.ID)
	meta.Summary = "Test summary"
	meta.Excerpt = "Test excerpt"
	meta.Description = "Test description"
	meta.Keywords = "test,keywords"
	meta.Robots = "index,follow"
	meta.CanonicalURL = "https://example.com/post"
	meta.Sitemap = "weekly"
	meta.TableOfContents = true
	meta.Share = true
	meta.Comments = true
	meta.CreatedBy = uuid.New()
	meta.UpdatedBy = meta.CreatedBy

	err := svc.CreateMeta(ctx, meta)
	if err != nil {
		t.Fatalf("CreateMeta() error = %v", err)
	}

	// Get meta
	got, err := svc.GetMetaByContentID(ctx, content.ID)
	if err != nil {
		t.Fatalf("GetMetaByContentID() error = %v", err)
	}
	if got.Summary != "Test summary" {
		t.Errorf("Summary = %q, want %q", got.Summary, "Test summary")
	}
	if got.Excerpt != "Test excerpt" {
		t.Errorf("Excerpt = %q, want %q", got.Excerpt, "Test excerpt")
	}
	if got.Description != "Test description" {
		t.Errorf("Description = %q, want %q", got.Description, "Test description")
	}
	if got.Keywords != "test,keywords" {
		t.Errorf("Keywords = %q, want %q", got.Keywords, "test,keywords")
	}
	if got.Robots != "index,follow" {
		t.Errorf("Robots = %q, want %q", got.Robots, "index,follow")
	}
	if got.CanonicalURL != "https://example.com/post" {
		t.Errorf("CanonicalURL = %q, want %q", got.CanonicalURL, "https://example.com/post")
	}
	if !got.TableOfContents {
		t.Error("TableOfContents should be true")
	}
	if !got.Share {
		t.Error("Share should be true")
	}
	if !got.Comments {
		t.Error("Comments should be true")
	}
}

func TestServiceLayoutWithAllFields(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Layout Fields Site", "layout-fields-site")

	// Create image for header
	image := NewImage(site.ID, "layout-header.jpg", "layouts/header.jpg")
	image.CreatedBy = uuid.New()
	image.UpdatedBy = image.CreatedBy
	svc.CreateImage(ctx, image)

	// Create layout with all fields
	layout := NewLayout(site.ID, "Full Layout", "Layout with all fields")
	layout.Code = "<html><body>{{.Content}}</body></html>"
	layout.HeaderImageID = image.ID
	layout.CreatedBy = uuid.New()
	layout.UpdatedBy = layout.CreatedBy

	err := svc.CreateLayout(ctx, layout)
	if err != nil {
		t.Fatalf("CreateLayout() error = %v", err)
	}

	// Get and verify
	got, err := svc.GetLayout(ctx, layout.ID)
	if err != nil {
		t.Errorf("GetLayout() error = %v", err)
	}
	if got.Code != layout.Code {
		t.Errorf("Code = %q, want %q", got.Code, layout.Code)
	}
	if got.HeaderImageID != image.ID {
		t.Errorf("HeaderImageID = %v, want %v", got.HeaderImageID, image.ID)
	}
}

func TestServiceSectionWithLayout(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Section Layout Site", "section-layout-site")

	// Create layout
	layout := NewLayout(site.ID, "Section Layout", "Layout for section")
	layout.CreatedBy = uuid.New()
	layout.UpdatedBy = layout.CreatedBy
	svc.CreateLayout(ctx, layout)

	// Create section with layout
	section := NewSection(site.ID, "Blog", "Blog section", "/blog")
	section.LayoutID = layout.ID
	section.LayoutName = layout.Name
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy

	err := svc.CreateSection(ctx, section)
	if err != nil {
		t.Fatalf("CreateSection() error = %v", err)
	}

	// Get and verify
	got, err := svc.GetSection(ctx, section.ID)
	if err != nil {
		t.Errorf("GetSection() error = %v", err)
	}
	if got.LayoutID != layout.ID {
		t.Errorf("LayoutID = %v, want %v", got.LayoutID, layout.ID)
	}
	if got.LayoutName != layout.Name {
		t.Errorf("LayoutName = %q, want %q", got.LayoutName, layout.Name)
	}
}

func TestServiceSettingWithAllFields(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Setting All Fields Site", "setting-all-fields-site")

	// Create setting with all fields
	setting := NewSetting(site.ID, "full_param", "param_value")
	setting.Description = "Full parameter description"
	setting.RefKey = "full_param_key"
	setting.Category = "general"
	setting.Position = 5
	setting.System = true
	setting.CreatedBy = uuid.New()
	setting.UpdatedBy = setting.CreatedBy

	err := svc.CreateSetting(ctx, setting)
	if err != nil {
		t.Fatalf("CreateSetting() error = %v", err)
	}

	// Get and verify
	got, err := svc.GetSetting(ctx, setting.ID)
	if err != nil {
		t.Errorf("GetSetting() error = %v", err)
	}
	if got.Description != "Full parameter description" {
		t.Errorf("Description = %q, want %q", got.Description, "Full parameter description")
	}
	if got.RefKey != "full_param_key" {
		t.Errorf("RefKey = %q, want %q", got.RefKey, "full_param_key")
	}
	if got.Category != "general" {
		t.Errorf("Category = %q, want %q", got.Category, "general")
	}
	if got.Position != 5 {
		t.Errorf("Position = %d, want %d", got.Position, 5)
	}
	if !got.System {
		t.Error("System should be true")
	}
}

func TestServiceImageWithAllFields(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Image All Fields Site", "image-all-fields-site")

	// Create image with all fields
	image := NewImage(site.ID, "full-image.jpg", "/images/full-image.jpg")
	image.AltText = "Full image alt text"
	image.Title = "Full image title"
	image.Width = 1920
	image.Height = 1080
	image.CreatedBy = uuid.New()
	image.UpdatedBy = image.CreatedBy

	err := svc.CreateImage(ctx, image)
	if err != nil {
		t.Fatalf("CreateImage() error = %v", err)
	}

	// Get and verify
	got, err := svc.GetImage(ctx, image.ID)
	if err != nil {
		t.Errorf("GetImage() error = %v", err)
	}
	if got.AltText != "Full image alt text" {
		t.Errorf("AltText = %q, want %q", got.AltText, "Full image alt text")
	}
	if got.Title != "Full image title" {
		t.Errorf("Title = %q, want %q", got.Title, "Full image title")
	}
	if got.Width != 1920 {
		t.Errorf("Width = %d, want %d", got.Width, 1920)
	}
	if got.Height != 1080 {
		t.Errorf("Height = %d, want %d", got.Height, 1080)
	}
}

func TestServiceDeleteContributorNonExistent(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Delete non-existent contributor - SQLite DELETE doesn't error
	err := svc.DeleteContributor(ctx, uuid.New())
	if err != nil {
		t.Logf("DeleteContributor() for non-existent returned: %v", err)
	}
}

func TestServiceDeleteSiteAndVerify(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Delete Me Site", "delete-me-site")

	// Verify exists
	_, err := svc.GetSite(ctx, site.ID)
	if err != nil {
		t.Fatalf("Site should exist: %v", err)
	}

	// Delete
	err = svc.DeleteSite(ctx, site.ID)
	if err != nil {
		t.Fatalf("DeleteSite() error = %v", err)
	}

	// Verify deleted
	_, err = svc.GetSite(ctx, site.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound after delete, got: %v", err)
	}
}

func TestServiceDeleteSectionAndVerify(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Del Section Site", "del-section-site")

	section := NewSection(site.ID, "Delete Me", "Delete me section", "/delete-me")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	// Verify exists
	_, err := svc.GetSection(ctx, section.ID)
	if err != nil {
		t.Fatalf("Section should exist: %v", err)
	}

	// Delete
	err = svc.DeleteSection(ctx, section.ID)
	if err != nil {
		t.Fatalf("DeleteSection() error = %v", err)
	}

	// Verify deleted
	_, err = svc.GetSection(ctx, section.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound after delete, got: %v", err)
	}
}

func TestServiceDeleteLayoutAndVerify(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Del Layout Site", "del-layout-site")

	layout := NewLayout(site.ID, "Delete Me", "Delete me layout")
	layout.CreatedBy = uuid.New()
	layout.UpdatedBy = layout.CreatedBy
	svc.CreateLayout(ctx, layout)

	// Verify exists
	_, err := svc.GetLayout(ctx, layout.ID)
	if err != nil {
		t.Fatalf("Layout should exist: %v", err)
	}

	// Delete
	err = svc.DeleteLayout(ctx, layout.ID)
	if err != nil {
		t.Fatalf("DeleteLayout() error = %v", err)
	}

	// Verify deleted
	_, err = svc.GetLayout(ctx, layout.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound after delete, got: %v", err)
	}
}

func TestServiceDeleteTagAndVerify(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Del Tag Site", "del-tag-site")

	tag := NewTag(site.ID, "delete-me-tag")
	tag.CreatedBy = uuid.New()
	tag.UpdatedBy = tag.CreatedBy
	svc.CreateTag(ctx, tag)

	// Verify exists
	_, err := svc.GetTag(ctx, tag.ID)
	if err != nil {
		t.Fatalf("Tag should exist: %v", err)
	}

	// Delete
	err = svc.DeleteTag(ctx, tag.ID)
	if err != nil {
		t.Fatalf("DeleteTag() error = %v", err)
	}

	// Verify deleted
	_, err = svc.GetTag(ctx, tag.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound after delete, got: %v", err)
	}
}

func TestServiceDeleteSettingAndVerify(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Del Setting Site", "del-setting-site")

	setting := NewSetting(site.ID, "delete_me", "value")
	setting.CreatedBy = uuid.New()
	setting.UpdatedBy = setting.CreatedBy
	svc.CreateSetting(ctx, setting)

	// Verify exists
	_, err := svc.GetSetting(ctx, setting.ID)
	if err != nil {
		t.Fatalf("Setting should exist: %v", err)
	}

	// Delete
	err = svc.DeleteSetting(ctx, setting.ID)
	if err != nil {
		t.Fatalf("DeleteSetting() error = %v", err)
	}

	// Verify deleted
	_, err = svc.GetSetting(ctx, setting.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound after delete, got: %v", err)
	}
}

func TestServiceDeleteImageAndVerify(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Del Image Site", "del-image-site")

	image := NewImage(site.ID, "delete.jpg", "/images/delete.jpg")
	image.CreatedBy = uuid.New()
	image.UpdatedBy = image.CreatedBy
	svc.CreateImage(ctx, image)

	// Verify exists
	_, err := svc.GetImage(ctx, image.ID)
	if err != nil {
		t.Fatalf("Image should exist: %v", err)
	}

	// Delete
	err = svc.DeleteImage(ctx, image.ID)
	if err != nil {
		t.Fatalf("DeleteImage() error = %v", err)
	}

	// Verify deleted
	_, err = svc.GetImage(ctx, image.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound after delete, got: %v", err)
	}
}

func TestServiceDeleteContentAndVerify(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Del Content Site", "del-content-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	content := NewContent(site.ID, section.ID, "Delete Me", "Body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	// Verify exists
	_, err := svc.GetContent(ctx, content.ID)
	if err != nil {
		t.Fatalf("Content should exist: %v", err)
	}

	// Delete
	err = svc.DeleteContent(ctx, content.ID)
	if err != nil {
		t.Fatalf("DeleteContent() error = %v", err)
	}

	// Verify deleted
	_, err = svc.GetContent(ctx, content.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound after delete, got: %v", err)
	}
}

func TestServiceLinkUnlinkImageToSectionFull(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Section Img Site 2", "section-img-site-2")

	section := NewSection(site.ID, "Images", "", "/images")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	image := NewImage(site.ID, "section.jpg", "/images/section.jpg")
	image.CreatedBy = uuid.New()
	image.UpdatedBy = image.CreatedBy
	svc.CreateImage(ctx, image)

	// Link image (with isHeader = false)
	err := svc.LinkImageToSection(ctx, section.ID, image.ID, false)
	if err != nil {
		t.Fatalf("LinkImageToSection() error = %v", err)
	}

	// Verify linked
	images, err := svc.GetSectionImagesWithDetails(ctx, section.ID)
	if err != nil {
		t.Fatalf("GetSectionImagesWithDetails() error = %v", err)
	}
	if len(images) != 1 {
		t.Fatalf("Expected 1 image, got %d", len(images))
	}

	// Get section image details
	details, err := svc.GetSectionImageDetails(ctx, images[0].SectionImageID)
	if err != nil {
		t.Fatalf("GetSectionImageDetails() error = %v", err)
	}
	if details.ImageID != image.ID {
		t.Errorf("ImageID = %v, want %v", details.ImageID, image.ID)
	}

	// Unlink image
	err = svc.UnlinkImageFromSection(ctx, images[0].SectionImageID)
	if err != nil {
		t.Fatalf("UnlinkImageFromSection() error = %v", err)
	}

	// Verify unlinked
	images, _ = svc.GetSectionImagesWithDetails(ctx, section.ID)
	if len(images) != 0 {
		t.Errorf("Expected 0 images after unlink, got %d", len(images))
	}
}

func TestServiceContentImageDetailsFull(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Content Img Details Site 2", "content-img-details-site-2")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	content := NewContent(site.ID, section.ID, "Image Post", "Body with images")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	image := NewImage(site.ID, "content.jpg", "/images/content.jpg")
	image.CreatedBy = uuid.New()
	image.UpdatedBy = image.CreatedBy
	svc.CreateImage(ctx, image)

	// Link image to content (not as header)
	err := svc.LinkImageToContent(ctx, content.ID, image.ID, false)
	if err != nil {
		t.Fatalf("LinkImageToContent() error = %v", err)
	}

	// Get content images
	images, err := svc.GetContentImagesWithDetails(ctx, content.ID)
	if err != nil {
		t.Fatalf("GetContentImagesWithDetails() error = %v", err)
	}
	if len(images) != 1 {
		t.Fatalf("Expected 1 image, got %d", len(images))
	}

	// Get content image details
	details, err := svc.GetContentImageDetails(ctx, images[0].ContentImageID)
	if err != nil {
		t.Fatalf("GetContentImageDetails() error = %v", err)
	}
	if details.ImageID != image.ID {
		t.Errorf("ImageID = %v, want %v", details.ImageID, image.ID)
	}

	// Unlink
	err = svc.UnlinkImageFromContent(ctx, images[0].ContentImageID)
	if err != nil {
		t.Fatalf("UnlinkImageFromContent() error = %v", err)
	}

	// Verify unlinked
	images, _ = svc.GetContentImagesWithDetails(ctx, content.ID)
	if len(images) != 0 {
		t.Errorf("Expected 0 images after unlink, got %d", len(images))
	}
}

// Tests to improve coverage on error paths

func TestServiceAddTagToContentDuplicate(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Dup Tag Site", "dup-tag-site")

	section := NewSection(site.ID, "Blog", "", "/blog")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy
	svc.CreateSection(ctx, section)

	content := NewContent(site.ID, section.ID, "Tagged Post", "Body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy
	svc.CreateContent(ctx, content)

	// Add tag first time - should succeed
	err := svc.AddTagToContent(ctx, content.ID, "test-tag", site.ID)
	if err != nil {
		t.Fatalf("First AddTagToContent() error = %v", err)
	}

	// Add same tag again - might fail with constraint or succeed
	err = svc.AddTagToContent(ctx, content.ID, "test-tag", site.ID)
	// Just log the result, we're exercising the code path
	if err != nil {
		t.Logf("Second AddTagToContent() returned: %v", err)
	}
}

func TestServiceCreateSiteDuplicateSlugError(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Create first site
	site1 := NewSite("Site One", "same-slug-err", "blog")
	err := svc.CreateSite(ctx, site1)
	if err != nil {
		t.Fatalf("First CreateSite() error = %v", err)
	}

	// Try to create second site with same slug
	site2 := NewSite("Site Two", "same-slug-err", "blog")
	err = svc.CreateSite(ctx, site2)
	if err == nil {
		t.Error("Expected error for duplicate slug")
	}
}

func TestServiceCreateContentInvalidSection(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Invalid Section Site", "invalid-section-site")

	// Try to create content with non-existent section ID
	content := NewContent(site.ID, uuid.New(), "Invalid Post", "Body")
	content.CreatedBy = uuid.New()
	content.UpdatedBy = content.CreatedBy

	err := svc.CreateContent(ctx, content)
	if err == nil {
		t.Error("Expected error for invalid section ID")
	}
}

func TestServiceCreateContributorInvalidSite(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Try to create contributor with non-existent site ID
	contributor := NewContributor(uuid.New(), "invalid", "Invalid", "User")
	contributor.CreatedBy = uuid.New()
	contributor.UpdatedBy = contributor.CreatedBy

	err := svc.CreateContributor(ctx, contributor)
	if err == nil {
		t.Error("Expected error for invalid site ID")
	}
}

func TestServiceCreateSectionInvalidSite(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Try to create section with non-existent site ID
	section := NewSection(uuid.New(), "Invalid Section", "Desc", "/invalid")
	section.CreatedBy = uuid.New()
	section.UpdatedBy = section.CreatedBy

	err := svc.CreateSection(ctx, section)
	if err == nil {
		t.Error("Expected error for invalid site ID")
	}
}

func TestServiceCreateLayoutInvalidSite(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Try to create layout with non-existent site ID
	layout := NewLayout(uuid.New(), "Invalid Layout", "Desc")
	layout.CreatedBy = uuid.New()
	layout.UpdatedBy = layout.CreatedBy

	err := svc.CreateLayout(ctx, layout)
	if err == nil {
		t.Error("Expected error for invalid site ID")
	}
}

func TestServiceCreateTagInvalidSite(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Try to create tag with non-existent site ID
	tag := NewTag(uuid.New(), "invalid-tag")
	tag.CreatedBy = uuid.New()
	tag.UpdatedBy = tag.CreatedBy

	err := svc.CreateTag(ctx, tag)
	if err == nil {
		t.Error("Expected error for invalid site ID")
	}
}

func TestServiceCreateSettingInvalidSite(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Try to create setting with non-existent site ID
	setting := NewSetting(uuid.New(), "invalid_param", "value")
	setting.CreatedBy = uuid.New()
	setting.UpdatedBy = setting.CreatedBy

	err := svc.CreateSetting(ctx, setting)
	if err == nil {
		t.Error("Expected error for invalid site ID")
	}
}

func TestServiceCreateImageInvalidSite(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Try to create image with non-existent site ID
	image := NewImage(uuid.New(), "invalid.jpg", "/images/invalid.jpg")
	image.CreatedBy = uuid.New()
	image.UpdatedBy = image.CreatedBy

	err := svc.CreateImage(ctx, image)
	if err == nil {
		t.Error("Expected error for invalid site ID")
	}
}

func TestServiceCreateMetaInvalidContent(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Invalid Meta Site", "invalid-meta-site")

	// Try to create meta with non-existent content ID
	meta := NewMeta(site.ID, uuid.New())
	meta.CreatedBy = uuid.New()
	meta.UpdatedBy = meta.CreatedBy

	err := svc.CreateMeta(ctx, meta)
	if err == nil {
		t.Error("Expected error for invalid content ID")
	}
}

func TestServiceLinkImageToContentInvalidIDs(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Try to link with non-existent IDs
	err := svc.LinkImageToContent(ctx, uuid.New(), uuid.New(), false)
	if err == nil {
		t.Error("Expected error for invalid IDs")
	}
}

func TestServiceLinkImageToSectionInvalidIDs(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Try to link with non-existent IDs
	err := svc.LinkImageToSection(ctx, uuid.New(), uuid.New(), false)
	if err == nil {
		t.Error("Expected error for invalid IDs")
	}
}

func TestServiceAddTagToContentByIDInvalidIDs(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Try to add tag with non-existent IDs
	err := svc.AddTagToContentByID(ctx, uuid.New(), uuid.New())
	if err == nil {
		t.Error("Expected error for invalid IDs")
	}
}

func TestServiceSetContributorProfileInvalidIDs(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Try to set profile with non-existent contributor ID
	// SQLite UPDATE doesn't error on non-existent rows
	err := svc.SetContributorProfile(ctx, uuid.New(), uuid.New(), uuid.New().String())
	// Just exercise the code path
	if err != nil {
		t.Logf("SetContributorProfile for non-existent returned: %v", err)
	}
}

func TestServiceGetContentImageDetailsNotFound(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Try to get details for non-existent content image
	_, err := svc.GetContentImageDetails(ctx, uuid.New())
	if err == nil {
		t.Error("Expected error for non-existent content image")
	}
}

func TestServiceGetSectionImageDetailsNotFound(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Try to get details for non-existent section image
	_, err := svc.GetSectionImageDetails(ctx, uuid.New())
	if err == nil {
		t.Error("Expected error for non-existent section image")
	}
}

func TestServiceUpdateContributorNonExistent(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Update Nonexistent Site", "update-nonexistent-site")

	// Try to update non-existent contributor
	contributor := NewContributor(site.ID, "nonexistent", "Non", "Existent")
	contributor.UpdatedAt = time.Now()

	err := svc.UpdateContributor(ctx, contributor)
	// SQLite UPDATE doesn't error on non-existent, just log
	if err != nil {
		t.Logf("UpdateContributor for non-existent returned: %v", err)
	}
}

func TestServiceUpdateSiteCancelledContext(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	site := createTestSite(t, svc, "Context Cancel Test", "context-cancel-test")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	site.Name = "Updated"
	err := svc.UpdateSite(ctx, site)
	if err == nil {
		t.Error("UpdateSite should fail with cancelled context")
	}
}

func TestServiceDeleteSiteCancelledContext(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	site := createTestSite(t, svc, "Delete Context Cancel", "delete-context-cancel")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := svc.DeleteSite(ctx, site.ID)
	if err == nil {
		t.Error("DeleteSite should fail with cancelled context")
	}
}

func TestServiceUpdateTagCancelledContext(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Tag Context Test", "tag-context-test")

	tag := NewTag(site.ID, "test-tag")
	if err := svc.CreateTag(ctx, tag); err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	ctx2, cancel := context.WithCancel(context.Background())
	cancel()

	tag.Name = "updated-tag"
	err := svc.UpdateTag(ctx2, tag)
	if err == nil {
		t.Error("UpdateTag should fail with cancelled context")
	}
}

func TestServiceDeleteTagCancelledContext(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Delete Tag Context", "delete-tag-context")

	tag := NewTag(site.ID, "tag-to-delete")
	if err := svc.CreateTag(ctx, tag); err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	ctx2, cancel := context.WithCancel(context.Background())
	cancel()

	err := svc.DeleteTag(ctx2, tag.ID)
	if err == nil {
		t.Error("DeleteTag should fail with cancelled context")
	}
}

func TestServiceUpdateLayoutCancelledContext(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Layout Context Test", "layout-context-test")

	layout := NewLayout(site.ID, "test-layout", "blog")
	if err := svc.CreateLayout(ctx, layout); err != nil {
		t.Fatalf("CreateLayout failed: %v", err)
	}

	ctx2, cancel := context.WithCancel(context.Background())
	cancel()

	layout.Name = "updated-layout"
	err := svc.UpdateLayout(ctx2, layout)
	if err == nil {
		t.Error("UpdateLayout should fail with cancelled context")
	}
}

func TestServiceUpdateSettingCancelledContext(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Setting Context Test", "setting-context-test")

	setting := NewSetting(site.ID, "test-setting", "value")
	if err := svc.CreateSetting(ctx, setting); err != nil {
		t.Fatalf("CreateSetting failed: %v", err)
	}

	ctx2, cancel := context.WithCancel(context.Background())
	cancel()

	setting.Value = "updated"
	err := svc.UpdateSetting(ctx2, setting)
	if err == nil {
		t.Error("UpdateSetting should fail with cancelled context")
	}
}

func TestServiceUpdateImageCancelledContext(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Image Context Test", "image-context-test")

	image := NewImage(site.ID, "test.jpg", "/images/test.jpg")
	if err := svc.CreateImage(ctx, image); err != nil {
		t.Fatalf("CreateImage failed: %v", err)
	}

	ctx2, cancel := context.WithCancel(context.Background())
	cancel()

	image.FileName = "updated.jpg"
	err := svc.UpdateImage(ctx2, image)
	if err == nil {
		t.Error("UpdateImage should fail with cancelled context")
	}
}

func TestServiceUpdateSectionCancelledContext(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Section Context Test", "section-context-test")

	section := NewSection(site.ID, "test-section", "Test description", "test-path")
	if err := svc.CreateSection(ctx, section); err != nil {
		t.Fatalf("CreateSection failed: %v", err)
	}

	ctx2, cancel := context.WithCancel(context.Background())
	cancel()

	section.Name = "updated-section"
	err := svc.UpdateSection(ctx2, section)
	if err == nil {
		t.Error("UpdateSection should fail with cancelled context")
	}
}

func TestServiceUpdateMetaCancelledContext(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Meta Context Test", "meta-context-test")

	section := NewSection(site.ID, "section", "Description", "path")
	if err := svc.CreateSection(ctx, section); err != nil {
		t.Fatalf("CreateSection failed: %v", err)
	}

	content := NewContent(site.ID, section.ID, "Test Content", "Body")
	if err := svc.CreateContent(ctx, content); err != nil {
		t.Fatalf("CreateContent failed: %v", err)
	}

	meta := NewMeta(site.ID, content.ID)
	if err := svc.CreateMeta(ctx, meta); err != nil {
		t.Fatalf("CreateMeta failed: %v", err)
	}

	ctx2, cancel := context.WithCancel(context.Background())
	cancel()

	meta.Description = "updated"
	err := svc.UpdateMeta(ctx2, meta)
	if err == nil {
		t.Error("UpdateMeta should fail with cancelled context")
	}
}

func TestServiceDeleteContributorCancelledContext(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Delete Contributor Context", "delete-contrib-context")

	contributor := NewContributor(site.ID, "test-contrib", "Test", "Contrib")
	contributor.CreatedBy = uuid.New()
	contributor.UpdatedBy = contributor.CreatedBy
	if err := svc.CreateContributor(ctx, contributor); err != nil {
		t.Fatalf("CreateContributor failed: %v", err)
	}

	ctx2, cancel := context.WithCancel(context.Background())
	cancel()

	err := svc.DeleteContributor(ctx2, contributor.ID)
	if err == nil {
		t.Error("DeleteContributor should fail with cancelled context")
	}
}

func TestServiceUnlinkImageFromContentCancelledContext(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Unlink Image Context", "unlink-image-context")

	section := NewSection(site.ID, "section", "Description", "path")
	if err := svc.CreateSection(ctx, section); err != nil {
		t.Fatalf("CreateSection failed: %v", err)
	}

	content := NewContent(site.ID, section.ID, "Test Content", "Body")
	if err := svc.CreateContent(ctx, content); err != nil {
		t.Fatalf("CreateContent failed: %v", err)
	}

	image := NewImage(site.ID, "test.jpg", "/images/test.jpg")
	if err := svc.CreateImage(ctx, image); err != nil {
		t.Fatalf("CreateImage failed: %v", err)
	}

	if err := svc.LinkImageToContent(ctx, content.ID, image.ID, false); err != nil {
		t.Fatalf("LinkImageToContent failed: %v", err)
	}

	// Get the content image ID
	images, err := svc.GetContentImagesWithDetails(ctx, content.ID)
	if err != nil || len(images) == 0 {
		t.Fatalf("GetContentImagesWithDetails failed: %v", err)
	}
	contentImageID := images[0].ContentImageID

	ctx2, cancel := context.WithCancel(context.Background())
	cancel()

	err = svc.UnlinkImageFromContent(ctx2, contentImageID)
	if err == nil {
		t.Error("UnlinkImageFromContent should fail with cancelled context")
	}
}

func TestServiceUnlinkImageFromSectionCancelledContext(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Unlink Section Image Context", "unlink-section-image")

	section := NewSection(site.ID, "section", "Description", "path")
	if err := svc.CreateSection(ctx, section); err != nil {
		t.Fatalf("CreateSection failed: %v", err)
	}

	image := NewImage(site.ID, "test.jpg", "/images/test.jpg")
	if err := svc.CreateImage(ctx, image); err != nil {
		t.Fatalf("CreateImage failed: %v", err)
	}

	if err := svc.LinkImageToSection(ctx, section.ID, image.ID, false); err != nil {
		t.Fatalf("LinkImageToSection failed: %v", err)
	}

	// Get the section image ID
	images, err := svc.GetSectionImagesWithDetails(ctx, section.ID)
	if err != nil || len(images) == 0 {
		t.Fatalf("GetSectionImagesWithDetails failed: %v", err)
	}
	sectionImageID := images[0].SectionImageID

	ctx2, cancel := context.WithCancel(context.Background())
	cancel()

	err = svc.UnlinkImageFromSection(ctx2, sectionImageID)
	if err == nil {
		t.Error("UnlinkImageFromSection should fail with cancelled context")
	}
}

func TestServiceAddTagToContentCancelledContext(t *testing.T) {
	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	site := createTestSite(t, svc, "Add Tag Context Test", "add-tag-context")

	section := NewSection(site.ID, "section", "Description", "path")
	if err := svc.CreateSection(ctx, section); err != nil {
		t.Fatalf("CreateSection failed: %v", err)
	}

	content := NewContent(site.ID, section.ID, "Test Content", "Body")
	if err := svc.CreateContent(ctx, content); err != nil {
		t.Fatalf("CreateContent failed: %v", err)
	}

	ctx2, cancel := context.WithCancel(context.Background())
	cancel()

	err := svc.AddTagToContent(ctx2, content.ID, "new-tag", site.ID)
	if err == nil {
		t.Error("AddTagToContent should fail with cancelled context")
	}
}
