package ssg

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// HTMLGenerator handles static site generation.
type HTMLGenerator struct {
	workspace *Workspace
	processor *Processor
	assetsFS  embed.FS
}

// NewHTMLGenerator creates a new HTML generator.
func NewHTMLGenerator(workspace *Workspace, assetsFS embed.FS) *HTMLGenerator {
	return &HTMLGenerator{
		workspace: workspace,
		processor: NewProcessor(),
		assetsFS:  assetsFS,
	}
}

// PageData holds data for rendering a page.
type SSGPageData struct {
	Site        *Site
	Content     *RenderedContent
	Contents    []*RenderedContent
	Section     *Section
	Sections    []*Section
	Menu        []*Section
	IsIndex     bool
	IsPaginated bool
	CurrentPage int
	TotalPages  int
	HasPrev     bool
	HasNext     bool
	PrevURL     string
	NextURL     string
	AssetPath   string
	Params      map[string]string
}

// RenderedContent holds content with HTML body.
type RenderedContent struct {
	*Content
	HTMLBody template.HTML
	URL      string
}

// GenerateHTMLResult contains the result of HTML generation.
type GenerateHTMLResult struct {
	TotalContent   int
	PagesGenerated int
	IndexPages     int
	Errors         []string
}

// GenerateHTML generates the static HTML site.
func (g *HTMLGenerator) GenerateHTML(ctx context.Context, site *Site, contents []*Content, sections []*Section, params []*Param) (*GenerateHTMLResult, error) {
	result := &GenerateHTMLResult{
		TotalContent: len(contents),
	}

	htmlPath := g.workspace.GetHTMLPath(site.Slug)

	// Clean existing HTML files
	if err := CleanDir(htmlPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to clean html directory: %w", err)
	}

	// Copy static assets
	if err := g.copyStaticAssets(htmlPath); err != nil {
		return nil, fmt.Errorf("failed to copy static assets: %w", err)
	}

	// Parse templates
	tmpl, err := g.parseTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	// Build menu from sections
	menu := g.buildMenu(sections, site.Mode)

	// Build params map
	paramsMap := make(map[string]string)
	for _, p := range params {
		paramsMap[p.Name] = p.Value
	}

	// Render individual content pages
	for _, content := range contents {
		if content.Draft {
			continue
		}

		if err := g.renderContentPage(tmpl, htmlPath, site, content, sections, menu, paramsMap); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("content %s: %v", content.Heading, err))
			continue
		}
		result.PagesGenerated++
	}

	// Render index pages
	indexCount, err := g.renderIndexPages(tmpl, htmlPath, site, contents, sections, menu, paramsMap)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("index pages: %v", err))
	}
	result.IndexPages = indexCount

	return result, nil
}

