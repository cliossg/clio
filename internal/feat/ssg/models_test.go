package ssg

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewSite(t *testing.T) {
	tests := []struct {
		name     string
		siteName string
		slug     string
		mode     string
	}{
		{
			name:     "blog mode site",
			siteName: "My Blog",
			slug:     "my-blog",
			mode:     "blog",
		},
		{
			name:     "structured mode site",
			siteName: "Documentation",
			slug:     "docs",
			mode:     "structured",
		},
		{
			name:     "empty values",
			siteName: "",
			slug:     "",
			mode:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now()
			site := NewSite(tt.siteName, tt.slug, tt.mode)
			after := time.Now()

			if site.Name != tt.siteName {
				t.Errorf("Name = %q, want %q", site.Name, tt.siteName)
			}
			if site.Slug != tt.slug {
				t.Errorf("Slug = %q, want %q", site.Slug, tt.slug)
			}
			if site.Mode != tt.mode {
				t.Errorf("Mode = %q, want %q", site.Mode, tt.mode)
			}
			if !site.Active {
				t.Error("Active should be true by default")
			}
			if site.ID == uuid.Nil {
				t.Error("ID should not be nil")
			}
			if site.ShortID == "" {
				t.Error("ShortID should not be empty")
			}
			if len(site.ShortID) != 8 {
				t.Errorf("ShortID length = %d, want 8", len(site.ShortID))
			}
			if site.CreatedAt.Before(before) || site.CreatedAt.After(after) {
				t.Error("CreatedAt should be approximately now")
			}
			if site.UpdatedAt.Before(before) || site.UpdatedAt.After(after) {
				t.Error("UpdatedAt should be approximately now")
			}
		})
	}
}

func TestNewSection(t *testing.T) {
	siteID := uuid.New()

	tests := []struct {
		name        string
		siteID      uuid.UUID
		sectionName string
		description string
		path        string
		wantPath    string
	}{
		{
			name:        "blog section",
			siteID:      siteID,
			sectionName: "Blog",
			description: "Blog posts",
			path:        "/blog",
			wantPath:    "blog", // leading slash removed
		},
		{
			name:        "path without leading slash",
			siteID:      siteID,
			sectionName: "Docs",
			description: "Documentation",
			path:        "docs",
			wantPath:    "docs",
		},
		{
			name:        "multiple leading slashes",
			siteID:      siteID,
			sectionName: "Deep",
			description: "Deep section",
			path:        "///deep",
			wantPath:    "deep",
		},
		{
			name:        "empty path",
			siteID:      siteID,
			sectionName: "Root",
			description: "Root section",
			path:        "",
			wantPath:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			section := NewSection(tt.siteID, tt.sectionName, tt.description, tt.path)

			if section.SiteID != tt.siteID {
				t.Errorf("SiteID = %v, want %v", section.SiteID, tt.siteID)
			}
			if section.Name != tt.sectionName {
				t.Errorf("Name = %q, want %q", section.Name, tt.sectionName)
			}
			if section.Description != tt.description {
				t.Errorf("Description = %q, want %q", section.Description, tt.description)
			}
			if section.Path != tt.wantPath {
				t.Errorf("Path = %q, want %q", section.Path, tt.wantPath)
			}
			if section.ID == uuid.Nil {
				t.Error("ID should not be nil")
			}
		})
	}
}

func TestNewContent(t *testing.T) {
	siteID := uuid.New()
	sectionID := uuid.New()

	tests := []struct {
		name      string
		siteID    uuid.UUID
		sectionID uuid.UUID
		heading   string
		body      string
	}{
		{
			name:      "standard content",
			siteID:    siteID,
			sectionID: sectionID,
			heading:   "My Post",
			body:      "Post content here",
		},
		{
			name:      "empty body",
			siteID:    siteID,
			sectionID: sectionID,
			heading:   "Draft Post",
			body:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := NewContent(tt.siteID, tt.sectionID, tt.heading, tt.body)

			if content.SiteID != tt.siteID {
				t.Errorf("SiteID = %v, want %v", content.SiteID, tt.siteID)
			}
			if content.SectionID != tt.sectionID {
				t.Errorf("SectionID = %v, want %v", content.SectionID, tt.sectionID)
			}
			if content.Heading != tt.heading {
				t.Errorf("Heading = %q, want %q", content.Heading, tt.heading)
			}
			if content.Body != tt.body {
				t.Errorf("Body = %q, want %q", content.Body, tt.body)
			}
			if !content.Draft {
				t.Error("Draft should be true by default")
			}
			if content.Kind != "post" {
				t.Errorf("Kind = %q, want %q", content.Kind, "post")
			}
		})
	}
}

