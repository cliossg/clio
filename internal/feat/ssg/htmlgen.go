package ssg

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"time"

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

// SSGPageData holds data for rendering a page.
type SSGPageData struct {
	Site        *Site
	Content     *RenderedContent
	Contents    []*RenderedContent
	Section     *Section
	Sections    []*Section
	Menu        []*Section
	Author      *Contributor
	Blocks      *GeneratedBlocks
	IsIndex     bool
	IsAuthor    bool
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
	AuthorPages    int
	Errors         []string
}

// GenerateHTML generates the static HTML site.
func (g *HTMLGenerator) GenerateHTML(ctx context.Context, site *Site, contents []*Content, sections []*Section, params []*Param, contributors []*Contributor, userAuthors map[string]*Contributor) (*GenerateHTMLResult, error) {
	result := &GenerateHTMLResult{
		TotalContent: len(contents),
	}

	htmlPath := g.workspace.GetHTMLPath(site.Slug)

	if err := CleanDir(htmlPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to clean html directory: %w", err)
	}

	if err := g.copyStaticAssets(htmlPath); err != nil {
		return nil, fmt.Errorf("failed to copy static assets: %w", err)
	}

	tmpl, err := g.parseTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	menu := g.buildMenu(sections, site.Mode)

	paramsMap := make(map[string]string)
	for _, p := range params {
		paramsMap[p.RefKey] = p.Value
	}

	basePath := g.getAssetPath(paramsMap)
	allRendered := g.preRenderAllContent(contents, basePath)

	blocksCfg := BlocksConfig{
		Enabled:      paramsMap["ssg.blocks.enabled"] != "false",
		MultiSection: paramsMap["ssg.blocks.multisection"] != "false",
		MaxItems:     5,
	}
	if v, ok := paramsMap["ssg.blocks.maxitems"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			blocksCfg.MaxItems = n
		}
	}

	for _, content := range contents {
		if content.Draft {
			continue
		}

		if err := g.renderContentPage(tmpl, htmlPath, site, content, sections, menu, paramsMap, allRendered, blocksCfg); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("content %s: %v", content.Heading, err))
			continue
		}
		result.PagesGenerated++
	}

	indexCount, err := g.renderIndexPages(tmpl, htmlPath, site, contents, sections, menu, paramsMap)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("index pages: %v", err))
	}
	result.IndexPages = indexCount

	authorCount, err := g.renderAuthorPages(tmpl, htmlPath, site, contents, contributors, userAuthors, menu, paramsMap)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("author pages: %v", err))
	}
	result.AuthorPages = authorCount

	if err := g.copyProfilePhotos(htmlPath, contributors, userAuthors); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("profile photos: %v", err))
	}

	return result, nil
}

