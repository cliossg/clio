package ssg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// Generator handles markdown generation from database content.
type Generator struct {
	workspace *Workspace
}

// NewGenerator creates a new markdown generator.
func NewGenerator(workspace *Workspace) *Generator {
	return &Generator{
		workspace: workspace,
	}
}

// ContentFrontmatter represents the YAML frontmatter for a content file.
type ContentFrontmatter struct {
	Title           string     `yaml:"title"`
	Slug            string     `yaml:"slug"`
	ShortID         string     `yaml:"short-id,omitempty"`
	Section         string     `yaml:"section,omitempty"`
	Author          string     `yaml:"author,omitempty"`
	Contributor     string     `yaml:"contributor,omitempty"`
	Tags            []string   `yaml:"tags,omitempty"`
	Layout          string     `yaml:"layout,omitempty"`
	Draft           bool       `yaml:"draft"`
	Featured        bool       `yaml:"featured"`
	Summary         string     `yaml:"summary,omitempty"`
	Description     string     `yaml:"description,omitempty"`
	Image           string     `yaml:"image,omitempty"`
	SocialImage     string     `yaml:"social-image,omitempty"`
	PublishedAt     *time.Time `yaml:"published-at,omitempty"`
	CreatedAt       time.Time  `yaml:"created-at"`
	UpdatedAt       time.Time  `yaml:"updated-at"`
	Robots          string     `yaml:"robots,omitempty"`
	Keywords        string     `yaml:"keywords,omitempty"`
	CanonicalURL    string     `yaml:"canonical-url,omitempty"`
	Sitemap         string     `yaml:"sitemap,omitempty"`
	TableOfContents bool       `yaml:"table-of-contents,omitempty"`
	Comments        bool       `yaml:"comments,omitempty"`
	Share           bool       `yaml:"share,omitempty"`
	Kind            string     `yaml:"kind,omitempty"`
	Series          string     `yaml:"series,omitempty"`
	SeriesOrder     int        `yaml:"series-order,omitempty"`
}

// GenerateMarkdownResult contains the result of markdown generation.
type GenerateMarkdownResult struct {
	TotalContent   int
	FilesGenerated int
	Errors         []string
}

// GenerateMarkdown generates markdown files for all content in a site.
func (g *Generator) GenerateMarkdown(ctx context.Context, siteSlug string, contents []*Content) (*GenerateMarkdownResult, error) {
	result := &GenerateMarkdownResult{
		TotalContent: len(contents),
	}

	basePath := g.workspace.GetMarkdownPath(siteSlug)

	// Clean existing markdown files
	if err := CleanDir(basePath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to clean markdown directory: %w", err)
	}

	for _, content := range contents {
		err := g.generateContentMarkdown(basePath, content)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("content %s: %v", content.Heading, err))
			continue
		}
		result.FilesGenerated++
	}

	return result, nil
}

// generateContentMarkdown generates a single markdown file for a content item.
func (g *Generator) generateContentMarkdown(basePath string, content *Content) error {
	frontmatter := ContentFrontmatter{
		Title:       content.Heading,
		Slug:        content.Slug(),
		ShortID:     content.ShortID,
		Section:     content.SectionPath,
		Author:      content.AuthorUsername,
		Contributor: content.ContributorHandle,
		Layout:      content.SectionName,
		Draft:       content.Draft,
		Featured:    content.Featured,
		Summary:     content.Summary,
		Image:       content.HeaderImageURL,
		SocialImage: content.HeaderImageURL,
		PublishedAt: content.PublishedAt,
		CreatedAt:   content.CreatedAt,
		UpdatedAt:   content.UpdatedAt,
		Kind:        content.Kind,
		Series:      content.Series,
		SeriesOrder: content.SeriesOrder,
	}

	if content.Meta != nil {
		frontmatter.Description = content.Meta.Description
		frontmatter.Robots = content.Meta.Robots
		frontmatter.Keywords = content.Meta.Keywords
		frontmatter.CanonicalURL = content.Meta.CanonicalURL
		frontmatter.Sitemap = content.Meta.Sitemap
		frontmatter.TableOfContents = content.Meta.TableOfContents
		frontmatter.Comments = content.Meta.Comments
		frontmatter.Share = content.Meta.Share
	}

	for _, tag := range content.Tags {
		frontmatter.Tags = append(frontmatter.Tags, tag.Name)
	}

	// Marshal frontmatter to YAML
	yamlBytes, err := yaml.Marshal(frontmatter)
	if err != nil {
		return fmt.Errorf("cannot marshal frontmatter: %w", err)
	}

	// Build file content
	fileContent := fmt.Sprintf("---\n%s---\n\n%s", string(yamlBytes), content.Body)

	// Determine file path
	sectionPath := content.SectionPath
	if sectionPath == "" {
		sectionPath = "posts" // Default section
	}
	fileName := content.Slug() + ".md"
	filePath := filepath.Join(basePath, sectionPath, fileName)

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("cannot create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(filePath, []byte(fileContent), 0644); err != nil {
		return fmt.Errorf("cannot write file: %w", err)
	}

	return nil
}

// GeneratorService provides generation functionality for the service layer.
type GeneratorService interface {
	GenerateMarkdown(ctx context.Context, siteID uuid.UUID) (*GenerateMarkdownResult, error)
}