func TestContentSlug(t *testing.T) {
	tests := []struct {
		name     string
		heading  string
		shortID  string
		wantSlug string
	}{
		{
			name:     "simple heading",
			heading:  "Hello World",
			shortID:  "abc12345",
			wantSlug: "hello-world-abc12345",
		},
		{
			name:     "heading with special chars",
			heading:  "What's New in 2024?",
			shortID:  "xyz99999",
			wantSlug: "what-s-new-in-2024-xyz99999",
		},
		{
			name:     "empty heading",
			heading:  "",
			shortID:  "empty123",
			wantSlug: "-empty123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := &Content{
				Heading: tt.heading,
				ShortID: tt.shortID,
			}
			if got := content.Slug(); got != tt.wantSlug {
				t.Errorf("Content.Slug() = %q, want %q", got, tt.wantSlug)
			}
		})
	}
}

func TestContentDisplayHandle(t *testing.T) {
	tests := []struct {
		name              string
		contributorHandle string
		authorUsername    string
		want              string
	}{
		{
			name:              "contributor handle takes precedence",
			contributorHandle: "contributor1",
			authorUsername:    "author1",
			want:              "contributor1",
		},
		{
			name:              "falls back to author username",
			contributorHandle: "",
			authorUsername:    "author1",
			want:              "author1",
		},
		{
			name:              "both empty",
			contributorHandle: "",
			authorUsername:    "",
			want:              "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := &Content{
				ContributorHandle: tt.contributorHandle,
				AuthorUsername:    tt.authorUsername,
			}
			if got := content.DisplayHandle(); got != tt.want {
				t.Errorf("Content.DisplayHandle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewLayout(t *testing.T) {
	siteID := uuid.New()

	layout := NewLayout(siteID, "Default", "Default layout template")

	if layout.SiteID != siteID {
		t.Errorf("SiteID = %v, want %v", layout.SiteID, siteID)
	}
	if layout.Name != "Default" {
		t.Errorf("Name = %q, want %q", layout.Name, "Default")
	}
	if layout.Description != "Default layout template" {
		t.Errorf("Description = %q, want %q", layout.Description, "Default layout template")
	}
	if layout.ID == uuid.Nil {
		t.Error("ID should not be nil")
	}
}

func TestNewTag(t *testing.T) {
	siteID := uuid.New()

	tests := []struct {
		name     string
		tagName  string
		wantSlug string
	}{
		{
			name:     "simple tag",
			tagName:  "Technology",
			wantSlug: "technology",
		},
		{
			name:     "tag with spaces",
			tagName:  "Web Development",
			wantSlug: "web-development",
		},
		{
			name:     "tag with special chars",
			tagName:  "C++ Programming",
			wantSlug: "c-programming",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag := NewTag(siteID, tt.tagName)

			if tag.SiteID != siteID {
				t.Errorf("SiteID = %v, want %v", tag.SiteID, siteID)
			}
			if tag.Name != tt.tagName {
				t.Errorf("Name = %q, want %q", tag.Name, tt.tagName)
			}
			if tag.Slug != tt.wantSlug {
				t.Errorf("Slug = %q, want %q", tag.Slug, tt.wantSlug)
			}
		})
	}
}

func TestNewMeta(t *testing.T) {
	siteID := uuid.New()
	contentID := uuid.New()

	meta := NewMeta(siteID, contentID)

	if meta.SiteID != siteID {
		t.Errorf("SiteID = %v, want %v", meta.SiteID, siteID)
	}
	if meta.ContentID != contentID {
		t.Errorf("ContentID = %v, want %v", meta.ContentID, contentID)
	}
	if meta.ID == uuid.Nil {
		t.Error("ID should not be nil")
	}
}

func TestNewSetting(t *testing.T) {
	siteID := uuid.New()

	setting := NewSetting(siteID, "site_title", "My Site")

	if setting.SiteID != siteID {
		t.Errorf("SiteID = %v, want %v", setting.SiteID, siteID)
	}
	if setting.Name != "site_title" {
		t.Errorf("Name = %q, want %q", setting.Name, "site_title")
	}
	if setting.Value != "My Site" {
		t.Errorf("Value = %q, want %q", setting.Value, "My Site")
	}
}

func TestSettingMaskedValue(t *testing.T) {
	tests := []struct {
		name   string
		param  Setting
		want   string
	}{
		{
			name:   "non-sensitive short value",
			param:  Setting{Name: "site_title", Value: "My Site"},
			want:   "My Site",
		},
		{
			name:   "non-sensitive long value",
			param:  Setting{Name: "description", Value: string(make([]byte, 60))},
			want:   string(make([]byte, 50)) + "...",
		},
		{
			name:   "sensitive with token in name",
			param:  Setting{Name: "api_token", Value: "sk_test_1234567890"},
			want:   "sk_t***...7890",
		},
		{
			name:   "sensitive with pass in name",
			param:  Setting{Name: "password", Value: "mysecretpassword"},
			want:   "myse***...word",
		},
		{
			name:   "sensitive with secret in name",
			param:  Setting{Name: "secret_key", Value: "secretvalue123"},
			want:   "secr***...e123",
		},
		{
			name:   "sensitive with key in name",
			param:  Setting{Name: "encryption_key", Value: "encryptionkey!"},
			want:   "encr***...key!",
		},
		{
			name:   "sensitive with credential in name",
			param:  Setting{Name: "credential", Value: "cred12345678"},
			want:   "cred***...5678",
		},
		{
			name:   "sensitive in refKey",
			param:  Setting{Name: "setting1", RefKey: "api_token", Value: "tokenvalue12"},
			want:   "toke***...ue12",
		},
		{
			name:   "sensitive short value",
			param:  Setting{Name: "token", Value: "short"},
			want:   "***",
		},
		{
			name:   "empty value",
			param:  Setting{Name: "token", Value: ""},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.param.MaskedValue(); got != tt.want {
				t.Errorf("Setting.MaskedValue() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewImage(t *testing.T) {
	siteID := uuid.New()

	image := NewImage(siteID, "photo.jpg", "/images/photo.jpg")

	if image.SiteID != siteID {
		t.Errorf("SiteID = %v, want %v", image.SiteID, siteID)
	}
	if image.FileName != "photo.jpg" {
		t.Errorf("FileName = %q, want %q", image.FileName, "photo.jpg")
	}
	if image.FilePath != "/images/photo.jpg" {
		t.Errorf("FilePath = %q, want %q", image.FilePath, "/images/photo.jpg")
	}
}

func TestNewContributor(t *testing.T) {
	siteID := uuid.New()

	contributor := NewContributor(siteID, "johndoe", "John", "Doe")

	if contributor.SiteID != siteID {
		t.Errorf("SiteID = %v, want %v", contributor.SiteID, siteID)
	}
	if contributor.Handle != "johndoe" {
		t.Errorf("Handle = %q, want %q", contributor.Handle, "johndoe")
	}
	if contributor.Name != "John" {
		t.Errorf("Name = %q, want %q", contributor.Name, "John")
	}
	if contributor.Surname != "Doe" {
		t.Errorf("Surname = %q, want %q", contributor.Surname, "Doe")
	}
	if contributor.Role != ContributorRoleEditor {
		t.Errorf("Role = %q, want %q", contributor.Role, ContributorRoleEditor)
	}
	if len(contributor.SocialLinks) != 0 {
		t.Error("SocialLinks should be empty by default")
	}
}

func TestContributorFullName(t *testing.T) {
	tests := []struct {
		name    string
		contrib Contributor
		want    string
	}{
		{
			name:    "full name with surname",
			contrib: Contributor{Name: "John", Surname: "Doe"},
			want:    "John Doe",
		},
		{
			name:    "name only",
			contrib: Contributor{Name: "John", Surname: ""},
			want:    "John",
		},
		{
			name:    "empty name",
			contrib: Contributor{Name: "", Surname: "Doe"},
			want:    " Doe",
		},
		{
			name:    "both empty",
			contrib: Contributor{Name: "", Surname: ""},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.contrib.FullName(); got != tt.want {
				t.Errorf("Contributor.FullName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple text",
			input: "Hello World",
			want:  "hello-world",
		},
		{
			name:  "already lowercase",
			input: "hello world",
			want:  "hello-world",
		},
		{
			name:  "special characters",
			input: "Hello! World? 2024",
			want:  "hello-world-2024",
		},
		{
			name:  "multiple spaces",
			input: "Hello   World",
			want:  "hello-world",
		},
		{
			name:  "leading and trailing spaces",
			input: "  Hello World  ",
			want:  "hello-world",
		},
		{
			name:  "unicode characters",
			input: "Héllo Wörld",
			want:  "h-llo-w-rld",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only special characters",
			input: "!@#$%",
			want:  "",
		},
		{
			name:  "numbers",
			input: "Article 123",
			want:  "article-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Slugify(tt.input); got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple text",
			input: "Hello World",
			want:  "hello-world",
		},
		{
			name:  "with numbers",
			input: "Test 123",
			want:  "test-123",
		},
		{
			name:  "unicode filtered",
			input: "Héllo",
			want:  "hllo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Normalize(tt.input); got != tt.want {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
