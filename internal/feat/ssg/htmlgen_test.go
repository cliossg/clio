package ssg

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGenerateSitemap(t *testing.T) {
	tmpDir := t.TempDir()
	g := &HTMLGenerator{}

	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	siteID := uuid.New()
	sectionCoding := &Section{ID: uuid.New(), SiteID: siteID, Name: "Coding", Path: "coding"}
	sectionMain := &Section{ID: uuid.New(), SiteID: siteID, Name: "main", Path: ""}

	sections := []*Section{sectionMain, sectionCoding}

	publishedAt := past
	contents := []*Content{
		{
			ID:          uuid.New(),
			SiteID:      siteID,
			SectionID:   sectionCoding.ID,
			ShortID:     "abc12345",
			Heading:     "My Post",
			SectionPath: "coding",
			Draft:       false,
			PublishedAt: &publishedAt,
			UpdatedAt:   past,
		},
		{
			ID:          uuid.New(),
			SiteID:      siteID,
			SectionID:   sectionCoding.ID,
			ShortID:     "def67890",
			Heading:     "Draft Post",
			SectionPath: "coding",
			Draft:       true,
			UpdatedAt:   now,
		},
		{
			ID:          uuid.New(),
			SiteID:      siteID,
			SectionID:   sectionCoding.ID,
			ShortID:     "exc12345",
			Heading:     "Excluded Post",
			SectionPath: "coding",
			Draft:       false,
			PublishedAt: &publishedAt,
			UpdatedAt:   past,
			Meta:        &Meta{Sitemap: "exclude"},
		},
		{
			ID:          uuid.New(),
			SiteID:      siteID,
			SectionID:   sectionCoding.ID,
			ShortID:     "noi12345",
			Heading:     "Noindex Post",
			SectionPath: "coding",
			Draft:       false,
			PublishedAt: &publishedAt,
			UpdatedAt:   past,
			Meta:        &Meta{Sitemap: "noindex"},
		},
		{
			ID:          uuid.New(),
			SiteID:      siteID,
			SectionID:   sectionCoding.ID,
			ShortID:     "fut12345",
			Heading:     "Future Post",
			SectionPath: "coding",
			Draft:       false,
			PublishedAt: &future,
			UpdatedAt:   now,
		},
	}

	site := &Site{ID: siteID, Name: "Test", Slug: "test"}

	err := g.generateSitemap(tmpDir, "https://example.com", "/", site, contents, sections)
	if err != nil {
		t.Fatalf("generateSitemap failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "sitemap.xml"))
	if err != nil {
		t.Fatalf("failed to read sitemap.xml: %v", err)
	}

	var urlSet sitemapURLSet
	if err := xml.Unmarshal(data, &urlSet); err != nil {
		t.Fatalf("invalid XML: %v", err)
	}

	if urlSet.XMLNS != "http://www.sitemaps.org/schemas/sitemap/0.9" {
		t.Errorf("wrong xmlns: %s", urlSet.XMLNS)
	}

	// Expected URLs: homepage + coding section + 1 published post (draft, excluded, noindex, future excluded)
	expectedCount := 3
	if len(urlSet.URLs) != expectedCount {
		t.Errorf("expected %d URLs, got %d", expectedCount, len(urlSet.URLs))
		for i, u := range urlSet.URLs {
			t.Logf("  URL[%d]: %s", i, u.Loc)
		}
	}

	// Check homepage URL
	if urlSet.URLs[0].Loc != "https://example.com/" {
		t.Errorf("homepage URL = %s, want https://example.com/", urlSet.URLs[0].Loc)
	}

	// Check section URL
	if urlSet.URLs[1].Loc != "https://example.com/coding/" {
		t.Errorf("section URL = %s, want https://example.com/coding/", urlSet.URLs[1].Loc)
	}

	// Check content URL contains the expected slug
	contentURL := urlSet.URLs[2].Loc
	expectedSlug := Slugify("My Post") + "-abc12345"
	if contentURL != "https://example.com/coding/"+expectedSlug+"/" {
		t.Errorf("content URL = %s, want https://example.com/coding/%s/", contentURL, expectedSlug)
	}

	// Check lastmod is a valid date
	for _, u := range urlSet.URLs {
		if _, err := time.Parse("2006-01-02", u.LastMod); err != nil {
			t.Errorf("invalid lastmod %q for %s: %v", u.LastMod, u.Loc, err)
		}
	}
}

func TestGenerateSitemapWithBasePath(t *testing.T) {
	tmpDir := t.TempDir()
	g := &HTMLGenerator{}

	siteID := uuid.New()
	sectionMain := &Section{ID: uuid.New(), SiteID: siteID, Name: "main", Path: ""}
	sections := []*Section{sectionMain}

	now := time.Now()
	publishedAt := now.Add(-time.Hour)
	contents := []*Content{
		{
			ID:          uuid.New(),
			SiteID:      siteID,
			SectionID:   sectionMain.ID,
			ShortID:     "abc12345",
			Heading:     "Home Page",
			SectionPath: "",
			Kind:        "page",
			Draft:       false,
			PublishedAt: &publishedAt,
			UpdatedAt:   now,
		},
	}

	site := &Site{ID: siteID, Name: "Test", Slug: "test"}

	err := g.generateSitemap(tmpDir, "https://example.com", "/blog/", site, contents, sections)
	if err != nil {
		t.Fatalf("generateSitemap failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "sitemap.xml"))
	if err != nil {
		t.Fatalf("failed to read sitemap.xml: %v", err)
	}

	var urlSet sitemapURLSet
	if err := xml.Unmarshal(data, &urlSet); err != nil {
		t.Fatalf("invalid XML: %v", err)
	}

	// Homepage should include base path
	if urlSet.URLs[0].Loc != "https://example.com/blog/" {
		t.Errorf("homepage URL = %s, want https://example.com/blog/", urlSet.URLs[0].Loc)
	}

	// Content URL should include base path
	expectedSlug := Slugify("Home Page") + "-abc12345"
	expectedURL := "https://example.com/blog/" + expectedSlug + "/"
	if urlSet.URLs[1].Loc != expectedURL {
		t.Errorf("content URL = %s, want %s", urlSet.URLs[1].Loc, expectedURL)
	}
}

func TestGenerateCNAME(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		wantFile    bool
		wantContent string
	}{
		{
			name:        "standard domain",
			baseURL:     "https://example.com",
			wantFile:    true,
			wantContent: "example.com",
		},
		{
			name:        "domain with subdomain",
			baseURL:     "https://blog.example.com",
			wantFile:    true,
			wantContent: "blog.example.com",
		},
		{
			name:        "domain with path",
			baseURL:     "https://example.com/blog",
			wantFile:    true,
			wantContent: "example.com",
		},
		{
			name:     "localhost skipped",
			baseURL:  "http://localhost:8080",
			wantFile: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			g := &HTMLGenerator{}

			err := g.generateCNAME(tmpDir, tt.baseURL)
			if err != nil {
				t.Fatalf("generateCNAME failed: %v", err)
			}

			path := filepath.Join(tmpDir, "CNAME")
			data, err := os.ReadFile(path)
			if tt.wantFile {
				if err != nil {
					t.Fatalf("expected CNAME file but got error: %v", err)
				}
				if string(data) != tt.wantContent {
					t.Errorf("CNAME content = %q, want %q", string(data), tt.wantContent)
				}
			} else {
				if err == nil {
					t.Errorf("expected no CNAME file, but found one with content: %q", string(data))
				}
			}
		})
	}
}