// parseTemplates parses the SSG templates from embedded filesystem.
func (g *HTMLGenerator) parseTemplates() (*template.Template, error) {
	funcMap := template.FuncMap{
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
		"add":      func(a, b int) int { return a + b },
		"subtract": func(a, b int) int { return a - b },
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(g.assetsFS,
		"assets/ssg/layout.html",
		"assets/ssg/partials/*.html",
	)
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}

// copyStaticAssets copies static assets to the output directory.
func (g *HTMLGenerator) copyStaticAssets(htmlPath string) error {
	staticPath := filepath.Join(htmlPath, "static")
	if err := os.MkdirAll(staticPath, 0755); err != nil {
		return err
	}

	// Walk embedded static assets
	return fs.WalkDir(g.assetsFS, "assets/ssg/static", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path from assets/ssg/static
		relPath, _ := filepath.Rel("assets/ssg/static", path)
		destPath := filepath.Join(staticPath, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		// Copy file
		data, err := g.assetsFS.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(destPath, data, 0644)
	})
}

// buildMenu builds the navigation menu from sections.
func (g *HTMLGenerator) buildMenu(sections []*Section, mode string) []*Section {
	if mode == "blog" {
		// In blog mode, hide section menu
		return nil
	}

	var menu []*Section
	for _, s := range sections {
		if s.Name != "root" && s.Path != "/" && s.Path != "" {
			menu = append(menu, s)
		}
	}
	return menu
}

// renderContentPage renders a single content page.
func (g *HTMLGenerator) renderContentPage(tmpl *template.Template, htmlPath string, site *Site, content *Content, sections []*Section, menu []*Section, params map[string]string) error {
	// Process markdown to HTML
	htmlBody, err := g.processor.ProcessContent(content)
	if err != nil {
		return err
	}

	rendered := &RenderedContent{
		Content:  content,
		HTMLBody: template.HTML(htmlBody),
		URL:      g.getContentURL(content, site.Mode),
	}

	// Find section
	var section *Section
	for _, s := range sections {
		if s.ID == content.SectionID {
			section = s
			break
		}
	}

	data := SSGPageData{
		Site:      site,
		Content:   rendered,
		Section:   section,
		Sections:  sections,
		Menu:      menu,
		IsIndex:   false,
		AssetPath: g.getAssetPath(content.SectionPath),
		Params:    params,
	}

	// Determine output path
	outputPath := g.workspace.GetContentHTMLPath(site.Slug, content.SectionPath, content.Slug())
	if err := EnsureDir(outputPath); err != nil {
		return err
	}

	// Write file
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.ExecuteTemplate(f, "layout.html", data)
}

// renderIndexPages renders index pages with pagination.
func (g *HTMLGenerator) renderIndexPages(tmpl *template.Template, htmlPath string, site *Site, contents []*Content, sections []*Section, menu []*Section, params map[string]string) (int, error) {
	pageSize := 10
	count := 0

	// Filter non-draft content
	var publishedContents []*Content
	for _, c := range contents {
		if !c.Draft {
			publishedContents = append(publishedContents, c)
		}
	}

	// Render main index
	if err := g.renderIndex(tmpl, htmlPath, site, "", publishedContents, sections, menu, params, pageSize); err != nil {
		return count, err
	}
	count++

	// Render section indices
	for _, section := range sections {
		var sectionContents []*Content
		for _, c := range publishedContents {
			if c.SectionID == section.ID {
				sectionContents = append(sectionContents, c)
			}
		}

		if len(sectionContents) > 0 {
			if err := g.renderIndex(tmpl, htmlPath, site, section.Path, sectionContents, sections, menu, params, pageSize); err != nil {
				return count, err
			}
			count++
		}
	}

	return count, nil
}

// renderIndex renders an index page with pagination.
func (g *HTMLGenerator) renderIndex(tmpl *template.Template, htmlPath string, site *Site, indexPath string, contents []*Content, sections []*Section, menu []*Section, params map[string]string, pageSize int) error {
	totalPages := (len(contents) + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	for page := 1; page <= totalPages; page++ {
		start := (page - 1) * pageSize
		end := start + pageSize
		if end > len(contents) {
			end = len(contents)
		}

		pageContents := contents[start:end]

		// Render content previews
		var renderedContents []*RenderedContent
		for _, c := range pageContents {
			htmlBody, _ := g.processor.ProcessContent(c)
			renderedContents = append(renderedContents, &RenderedContent{
				Content:  c,
				HTMLBody: template.HTML(htmlBody),
				URL:      g.getContentURL(c, site.Mode),
			})
		}

		data := SSGPageData{
			Site:        site,
			Contents:    renderedContents,
			Sections:    sections,
			Menu:        menu,
			IsIndex:     true,
			IsPaginated: totalPages > 1,
			CurrentPage: page,
			TotalPages:  totalPages,
			HasPrev:     page > 1,
			HasNext:     page < totalPages,
			AssetPath:   g.getAssetPath(indexPath),
			Params:      params,
		}

		if page > 1 {
			data.PrevURL = g.getPaginationURL(indexPath, page-1)
		}
		if page < totalPages {
			data.NextURL = g.getPaginationURL(indexPath, page+1)
		}

		// Determine output path
		outputPath := g.workspace.GetPaginationHTMLPath(site.Slug, indexPath, page)
		if err := EnsureDir(outputPath); err != nil {
			return err
		}

		f, err := os.Create(outputPath)
		if err != nil {
			return err
		}

		if err := tmpl.ExecuteTemplate(f, "layout.html", data); err != nil {
			f.Close()
			return err
		}
		f.Close()
	}

	return nil
}

// getContentURL returns the URL for a content item.
func (g *HTMLGenerator) getContentURL(content *Content, mode string) string {
	if mode == "blog" {
		return "/" + content.Slug() + "/"
	}
	if content.SectionPath == "" || content.SectionPath == "/" {
		return "/" + content.Slug() + "/"
	}
	return "/" + content.SectionPath + "/" + content.Slug() + "/"
}

// getPaginationURL returns the URL for a pagination page.
func (g *HTMLGenerator) getPaginationURL(indexPath string, page int) string {
	if page == 1 {
		if indexPath == "" || indexPath == "/" {
			return "/"
		}
		return "/" + indexPath + "/"
	}
	if indexPath == "" || indexPath == "/" {
		return fmt.Sprintf("/page/%d/", page)
	}
	return fmt.Sprintf("/%s/page/%d/", indexPath, page)
}

// getAssetPath returns the relative path to assets from a content path.
func (g *HTMLGenerator) getAssetPath(contentPath string) string {
	if contentPath == "" || contentPath == "/" {
		return "./"
	}
	// Count depth and return appropriate number of ../
	depth := 1
	for _, ch := range contentPath {
		if ch == '/' {
			depth++
		}
	}
	path := ""
	for i := 0; i < depth; i++ {
		path += "../"
	}
	return path
}

// HTMLGeneratorService provides HTML generation functionality for the service layer.
type HTMLGeneratorService interface {
	GenerateHTML(ctx context.Context, siteID uuid.UUID) (*GenerateHTMLResult, error)
}
