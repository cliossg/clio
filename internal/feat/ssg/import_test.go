package ssg

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

func ptrUUID() *uuid.UUID {
	id := uuid.New()
	return &id
}

func TestYAMLDateParsing(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantOK bool
	}{
		{
			name:   "with nanoseconds and offset",
			input:  "created-at: 2026-01-28T21:24:34.700296974+01:00",
			wantOK: true,
		},
		{
			name:   "simple Z format",
			input:  "created-at: 2026-01-09T19:37:45Z",
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m map[string]interface{}
			if err := yaml.Unmarshal([]byte(tt.input), &m); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}

			val := m["created-at"]
			_, ok := val.(time.Time)
			t.Logf("type=%T value=%v isTime=%v", val, val, ok)

			if ok != tt.wantOK {
				t.Errorf("got time.Time=%v, want %v", ok, tt.wantOK)
			}
		})
	}
}

func TestParseFrontmatterBuenosAires(t *testing.T) {
	content := `---
title: 'Buenos Aires: A Love Letter'
slug: buenos-aires-a-love-letter-smpl005
short-id: smpl005
section: places
contributor: johndoe
layout: Places
draft: false
featured: false
summary: Tango, steak, and conversations that last until dawn.
image: /images/colorful-la-boca.png
social-image: /images/colorful-la-boca.png
published-at: 2025-12-17T19:57:16Z
created-at: 2026-01-09T19:37:45Z
updated-at: 2026-01-09T19:37:45Z
kind: article
---

Content coming soon.`

	fm, body := parseFrontmatter(content)

	t.Logf("frontmatter keys: %v", fm)
	t.Logf("body: %q", body[:20])

	if _, ok := fm["created-at"]; !ok {
		t.Error("created-at not found in frontmatter")
	} else {
		t.Logf("created-at value: %q", fm["created-at"])
	}
}

func TestImportFlowBuenosAires(t *testing.T) {
	content := `---
title: 'Buenos Aires: A Love Letter'
slug: buenos-aires-a-love-letter-smpl005
short-id: smpl005
section: places
contributor: johndoe
layout: Places
draft: false
featured: false
summary: Tango, steak, and conversations that last until dawn.
image: /images/colorful-la-boca.png
social-image: /images/colorful-la-boca.png
published-at: 2025-12-17T19:57:16Z
created-at: 2026-01-09T19:37:45Z
updated-at: 2026-01-09T19:37:45Z
kind: article
---

Content coming soon.`

	// Step 1: parseFrontmatter (like ImportScanner.parseFile does)
	fm, _ := parseFrontmatter(content)
	t.Logf("Step 1 - parseFrontmatter: created-at = %q", fm["created-at"])

	// Step 2: joinFrontmatter + ParseTypedFrontmatter (like ImportFile does)
	joined := "---\n" + joinFrontmatter(fm) + "\n---\n"
	t.Logf("Step 2 - joined frontmatter:\n%s", joined)

	typedFM, _, err := ParseTypedFrontmatter(joined)
	if err != nil {
		t.Fatalf("ParseTypedFrontmatter error: %v", err)
	}

	t.Logf("Step 3 - typedFM.CreatedAt: %v (nil=%v)", typedFM.CreatedAt, typedFM.CreatedAt == nil)

	if typedFM.CreatedAt == nil {
		t.Error("typedFM.CreatedAt is nil - dates not being parsed!")
	} else {
		expected := time.Date(2026, 1, 9, 19, 37, 45, 0, time.UTC)
		if !typedFM.CreatedAt.Equal(expected) {
			t.Errorf("CreatedAt = %v, want %v", typedFM.CreatedAt, expected)
		}
	}
}


func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantKeys []string
		wantBody string
	}{
		{
			name: "basic frontmatter",
			content: `---
title: Test Post
slug: test-post
---
Body content`,
			wantKeys: []string{"title", "slug"},
			wantBody: "Body content",
		},
		{
			name:     "no frontmatter",
			content:  "Just body content",
			wantKeys: nil,
			wantBody: "Just body content",
		},
		{
			name: "unclosed frontmatter",
			content: `---
title: Test
Body without closing`,
			wantKeys: nil,
			wantBody: "---\ntitle: Test\nBody without closing",
		},
		{
			name: "frontmatter with dates",
			content: `---
title: Dated Post
created-at: 2025-01-15T10:30:00Z
published-at: 2025-01-16T14:00:00Z
---
Body`,
			wantKeys: []string{"title", "created-at", "published-at"},
			wantBody: "Body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body := parseFrontmatter(tt.content)

			for _, key := range tt.wantKeys {
				if _, ok := fm[key]; !ok {
					t.Errorf("missing expected key: %s", key)
				}
			}

			if body != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestParseTypedFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantTitle   string
		wantSection string
		wantTags    []string
		wantErr     bool
	}{
		{
			name: "basic fields",
			content: `---
title: My Post
section: blog
---
Body`,
			wantTitle:   "My Post",
			wantSection: "blog",
		},
		{
			name: "with tags array",
			content: `---
title: Tagged Post
tags:
  - go
  - testing
---
Body`,
			wantTitle: "Tagged Post",
			wantTags:  []string{"go", "testing"},
		},
		{
			name:    "no frontmatter",
			content: "Just body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, _, err := ParseTypedFrontmatter(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if fm == nil {
				if tt.wantTitle != "" {
					t.Error("expected frontmatter, got nil")
				}
				return
			}

			if fm.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", fm.Title, tt.wantTitle)
			}
			if fm.Section != tt.wantSection {
				t.Errorf("Section = %q, want %q", fm.Section, tt.wantSection)
			}
			if len(tt.wantTags) > 0 {
				if len(fm.Tags) != len(tt.wantTags) {
					t.Errorf("Tags = %v, want %v", fm.Tags, tt.wantTags)
				}
			}
		})
	}
}

