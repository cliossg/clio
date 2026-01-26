package ssg

import (
	"fmt"
	"os"
	"path/filepath"
)

// Default workspace paths
const (
	DefaultSitesBasePath = "_workspace/sites"
)

// Workspace handles site directory operations.
type Workspace struct {
	basePath string
}

// NewWorkspace creates a new workspace manager.
func NewWorkspace(basePath string) *Workspace {
	if basePath == "" {
		basePath = DefaultSitesBasePath
	}
	return &Workspace{basePath: basePath}
}

// CreateSiteDirectories creates the directory structure for a site.
// Structure:
//
//	_workspace/sites/{slug}/
//	├── markdown/      # Archivos .md generados
//	├── html/          # Archivos .html generados
//	└── images/        # Imágenes del site
func (w *Workspace) CreateSiteDirectories(slug string) error {
	dirs := []string{
		w.GetMarkdownPath(slug),
		w.GetHTMLPath(slug),
		w.GetImagesPath(slug),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// DeleteSiteDirectories removes the directory structure for a site.
func (w *Workspace) DeleteSiteDirectories(slug string) error {
	siteBase := w.GetSiteBasePath(slug)
	if err := os.RemoveAll(siteBase); err != nil {
		return fmt.Errorf("failed to remove site directories: %w", err)
	}
	return nil
}

// SiteDirectoriesExist checks if the site directories exist.
func (w *Workspace) SiteDirectoriesExist(slug string) bool {
	siteBase := w.GetSiteBasePath(slug)
	_, err := os.Stat(siteBase)
	return err == nil
}

// Path helper functions

// GetSiteBasePath returns the base path for a specific site.
// e.g., _workspace/sites/my-blog
func (w *Workspace) GetSiteBasePath(slug string) string {
	return filepath.Join(w.basePath, slug)
}

// GetMarkdownPath returns the markdown output path for a specific site.
// e.g., _workspace/sites/my-blog/markdown
func (w *Workspace) GetMarkdownPath(slug string) string {
	return filepath.Join(w.basePath, slug, "markdown")
}

// GetHTMLPath returns the HTML output path for a specific site.
// e.g., _workspace/sites/my-blog/html
func (w *Workspace) GetHTMLPath(slug string) string {
	return filepath.Join(w.basePath, slug, "html")
}

// GetImagesPath returns the images path for a specific site.
// e.g., _workspace/sites/my-blog/images
func (w *Workspace) GetImagesPath(slug string) string {
	return filepath.Join(w.basePath, slug, "images")
}

// GetStaticPath returns the static assets path inside HTML output.
// e.g., _workspace/sites/my-blog/html/static
func (w *Workspace) GetStaticPath(slug string) string {
	return filepath.Join(w.GetHTMLPath(slug), "static")
}

// GetContentMarkdownPath returns the path for a content's markdown file.
// e.g., _workspace/sites/my-blog/markdown/posts/my-post.md
func (w *Workspace) GetContentMarkdownPath(slug, sectionPath, contentSlug string) string {
	return filepath.Join(w.GetMarkdownPath(slug), sectionPath, contentSlug+".md")
}

// GetContentHTMLPath returns the path for a content's HTML file.
// e.g., _workspace/sites/my-blog/html/posts/my-post/index.html
func (w *Workspace) GetContentHTMLPath(slug, sectionPath, contentSlug string) string {
	return filepath.Join(w.GetHTMLPath(slug), sectionPath, contentSlug, "index.html")
}

// GetIndexHTMLPath returns the path for an index HTML file.
// e.g., _workspace/sites/my-blog/html/posts/index.html
func (w *Workspace) GetIndexHTMLPath(slug, path string) string {
	if path == "/" || path == "" {
		return filepath.Join(w.GetHTMLPath(slug), "index.html")
	}
	return filepath.Join(w.GetHTMLPath(slug), path, "index.html")
}

// GetPaginationHTMLPath returns the path for a paginated index HTML file.
// e.g., _workspace/sites/my-blog/html/posts/page/2/index.html
func (w *Workspace) GetPaginationHTMLPath(slug, path string, page int) string {
	if page == 1 {
		return w.GetIndexHTMLPath(slug, path)
	}
	if path == "/" || path == "" {
		return filepath.Join(w.GetHTMLPath(slug), "page", fmt.Sprintf("%d", page), "index.html")
	}
	return filepath.Join(w.GetHTMLPath(slug), path, "page", fmt.Sprintf("%d", page), "index.html")
}

// GetSectionImagesPath returns the path for section-specific images.
// e.g., _workspace/sites/my-blog/images/posts/
func (w *Workspace) GetSectionImagesPath(slug, sectionPath string) string {
	return filepath.Join(w.GetImagesPath(slug), sectionPath)
}

// EnsureDir creates a directory if it doesn't exist.
func EnsureDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0755)
}

// CleanDir removes all contents of a directory but keeps the directory itself.
func CleanDir(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())
		if err := os.RemoveAll(entryPath); err != nil {
			return fmt.Errorf("failed to remove %s: %w", entryPath, err)
		}
	}

	return nil
}
