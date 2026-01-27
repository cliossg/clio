package ssg

import (
	"context"
	"fmt"
	"time"

	"github.com/cliossg/clio/pkg/cl/logger"
	"github.com/google/uuid"
)

type Seeder struct {
	service Service
	log     logger.Logger
}

func NewSeeder(service Service, log logger.Logger) *Seeder {
	return &Seeder{
		service: service,
		log:     log,
	}
}

func (s *Seeder) Start(ctx context.Context) error {
	sites, err := s.service.ListSites(ctx)
	if err != nil {
		return fmt.Errorf("cannot list sites: %w", err)
	}

	if len(sites) > 0 {
		s.log.Info("Sites already exist, skipping SSG seeding")
		return nil
	}

	site, err := s.seedDemoSite(ctx)
	if err != nil {
		return fmt.Errorf("cannot seed demo site: %w", err)
	}

	if err := s.seedDemoContent(ctx, site); err != nil {
		return fmt.Errorf("cannot seed demo content: %w", err)
	}

	s.log.Infof("Seeded demo site: %s", site.Name)
	return nil
}

func (s *Seeder) seedDemoSite(ctx context.Context) (*Site, error) {
	site := NewSite("Demo Site", "demo", "blog")
	if err := s.service.CreateSite(ctx, site); err != nil {
		return nil, err
	}

	// Create root section
	root := NewSection(site.ID, "/ (root)", "Root section for top-level content", "")
	if err := s.service.CreateSection(ctx, root); err != nil {
		return nil, err
	}

	// Create blog section
	blog := NewSection(site.ID, "Blog", "Blog posts and articles", "blog")
	if err := s.service.CreateSection(ctx, blog); err != nil {
		return nil, err
	}

	return site, nil
}

func (s *Seeder) seedDemoContent(ctx context.Context, site *Site) error {
	sections, err := s.service.GetSections(ctx, site.ID)
	if err != nil {
		return err
	}

	var rootSection, blogSection *Section
	for _, sec := range sections {
		if sec.Path == "" {
			rootSection = sec
		} else if sec.Path == "blog" {
			blogSection = sec
		}
	}

	if rootSection == nil || blogSection == nil {
		return fmt.Errorf("sections not found")
	}

	now := time.Now()

	// Home page
	home := &Content{
		ID:        uuid.New(),
		SiteID:    site.ID,
		SectionID: rootSection.ID,
		ShortID:   uuid.New().String()[:8],
		Kind:      "page",
		Heading:   "Welcome to Clio",
		Body:      "# Welcome to Clio\n\nClio is a static site generator with a built-in admin interface.\n\n## Features\n\n- Markdown content editing\n- Live preview\n- Image management\n- Multiple sections\n- Tags and categories\n\nStart creating content using the admin panel.",
		Summary:   "Welcome page for the demo site",
		Draft:     false,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.service.CreateContent(ctx, home); err != nil {
		return err
	}

	// Blog posts
	posts := []struct {
		heading string
		body    string
		summary string
	}{
		{
			heading: "Getting Started with Clio",
			body:    "# Getting Started with Clio\n\nClio makes it easy to create and manage your static site.\n\n## Creating Content\n\n1. Navigate to **Contents** in the admin panel\n2. Click **New Content**\n3. Write your content in Markdown\n4. Save and preview\n\n## Generating Your Site\n\nOnce you're happy with your content:\n\n1. Go to your site dashboard\n2. Click **Generate HTML**\n3. Your static site is ready!\n\nHappy writing!",
			summary: "Learn how to create your first content with Clio",
		},
		{
			heading: "Markdown Tips and Tricks",
			body:    "# Markdown Tips and Tricks\n\nMarkdown is a lightweight markup language that's easy to learn.\n\n## Basic Formatting\n\n- **Bold**: `**text**`\n- *Italic*: `*text*`\n- `Code`: `` `code` ``\n\n## Lists\n\n```markdown\n- Item 1\n- Item 2\n  - Nested item\n```\n\n## Links and Images\n\n```markdown\n[Link text](url)\n![Alt text](image-url)\n```\n\n## Code Blocks\n\nUse triple backticks for code blocks with syntax highlighting.",
			summary: "Essential Markdown formatting for your content",
		},
	}

	for i, p := range posts {
		pubTime := now.Add(time.Duration(-i) * 24 * time.Hour)
		post := &Content{
			ID:          uuid.New(),
			SiteID:      site.ID,
			SectionID:   blogSection.ID,
			ShortID:     uuid.New().String()[:8],
			Kind:        "article",
			Heading:     p.heading,
			Body:        p.body,
			Summary:     p.summary,
			Draft:       false,
			PublishedAt: &pubTime,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := s.service.CreateContent(ctx, post); err != nil {
			return err
		}
	}

	return nil
}

func (s *Seeder) Name() string {
	return "ssg"
}

func (s *Seeder) Depends() []string {
	return []string{"auth"}
}