// parseTemplates parses the SSG templates from embedded filesystem.
func (g *HTMLGenerator) parseTemplates() (*template.Template, error) {
	funcMap := template.FuncMap{
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
		"add":      func(a, b int) int { return a + b },
		"subtract": func(a, b int) int { return a - b },
		"now":      func() time.Time { return time.Now() },
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

func (g *HTMLGenerator) copyProfilePhotos(htmlPath string, contributors []*Contributor, userAuthors map[string]*Contributor) error {
	profilesPath := filepath.Join(htmlPath, "profiles")
	if err := os.MkdirAll(profilesPath, 0755); err != nil {
		return err
	}

	copied := make(map[string]bool)

	for _, c := range contributors {
		if c.PhotoPath == "" || copied[c.PhotoPath] {
			continue
		}

		srcPath := filepath.Join("_workspace", "profiles", c.PhotoPath)
		dstPath := filepath.Join(profilesPath, c.PhotoPath)

		data, err := os.ReadFile(srcPath)
		if err != nil {
			continue
		}

		if err := os.WriteFile(dstPath, data, 0644); err != nil {
			return err
		}
		copied[c.PhotoPath] = true
	}

	for _, u := range userAuthors {
		if u == nil || u.PhotoPath == "" || copied[u.PhotoPath] {
			continue
		}

		srcPath := filepath.Join("_workspace", "profiles", u.PhotoPath)
		dstPath := filepath.Join(profilesPath, u.PhotoPath)

		data, err := os.ReadFile(srcPath)
		if err != nil {
			continue
		}

		if err := os.WriteFile(dstPath, data, 0644); err != nil {
			return err
		}
		copied[u.PhotoPath] = true
	}

	return nil
}

// buildMenu builds the navigation menu from sections.
func (g *HTMLGenerator) buildMenu(sections []*Section, mode string) []*Section {
	var menu []*Section
	for _, s := range sections {
		if s.Name != "/ (root)" && s.Path != "/" && s.Path != "" {
			menu = append(menu, s)
		}
	}
	return menu
}

func (g *HTMLGenerator) preRenderAllContent(contents []*Content, basePath string) []*RenderedContent {
	var rendered []*RenderedContent
	for _, c := range contents {
		if c.Draft {
			continue
		}
		htmlBody, _ := g.processor.ProcessContent(c)
		rendered = append(rendered, &RenderedContent{
			Content:  c,
			HTMLBody: template.HTML(htmlBody),
			URL:      g.getContentURL(c, basePath),
		})
	}
	return rendered
}

// renderContentPage renders a single content page.
func (g *HTMLGenerator) renderContentPage(tmpl *template.Template, htmlPath string, site *Site, content *Content, sections []*Section, menu []*Section, params map[string]string, allRendered []*RenderedContent, blocksCfg BlocksConfig) error {
	basePath := g.getAssetPath(params)

	var rendered *RenderedContent
	for _, r := range allRendered {
		if r.ID == content.ID {
			rendered = r
			break
		}
	}
	if rendered == nil {
		htmlBody, err := g.processor.ProcessContent(content)
		if err != nil {
			return err
		}
		rendered = &RenderedContent{
			Content:  content,
			HTMLBody: template.HTML(htmlBody),
			URL:      g.getContentURL(content, basePath),
		}
	}

	var section *Section
	for _, s := range sections {
		if s.ID == content.SectionID {
			section = s
			break
		}
	}

	blocks := BuildBlocks(rendered, allRendered, blocksCfg)

	data := SSGPageData{
		Site:      site,
		Content:   rendered,
		Section:   section,
		Sections:  sections,
		Menu:      menu,
		Blocks:    blocks,
		IsIndex:   false,
		AssetPath: basePath,
		Params:    params,
	}

	outputPath := g.workspace.GetContentHTMLPath(site.Slug, content.SectionPath, content.Slug())
	if err := EnsureDir(outputPath); err != nil {
		return err
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.ExecuteTemplate(f, "layout.html", data)
}

// renderIndexPages renders index pages with pagination.
func (g *HTMLGenerator) renderIndexPages(tmpl *template.Template, htmlPath string, site *Site, contents []*Content, sections []*Section, menu []*Section, params map[string]string) (int, error) {
	pageSize := 9
	if v, ok := params["ssg.index.maxitems"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			pageSize = n
		}
	}
	count := 0

	// Filter non-draft articles (exclude pages from index listings)
	var publishedContents []*Content
	for _, c := range contents {
		if !c.Draft && c.Kind != "page" {
			publishedContents = append(publishedContents, c)
		}
	}

	// Render main index
	if err := g.renderIndex(tmpl, htmlPath, site, "", publishedContents, sections, menu, params, pageSize); err != nil {
		return count, err
	}
	count++

	// Render section indices (skip root section to avoid overwriting main index)
	for _, section := range sections {
		if section.Path == "" || section.Path == "/" {
			continue
		}

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

	basePath := g.getAssetPath(params)

	for page := 1; page <= totalPages; page++ {
		start := (page - 1) * pageSize
		end := start + pageSize
		if end > len(contents) {
			end = len(contents)
		}

		pageContents := contents[start:end]

		var renderedContents []*RenderedContent
		for _, c := range pageContents {
			htmlBody, _ := g.processor.ProcessContent(c)
			renderedContents = append(renderedContents, &RenderedContent{
				Content:  c,
				HTMLBody: template.HTML(htmlBody),
				URL:      g.getContentURL(c, basePath),
			})
		}

		var currentSection *Section
		for _, s := range sections {
			if s.Path == indexPath {
				currentSection = s
				break
			}
		}

		data := SSGPageData{
			Site:        site,
			Contents:    renderedContents,
			Section:     currentSection,
			Sections:    sections,
			Menu:        menu,
			IsIndex:     true,
			IsPaginated: totalPages > 1,
			CurrentPage: page,
			TotalPages:  totalPages,
			HasPrev:     page > 1,
			HasNext:     page < totalPages,
			AssetPath:   basePath,
			Params:      params,
		}

		if page > 1 {
			data.PrevURL = g.getPaginationURL(basePath, indexPath, page-1)
		}
		if page < totalPages {
			data.NextURL = g.getPaginationURL(basePath, indexPath, page+1)
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
func (g *HTMLGenerator) getContentURL(content *Content, basePath string) string {
	if content.SectionPath == "" || content.SectionPath == "/" {
		return basePath + content.Slug() + "/"
	}
	return basePath + content.SectionPath + "/" + content.Slug() + "/"
}

// getPaginationURL returns the URL for a pagination page.
func (g *HTMLGenerator) getPaginationURL(basePath, indexPath string, page int) string {
	if page == 1 {
		if indexPath == "" || indexPath == "/" {
			return basePath
		}
		return basePath + indexPath + "/"
	}
	if indexPath == "" || indexPath == "/" {
		return fmt.Sprintf("%spage/%d/", basePath, page)
	}
	return fmt.Sprintf("%s%s/page/%d/", basePath, indexPath, page)
}

func (g *HTMLGenerator) getAssetPath(params map[string]string) string {
	if basePath, ok := params["ssg.site.base_path"]; ok && basePath != "" {
		if basePath[0] != '/' {
			basePath = "/" + basePath
		}
		if basePath[len(basePath)-1] != '/' {
			basePath = basePath + "/"
		}
		return basePath
	}
	return "/"
}

func (g *HTMLGenerator) renderAuthorPages(tmpl *template.Template, htmlPath string, site *Site, contents []*Content, contributors []*Contributor, userAuthors map[string]*Contributor, menu []*Section, params map[string]string) (int, error) {
	count := 0
	generatedHandles := make(map[string]bool)
	basePath := g.getAssetPath(params)

	for _, contributor := range contributors {
		authorContents := g.getContentsByAuthor(contents, contributor.Handle)
		generatedHandles[contributor.Handle] = true

		var renderedContents []*RenderedContent
		for _, c := range authorContents {
			htmlBody, _ := g.processor.ProcessContent(c)
			renderedContents = append(renderedContents, &RenderedContent{
				Content:  c,
				HTMLBody: template.HTML(htmlBody),
				URL:      g.getContentURL(c, basePath),
			})
		}

		data := SSGPageData{
			Site:      site,
			Author:    contributor,
			Contents:  renderedContents,
			Menu:      menu,
			IsAuthor:  true,
			AssetPath: basePath,
			Params:    params,
		}

		outputPath := filepath.Join(htmlPath, "authors", contributor.Handle, "index.html")
		if err := EnsureDir(outputPath); err != nil {
			return count, err
		}

		f, err := os.Create(outputPath)
		if err != nil {
			return count, err
		}

		if err := tmpl.ExecuteTemplate(f, "layout.html", data); err != nil {
			f.Close()
			return count, err
		}
		f.Close()
		count++
	}

	usernames := g.getUniqueUserAuthors(contents, generatedHandles)
	for _, username := range usernames {
		authorContents := g.getContentsByAuthor(contents, username)

		var renderedContents []*RenderedContent
		for _, c := range authorContents {
			htmlBody, _ := g.processor.ProcessContent(c)
			renderedContents = append(renderedContents, &RenderedContent{
				Content:  c,
				HTMLBody: template.HTML(htmlBody),
				URL:      g.getContentURL(c, basePath),
			})
		}

		userAuthor := userAuthors[username]
		if userAuthor == nil {
			userAuthor = &Contributor{
				Handle: username,
				Name:   username,
			}
		}

		data := SSGPageData{
			Site:      site,
			Author:    userAuthor,
			Contents:  renderedContents,
			Menu:      menu,
			IsAuthor:  true,
			AssetPath: basePath,
			Params:    params,
		}

		outputPath := filepath.Join(htmlPath, "authors", username, "index.html")
		if err := EnsureDir(outputPath); err != nil {
			return count, err
		}

		f, err := os.Create(outputPath)
		if err != nil {
			return count, err
		}

		if err := tmpl.ExecuteTemplate(f, "layout.html", data); err != nil {
			f.Close()
			return count, err
		}
		f.Close()
		count++
	}

	return count, nil
}

func (g *HTMLGenerator) getUniqueUserAuthors(contents []*Content, excludeHandles map[string]bool) []string {
	seen := make(map[string]bool)
	var result []string
	for _, c := range contents {
		if c.AuthorUsername != "" && !excludeHandles[c.AuthorUsername] && !seen[c.AuthorUsername] {
			seen[c.AuthorUsername] = true
			result = append(result, c.AuthorUsername)
		}
	}
	return result
}

func (g *HTMLGenerator) getContentsByAuthor(contents []*Content, handle string) []*Content {
	var result []*Content
	for _, c := range contents {
		if c.ContributorHandle == handle || c.AuthorUsername == handle {
			result = append(result, c)
		}
	}
	return result
}

// HTMLGeneratorService provides HTML generation functionality for the service layer.
type HTMLGeneratorService interface {
	GenerateHTML(ctx context.Context, siteID uuid.UUID) (*GenerateHTMLResult, error)
}
