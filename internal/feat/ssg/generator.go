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
	Title       string     `yaml:"title"`
	Slug        string     `yaml:"slug"`
	Draft       bool       `yaml:"draft"`
	Featured    bool       `yaml:"featured"`
	Tags        []string   `yaml:"tags,omitempty"`
	Section     string     `yaml:"section,omitempty"`
	Kind        string     `yaml:"kind,omitempty"`
	Series      string     `yaml:"series,omitempty"`
	SeriesOrder int        `yaml:"series_order,omitempty"`
	Summary     string     `yaml:"summary,omitempty"`
	PublishedAt *time.Time `yaml:"published_at,omitempty"`
	CreatedAt   time.Time  `yaml:"created_at"`
	UpdatedAt   time.Time  `yaml:"updated_at"`
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
	// Build frontmatter
	frontmatter := ContentFrontmatter{
		Title:       content.Heading,
		Slug:        content.Slug(),
		Draft:       content.Draft,
		Featured:    content.Featured,
		Section:     content.SectionName,
		Kind:        content.Kind,
		Series:      content.Series,
		SeriesOrder: content.SeriesOrder,
		Summary:     content.Summary,
		PublishedAt: content.PublishedAt,
		CreatedAt:   content.CreatedAt,
		UpdatedAt:   content.UpdatedAt,
	}

	// Extract tag names
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
