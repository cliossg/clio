package ssg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

type ImageMeta struct {
	Title          string `json:"title"`
	Alt            string `json:"alt"`
	Attribution    string `json:"attribution"`
	AttributionURL string `json:"attribution_url"`
}

// Processor handles markdown to HTML conversion.
type Processor struct {
	parser goldmark.Markdown
}

// NewProcessor creates a new markdown processor with GFM extensions.
func NewProcessor() *Processor {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM, // GitHub Flavored Markdown
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			html.WithUnsafe(), // Allow raw HTML in markdown
		),
	)

	return &Processor{
		parser: md,
	}
}

// ToHTML converts markdown bytes to HTML string.
func (p *Processor) ToHTML(markdown []byte) (string, error) {
	var buf bytes.Buffer
	if err := p.parser.Convert(markdown, &buf); err != nil {
		return "", fmt.Errorf("markdown conversion failed: %w", err)
	}
	return buf.String(), nil
}

// ToHTMLString converts markdown string to HTML string.
func (p *Processor) ToHTMLString(markdown string) (string, error) {
	return p.ToHTML([]byte(markdown))
}

// ProcessContent processes a Content's body and returns HTML.
func (p *Processor) ProcessContent(content *Content) (string, error) {
	html, err := p.ToHTML([]byte(content.Body))
	if err != nil {
		return "", err
	}

	// Transform workspace image paths to static site paths FIRST
	html = p.transformImagePaths(html)

	// Parse images metadata if available
	var imagesMeta map[string]ImageMeta
	if content.ImagesMeta != "" {
		json.Unmarshal([]byte(content.ImagesMeta), &imagesMeta)
	}

	// Post-process images with captions (using |||long description syntax)
	html = p.enhanceImages(html, imagesMeta)

	// Process embed code blocks
	html = processEmbeds(html)

	return html, nil
}

// transformImagePaths converts workspace paths to static site paths.
// /ssg/workspace/{slug}/images/{file} -> /images/{file}
func (p *Processor) transformImagePaths(html string) string {
	re := regexp.MustCompile(`/ssg/workspace/[^/]+/images/`)
	return re.ReplaceAllString(html, "/images/")
}

// enhanceImages post-processes HTML to enhance images with captions and credits.
// Supports syntax: ![alt text|||caption](image.jpg)
// Also adds attribution credits from imagesMeta if available.
func (p *Processor) enhanceImages(html string, imagesMeta map[string]ImageMeta) string {
	imgRegex := regexp.MustCompile(`<img([^>]*?)alt="([^"]*?)"([^>]*?)>`)

	result := imgRegex.ReplaceAllStringFunc(html, func(match string) string {
		srcRegex := regexp.MustCompile(`src="([^"]*)"`)
		altRegex := regexp.MustCompile(`alt="([^"]*)"`)

		srcMatch := srcRegex.FindStringSubmatch(match)
		altMatch := altRegex.FindStringSubmatch(match)

		if len(srcMatch) < 2 || len(altMatch) < 2 {
			return match
		}

		srcValue := srcMatch[1]
		altValue := altMatch[1]

		var altText, caption string
		if strings.Contains(altValue, "|||") {
			parts := strings.SplitN(altValue, "|||", 2)
			altText = strings.TrimSpace(parts[0])
			caption = strings.TrimSpace(parts[1])
		} else {
			altText = altValue
		}

		enhancedImg := fmt.Sprintf(`<img src="%s" alt="%s" class="content-img" loading="lazy">`, srcValue, altText)

		// Check for image metadata (attribution)
		var credit string
		if imagesMeta != nil {
			if meta, ok := imagesMeta[srcValue]; ok && meta.Attribution != "" {
				if meta.AttributionURL != "" {
					credit = fmt.Sprintf(`<figcaption class="content-credit"><span class="content-credit-title">%s</span><span class="content-credit-attr"><a href="%s" target="_blank" rel="noopener">%s</a></span></figcaption>`,
						meta.Title, meta.AttributionURL, meta.Attribution)
				} else {
					credit = fmt.Sprintf(`<figcaption class="content-credit"><span class="content-credit-title">%s</span><span class="content-credit-attr">%s</span></figcaption>`,
						meta.Title, meta.Attribution)
				}
			}
		}

		if caption != "" || credit != "" {
			var figContent string
			if caption != "" {
				figContent = fmt.Sprintf(`<figcaption class="content-caption">%s</figcaption>`, caption)
			}
			return fmt.Sprintf(`<figure class="content-figure">%s%s%s</figure>`, enhancedImg, credit, figContent)
		}

		return enhancedImg
	})

	return result
}

// ExtractFirstParagraph extracts the first paragraph from markdown for use as excerpt.
func (p *Processor) ExtractFirstParagraph(markdown string) string {
	// Split by double newlines to find paragraphs
	paragraphs := strings.Split(markdown, "\n\n")
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		// Skip empty lines, headers, code blocks, lists
		if para == "" {
			continue
		}
		if strings.HasPrefix(para, "#") {
			continue
		}
		if strings.HasPrefix(para, "```") {
			continue
		}
		if strings.HasPrefix(para, "-") || strings.HasPrefix(para, "*") || strings.HasPrefix(para, "1.") {
			continue
		}
		if strings.HasPrefix(para, ">") {
			continue
		}
		if strings.HasPrefix(para, "![") {
			continue
		}

		// Found a regular paragraph
		return para
	}
	return ""
}

// ExtractHeadings extracts all headings from markdown for TOC generation.
func (p *Processor) ExtractHeadings(markdown string) []Heading {
	var headings []Heading
	lines := strings.Split(markdown, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#") {
			continue
		}

		// Count # characters for level
		level := 0
		for _, ch := range line {
			if ch == '#' {
				level++
			} else {
				break
			}
		}

		if level > 0 && level <= 6 {
			text := strings.TrimSpace(strings.TrimLeft(line, "# "))
			if text != "" {
				headings = append(headings, Heading{
					Level: level,
					Text:  text,
					ID:    Slugify(text),
				})
			}
		}
	}

	return headings
}

// Heading represents a heading extracted from markdown.
type Heading struct {
	Level int
	Text  string
	ID    string
}
