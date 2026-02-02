package ssg

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// ImportScanner scans directories for markdown files to import.
type ImportScanner struct {
	paths []string
}

// NewImportScanner creates a new ImportScanner with the given paths.
func NewImportScanner(paths []string) *ImportScanner {
	return &ImportScanner{
		paths: paths,
	}
}

// ScanFiles scans all configured directories recursively for markdown files.
func (s *ImportScanner) ScanFiles() ([]ImportFile, error) {
	var files []ImportFile

	for _, basePath := range s.paths {
		// Expand home directory if needed
		if strings.HasPrefix(basePath, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				continue
			}
			basePath = filepath.Join(home, basePath[2:])
		}

		// Check if directory exists
		info, err := os.Stat(basePath)
		if err != nil || !info.IsDir() {
			continue
		}

		// Walk the directory
		err = filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip files with errors
			}

			// Skip directories and non-markdown files
			if info.IsDir() || !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
				return nil
			}

			// Parse the file
			bf, err := s.parseFile(path, info)
			if err != nil {
				return nil // Skip files that fail to parse
			}

			files = append(files, *bf)
			return nil
		})
		if err != nil {
			continue
		}
	}

	return files, nil
}

// parseFile reads a markdown file and extracts its metadata.
func (s *ImportScanner) parseFile(path string, info os.FileInfo) (*ImportFile, error) {
	// If info is nil, get it from the file
	if info == nil {
		var err error
		info, err = os.Stat(path)
		if err != nil {
			return nil, err
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Calculate file hash
	hash, err := calculateFileHash(f)
	if err != nil {
		return nil, err
	}

	// Reset file pointer
	_, err = f.Seek(0, 0)
	if err != nil {
		return nil, err
	}

	// Read file content
	content, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	// Parse frontmatter and body
	frontmatter, body := parseFrontmatter(string(content))

	// Extract title from frontmatter or first H1
	title := ""
	if fm, ok := frontmatter["title"]; ok {
		title = fm
	} else {
		title = extractFirstH1(body)
	}

	// If no title found, use filename
	if title == "" {
		title = strings.TrimSuffix(info.Name(), ".md")
	}

	return &ImportFile{
		Path:        path,
		Name:        info.Name(),
		Mtime:       info.ModTime(),
		Hash:        hash,
		Title:       title,
		Body:        body,
		Frontmatter: frontmatter,
	}, nil
}

// calculateFileHash computes SHA256 hash of file contents.
func calculateFileHash(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// parseFrontmatter extracts YAML frontmatter from markdown content.
// Returns the frontmatter as a map and the remaining body.
func parseFrontmatter(content string) (map[string]string, string) {
	frontmatter := make(map[string]string)

	// Check for frontmatter delimiters
	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		return frontmatter, content
	}

	// Find the closing delimiter
	scanner := bufio.NewScanner(strings.NewReader(content))
	var fmLines []string
	inFrontmatter := false
	lineNum := 0
	bodyStart := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		if lineNum == 1 && strings.TrimSpace(line) == "---" {
			inFrontmatter = true
			bodyStart = len(line) + 1 // Account for newline
			continue
		}

		if inFrontmatter {
			if strings.TrimSpace(line) == "---" {
				inFrontmatter = false
				bodyStart += len(line) + 1
				break
			}
			fmLines = append(fmLines, line)
			bodyStart += len(line) + 1
		}
	}

	// If we didn't find closing delimiter, return original content
	if inFrontmatter {
		return make(map[string]string), content
	}

	// Parse YAML frontmatter
	if len(fmLines) > 0 {
		fmContent := strings.Join(fmLines, "\n")
		var parsed map[string]interface{}
		if err := yaml.Unmarshal([]byte(fmContent), &parsed); err == nil {
			for k, v := range parsed {
				switch val := v.(type) {
				case string:
					frontmatter[k] = val
				case bool:
					if val {
						frontmatter[k] = "true"
					} else {
						frontmatter[k] = "false"
					}
				case int:
					frontmatter[k] = fmt.Sprintf("%d", val)
				case int64:
					frontmatter[k] = fmt.Sprintf("%d", val)
				case float64:
					frontmatter[k] = fmt.Sprintf("%v", val)
				case time.Time:
					frontmatter[k] = val.Format(time.RFC3339)
				}
			}
		}
	}

	// Get body (content after frontmatter)
	body := ""
	if bodyStart < len(content) {
		body = strings.TrimLeft(content[bodyStart:], "\n\r")
	}

	return frontmatter, body
}

// extractFirstH1 extracts the first H1 heading from markdown content.
var h1Regex = regexp.MustCompile(`(?m)^#\s+(.+)$`)

var markdownImageRegex = regexp.MustCompile(`!\[.*?\]\((/images/[^)]+)\)`)
var htmlImageRegex = regexp.MustCompile(`<img[^>]+src=["'](/images/[^"']+)["']`)

func ExtractImagePaths(body string) []string {
	pathMap := make(map[string]bool)

	for _, match := range markdownImageRegex.FindAllStringSubmatch(body, -1) {
		if len(match) > 1 {
			path := strings.TrimPrefix(match[1], "/images/")
			pathMap[path] = true
		}
	}

	for _, match := range htmlImageRegex.FindAllStringSubmatch(body, -1) {
		if len(match) > 1 {
			path := strings.TrimPrefix(match[1], "/images/")
			pathMap[path] = true
		}
	}

	var paths []string
	for path := range pathMap {
		paths = append(paths, path)
	}
	return paths
}

func extractFirstH1(content string) string {
	matches := h1Regex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// ComputeImportStatus determines the status of an import based on timestamps.
func ComputeImportStatus(imp *Import, fileMtime time.Time) string {
	// If no content yet, it's pending
	if imp.ContentID == nil {
		return ImportStatusPending
	}

	// File was modified after last import
	if imp.FileMtime != nil && fileMtime.After(*imp.FileMtime) {
		// Check if content was also modified in web UI AFTER the import
		if imp.ContentUpdatedAt != nil && imp.ImportedAt != nil && imp.ContentUpdatedAt.After(*imp.ImportedAt) {
			// Both file and content were modified - conflict
			return ImportStatusConflict
		}
		// Only file was modified - can reimport
		return ImportStatusUpdated
	}

	return ImportStatusSynced
}

// ParseImportFrontmatter parses frontmatter into a ContentFrontmatter struct.
func ParseImportFrontmatter(fm map[string]string) *ContentFrontmatter {
	cf := &ContentFrontmatter{}

	if v, ok := fm["title"]; ok {
		cf.Title = v
	}
	if v, ok := fm["slug"]; ok {
		cf.Slug = v
	}
	if v, ok := fm["short-id"]; ok {
		cf.ShortID = v
	}
	if v, ok := fm["author"]; ok {
		cf.Author = v
	}
	if v, ok := fm["contributor"]; ok {
		cf.Contributor = v
	}
	if v, ok := fm["layout"]; ok {
		cf.Layout = v
	}
	if v, ok := fm["draft"]; ok {
		cf.Draft = v == "true"
	}
	if v, ok := fm["featured"]; ok {
		cf.Featured = v == "true"
	}
	if v, ok := fm["summary"]; ok {
		cf.Summary = v
	}
	if v, ok := fm["description"]; ok {
		cf.Description = v
	}
	if v, ok := fm["image"]; ok {
		cf.Image = v
	}
	if v, ok := fm["social-image"]; ok {
		cf.SocialImage = v
	}
	if v, ok := fm["robots"]; ok {
		cf.Robots = v
	}
	if v, ok := fm["keywords"]; ok {
		cf.Keywords = v
	}
	if v, ok := fm["canonical-url"]; ok {
		cf.CanonicalURL = v
	}
	if v, ok := fm["sitemap"]; ok {
		cf.Sitemap = v
	}
	if v, ok := fm["table-of-contents"]; ok {
		cf.TableOfContents = v == "true"
	}
	if v, ok := fm["comments"]; ok {
		cf.Comments = v == "true"
	}
	if v, ok := fm["share"]; ok {
		cf.Share = v == "true"
	}
	if v, ok := fm["kind"]; ok {
		cf.Kind = v
	}
	if v, ok := fm["series"]; ok {
		cf.Series = v
	}

	return cf
}

// DefaultImportBasePath is the default base path for import directories.
const DefaultImportBasePath = "~/Documents"

// GetImportPath returns the import path for a site.
// Structure: {basePath}/Clio/{siteSlug}/
func GetImportPath(basePath, siteSlug string) string {
	if basePath == "" {
		basePath = DefaultImportBasePath
	}
	return filepath.Join(basePath, "Clio", siteSlug)
}

// ImportFrontmatter represents typed frontmatter that preserves arrays.
type ImportFrontmatter struct {
	Title           string     `yaml:"title"`
	Slug            string     `yaml:"slug"`
	ShortID         string     `yaml:"short-id"`
	Section         string     `yaml:"section"`
	Author          string     `yaml:"author"`
	Contributor     string     `yaml:"contributor"`
	Tags            []string   `yaml:"tags"`
	Layout          string     `yaml:"layout"`
	Draft           bool       `yaml:"draft"`
	Featured        bool       `yaml:"featured"`
	Summary         string     `yaml:"summary"`
	Description     string     `yaml:"description"`
	Image           string     `yaml:"image"`
	SocialImage     string     `yaml:"social-image"`
	PublishedAt     *time.Time `yaml:"published-at"`
	CreatedAt       *time.Time `yaml:"created-at"`
	UpdatedAt       *time.Time `yaml:"updated-at"`
	Robots          string     `yaml:"robots"`
	Keywords        string     `yaml:"keywords"`
	CanonicalURL    string     `yaml:"canonical-url"`
	Sitemap         string     `yaml:"sitemap"`
	TableOfContents bool       `yaml:"table-of-contents"`
	Comments        bool       `yaml:"comments"`
	Share           bool       `yaml:"share"`
	Kind            string     `yaml:"kind"`
	Series          string     `yaml:"series"`
	SeriesOrder     int        `yaml:"series-order"`
}

// ParseTypedFrontmatter extracts typed YAML frontmatter from markdown content.
func ParseTypedFrontmatter(content string) (*ImportFrontmatter, string, error) {
	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		return nil, content, nil
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	var fmLines []string
	inFrontmatter := false
	lineNum := 0
	bodyStart := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		if lineNum == 1 && strings.TrimSpace(line) == "---" {
			inFrontmatter = true
			bodyStart = len(line) + 1
			continue
		}

		if inFrontmatter {
			if strings.TrimSpace(line) == "---" {
				inFrontmatter = false
				bodyStart += len(line) + 1
				break
			}
			fmLines = append(fmLines, line)
			bodyStart += len(line) + 1
		}
	}

	if inFrontmatter {
		return nil, content, nil
	}

	fm := &ImportFrontmatter{}
	if len(fmLines) > 0 {
		fmContent := strings.Join(fmLines, "\n")
		if err := yaml.Unmarshal([]byte(fmContent), fm); err != nil {
			return nil, content, err
		}
	}

	body := ""
	if bodyStart < len(content) {
		body = strings.TrimLeft(content[bodyStart:], "\n\r")
	}

	return fm, body, nil
}

// ImportType represents the type of import based on available metadata.
type ImportType string

const (
	ImportTypeRich  ImportType = "rich"
	ImportTypeBasic ImportType = "basic"
	ImportTypePoor  ImportType = "poor"
)

// DetectImportType determines the import type based on directory structure.
func DetectImportType(importPath string) ImportType {
	metaPath := filepath.Join(importPath, "meta")
	if info, err := os.Stat(metaPath); err == nil && info.IsDir() {
		return ImportTypeRich
	}

	contentPath := filepath.Join(importPath, "content")
	if info, err := os.Stat(contentPath); err == nil && info.IsDir() {
		return ImportTypeRich
	}

	return ImportTypePoor
}

// DetectImportTypeFromFiles determines import type by checking frontmatter in files.
func DetectImportTypeFromFiles(files []ImportFile) ImportType {
	for _, file := range files {
		if section, ok := file.Frontmatter["section"]; ok && section != "" {
			return ImportTypeBasic
		}
	}
	return ImportTypePoor
}

// ImportPresumptions provides defaults for poor imports.
type ImportPresumptions struct {
	DefaultSectionID uuid.UUID
	DirectoryMapping map[string]uuid.UUID
}

// NewImportPresumptions creates default presumptions.
func NewImportPresumptions() *ImportPresumptions {
	return &ImportPresumptions{
		DirectoryMapping: make(map[string]uuid.UUID),
	}
}