func TestParseTypedFrontmatterDates(t *testing.T) {
	content := `---
title: Post with Dates
created-at: 2025-01-15T10:30:00Z
published-at: 2025-01-16T14:00:00Z
---
Body`

	fm, body, err := ParseTypedFrontmatter(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm == nil {
		t.Fatal("expected frontmatter, got nil")
	}

	if fm.Title != "Post with Dates" {
		t.Errorf("Title = %q, want %q", fm.Title, "Post with Dates")
	}

	if fm.CreatedAt == nil {
		t.Fatal("CreatedAt is nil")
	}

	expectedCreated := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	if !fm.CreatedAt.Equal(expectedCreated) {
		t.Errorf("CreatedAt = %v, want %v", fm.CreatedAt, expectedCreated)
	}

	if fm.PublishedAt == nil {
		t.Fatal("PublishedAt is nil")
	}

	expectedPublished := time.Date(2025, 1, 16, 14, 0, 0, 0, time.UTC)
	if !fm.PublishedAt.Equal(expectedPublished) {
		t.Errorf("PublishedAt = %v, want %v", fm.PublishedAt, expectedPublished)
	}

	if body != "Body" {
		t.Errorf("body = %q, want %q", body, "Body")
	}
}

func TestExtractFirstH1(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"basic h1", "# Hello World", "Hello World"},
		{"h1 with whitespace", "#   Spaced Title  ", "Spaced Title"},
		{"no h1", "No heading here", ""},
		{"h2 not h1", "## Not H1", ""},
		{"h1 in middle", "Text\n# Middle H1\nMore text", "Middle H1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFirstH1(tt.content)
			if got != tt.want {
				t.Errorf("extractFirstH1() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractImagePaths(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{
			name: "markdown image",
			body: "![Alt](/images/photo.jpg)",
			want: 1,
		},
		{
			name: "html image",
			body: `<img src="/images/photo.jpg" alt="Alt">`,
			want: 1,
		},
		{
			name: "multiple images",
			body: "![A](/images/a.jpg)\n![B](/images/b.png)",
			want: 2,
		},
		{
			name: "no images",
			body: "Just text",
			want: 0,
		},
		{
			name: "external image ignored",
			body: "![Alt](https://example.com/photo.jpg)",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractImagePaths(tt.body)
			if len(got) != tt.want {
				t.Errorf("ExtractImagePaths() returned %d paths, want %d", len(got), tt.want)
			}
		})
	}
}

func TestDetectImportType(t *testing.T) {
	tests := []struct {
		name string
		path string
		want ImportType
	}{
		{"nonexistent path", "/nonexistent/path", ImportTypePoor},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectImportType(tt.path)
			if got != tt.want {
				t.Errorf("DetectImportType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectImportTypeFromFiles(t *testing.T) {
	tests := []struct {
		name  string
		files []ImportFile
		want  ImportType
	}{
		{
			name:  "empty files",
			files: nil,
			want:  ImportTypePoor,
		},
		{
			name: "file with section",
			files: []ImportFile{
				{Frontmatter: map[string]string{"section": "blog"}},
			},
			want: ImportTypeBasic,
		},
		{
			name: "file without section",
			files: []ImportFile{
				{Frontmatter: map[string]string{"title": "Test"}},
			},
			want: ImportTypePoor,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectImportTypeFromFiles(tt.files)
			if got != tt.want {
				t.Errorf("DetectImportTypeFromFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComputeImportStatus(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name      string
		imp       *Import
		fileMtime time.Time
		want      string
	}{
		{
			name: "no content - pending",
			imp:  &Import{ContentID: nil},
			want: ImportStatusPending,
		},
		{
			name: "file not modified - synced",
			imp: &Import{
				ContentID: ptrUUID(),
				FileMtime: &past,
			},
			want:      ImportStatusSynced,
			fileMtime: past,
		},
		{
			name: "file modified - updated",
			imp: &Import{
				ContentID: ptrUUID(),
				FileMtime: &past,
			},
			want:      ImportStatusUpdated,
			fileMtime: future,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeImportStatus(tt.imp, tt.fileMtime)
			if got != tt.want {
				t.Errorf("ComputeImportStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetImportPath(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
		siteSlug string
		want     string
	}{
		{
			name:     "with base path",
			basePath: "/custom/path",
			siteSlug: "mysite",
			want:     "/custom/path/Clio/mysite",
		},
		{
			name:     "empty base path uses default",
			basePath: "",
			siteSlug: "mysite",
			want:     "~/Documents/Clio/mysite",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetImportPath(tt.basePath, tt.siteSlug)
			if got != tt.want {
				t.Errorf("GetImportPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
