package ssg

import (
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

// Site represents a site in the multi-site system.
type Site struct {
	ID                uuid.UUID `json:"id"`
	ShortID           string    `json:"short_id"`
	Name              string    `json:"name"`
	Slug              string    `json:"slug"`
	Mode              string    `json:"mode"` // "blog" or "structured"
	Active            bool      `json:"active"`
	DefaultLayoutID   uuid.UUID `json:"default_layout_id"`
	DefaultLayoutName string    `json:"default_layout_name"`
	CreatedBy         uuid.UUID `json:"-"`
	UpdatedBy         uuid.UUID `json:"-"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// NewSite creates a new Site instance.
func NewSite(name, slug, mode string) *Site {
	now := time.Now()
	return &Site{
		ID:        uuid.New(),
		ShortID:   uuid.New().String()[:8],
		Name:      name,
		Slug:      slug,
		Mode:      mode,
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Section represents a content section (e.g., /blog, /docs).
type Section struct {
	ID          uuid.UUID `json:"id"`
	SiteID      uuid.UUID `json:"site_id"`
	ShortID     string    `json:"short_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Path        string    `json:"path"`
	LayoutID    uuid.UUID `json:"layout_id"`
	LayoutName  string    `json:"layout_name"`
	CreatedBy   uuid.UUID `json:"-"`
	UpdatedBy   uuid.UUID `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func normalizePath(path string) string {
	return strings.TrimLeft(path, "/")
}

// NewSection creates a new Section instance.
func NewSection(siteID uuid.UUID, name, description, path string) *Section {
	now := time.Now()
	return &Section{
		ID:          uuid.New(),
		SiteID:      siteID,
		ShortID:     uuid.New().String()[:8],
		Name:        name,
		Description: description,
		Path:        normalizePath(path),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Content represents a content item (article, page, etc.).
type Content struct {
	ID            uuid.UUID  `json:"id"`
	SiteID        uuid.UUID  `json:"site_id"`
	UserID        uuid.UUID  `json:"user_id"`
	ShortID       string     `json:"short_id"`
	SectionID     uuid.UUID  `json:"section_id"`
	ContributorID     *uuid.UUID `json:"contributor_id,omitempty"`
	ContributorHandle string     `json:"contributor_handle,omitempty"`
	AuthorUsername    string     `json:"author_username,omitempty"`
	Kind              string     `json:"kind"` // "post", "page", "series"
	Heading       string     `json:"heading"`
	Summary       string     `json:"summary"`
	Body          string     `json:"body"`
	Draft         bool       `json:"draft"`
	Featured      bool       `json:"featured"`
	Series        string     `json:"series,omitempty"`
	SeriesOrder   int        `json:"series_order,omitempty"`
	PublishedAt   *time.Time `json:"published_at"`

	// Joined fields
	SectionPath string       `json:"section_path,omitempty"`
	SectionName string       `json:"section_name,omitempty"`
	Tags        []*Tag       `json:"tags,omitempty"`
	Meta        *Meta        `json:"meta,omitempty"`
	Contributor *Contributor `json:"contributor,omitempty"`

	// Image fields (from relationships)
	HeaderImageURL     string `json:"header_image_url,omitempty"`
	HeaderImageAlt     string `json:"header_image_alt,omitempty"`
	HeaderImageCaption string `json:"header_image_caption,omitempty"`

	CreatedBy uuid.UUID `json:"-"`
	UpdatedBy uuid.UUID `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewContent creates a new Content instance.
func NewContent(siteID, sectionID uuid.UUID, heading, body string) *Content {
	now := time.Now()
	return &Content{
		ID:        uuid.New(),
		SiteID:    siteID,
		SectionID: sectionID,
		ShortID:   uuid.New().String()[:8],
		Heading:   heading,
		Body:      body,
		Draft:     true,
		Kind:      "post",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Slug returns a URL-friendly slug for the content.
func (c *Content) Slug() string {
	return Slugify(c.Heading) + "-" + c.ShortID
}

// DisplayHandle returns the handle to display (contributor takes precedence).
func (c *Content) DisplayHandle() string {
	if c.ContributorHandle != "" {
		return c.ContributorHandle
	}
	return c.AuthorUsername
}

// Layout represents a content layout template.
type Layout struct {
	ID            uuid.UUID `json:"id"`
	SiteID        uuid.UUID `json:"site_id"`
	ShortID       string    `json:"short_id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Code          string    `json:"code"`
	HeaderImageID uuid.UUID `json:"header_image_id"`
	CreatedBy     uuid.UUID `json:"-"`
	UpdatedBy     uuid.UUID `json:"-"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// NewLayout creates a new Layout instance.
func NewLayout(siteID uuid.UUID, name, description string) *Layout {
	now := time.Now()
	return &Layout{
		ID:        uuid.New(),
		SiteID:    siteID,
		ShortID:   uuid.New().String()[:8],
		Name:      name,
		Description: description,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Tag represents a content tag.
type Tag struct {
	ID        uuid.UUID `json:"id"`
	SiteID    uuid.UUID `json:"site_id"`
	ShortID   string    `json:"short_id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedBy uuid.UUID `json:"-"`
	UpdatedBy uuid.UUID `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewTag creates a new Tag instance.
func NewTag(siteID uuid.UUID, name string) *Tag {
	now := time.Now()
	return &Tag{
		ID:        uuid.New(),
		SiteID:    siteID,
		ShortID:   uuid.New().String()[:8],
		Name:      name,
		Slug:      Slugify(name),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Meta represents SEO metadata for content.
type Meta struct {
	ID              uuid.UUID `json:"id"`
	SiteID          uuid.UUID `json:"site_id"`
	ShortID         string    `json:"short_id"`
	ContentID       uuid.UUID `json:"content_id"`
	Summary         string    `json:"summary"`
	Excerpt         string    `json:"excerpt"`
	Description     string    `json:"description"`
	Keywords        string    `json:"keywords"`
	Robots          string    `json:"robots"`
	CanonicalURL    string    `json:"canonical_url"`
	Sitemap         string    `json:"sitemap"`
	TableOfContents bool      `json:"table_of_contents"`
	Share           bool      `json:"share"`
	Comments        bool      `json:"comments"`
	CreatedBy       uuid.UUID `json:"-"`
	UpdatedBy       uuid.UUID `json:"-"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// NewMeta creates a new Meta instance.
func NewMeta(siteID, contentID uuid.UUID) *Meta {
	now := time.Now()
	return &Meta{
		ID:        uuid.New(),
		SiteID:    siteID,
		ShortID:   uuid.New().String()[:8],
		ContentID: contentID,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Param represents a site configuration parameter.
type Param struct {
	ID          uuid.UUID `json:"id"`
	SiteID      uuid.UUID `json:"site_id"`
	ShortID     string    `json:"short_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Value       string    `json:"value"`
	RefKey      string    `json:"ref_key"`
	Category    string    `json:"category"`
	Position    int       `json:"position"`
	System      bool      `json:"system"`
	CreatedBy   uuid.UUID `json:"-"`
	UpdatedBy   uuid.UUID `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewParam creates a new Param instance.
func NewParam(siteID uuid.UUID, name, value string) *Param {
	now := time.Now()
	return &Param{
		ID:        uuid.New(),
		SiteID:    siteID,
		ShortID:   uuid.New().String()[:8],
		Name:      name,
		Value:     value,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (p *Param) MaskedValue() string {
	if p.Value == "" {
		return ""
	}

	lower := strings.ToLower(p.Name) + strings.ToLower(p.RefKey)
	sensitive := strings.Contains(lower, "token") ||
		strings.Contains(lower, "pass") ||
		strings.Contains(lower, "secret") ||
		strings.Contains(lower, "key") ||
		strings.Contains(lower, "credential")

	if !sensitive {
		if len(p.Value) > 50 {
			return p.Value[:50] + "..."
		}
		return p.Value
	}

	if len(p.Value) <= 8 {
		return "***"
	}
	return p.Value[:4] + "***..." + p.Value[len(p.Value)-4:]
}

// Image represents an image asset.
type Image struct {
	ID        uuid.UUID `json:"id"`
	SiteID    uuid.UUID `json:"site_id"`
	ShortID   string    `json:"short_id"`
	FileName  string    `json:"file_name"`
	FilePath  string    `json:"file_path"`
	AltText   string    `json:"alt_text"`
	Title     string    `json:"title"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	CreatedBy uuid.UUID `json:"-"`
	UpdatedBy uuid.UUID `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewImage creates a new Image instance.
func NewImage(siteID uuid.UUID, fileName, filePath string) *Image {
	now := time.Now()
	return &Image{
		ID:        uuid.New(),
		SiteID:    siteID,
		ShortID:   uuid.New().String()[:8],
		FileName:  fileName,
		FilePath:  filePath,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// ImageVariant represents a variant of an image (thumbnail, etc.).
type ImageVariant struct {
	ID            uuid.UUID `json:"id"`
	ShortID       string    `json:"short_id"`
	ImageID       uuid.UUID `json:"image_id"`
	Kind          string    `json:"kind"`
	BlobRef       string    `json:"blob_ref"`
	Width         int       `json:"width"`
	Height        int       `json:"height"`
	FilesizeBytes int       `json:"filesize_bytes"`
	Mime          string    `json:"mime"`
	CreatedBy     uuid.UUID `json:"-"`
	UpdatedBy     uuid.UUID `json:"-"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ContentImage represents a content-image relationship.
type ContentImage struct {
	ID         uuid.UUID `json:"id"`
	ContentID  uuid.UUID `json:"content_id"`
	ImageID    uuid.UUID `json:"image_id"`
	IsHeader   bool      `json:"is_header"`
	IsFeatured bool      `json:"is_featured"`
	OrderNum   int       `json:"order_num"`
	CreatedAt  time.Time `json:"created_at"`
}

// ContentImageWithDetails represents a content-image with full image data.
type ContentImageWithDetails struct {
	ContentImageID uuid.UUID `json:"content_image_id"`
	ContentID      uuid.UUID `json:"content_id"`
	IsHeader       bool      `json:"is_header"`
	IsFeatured     bool      `json:"is_featured"`
	OrderNum       int       `json:"order_num"`
	// Image fields
	ID        uuid.UUID `json:"id"`
	SiteID    uuid.UUID `json:"site_id"`
	ShortID   string    `json:"short_id"`
	FileName  string    `json:"file_name"`
	FilePath  string    `json:"file_path"`
	AltText   string    `json:"alt_text"`
	Title     string    `json:"title"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ContentImageDetails represents minimal info for deletion.
type ContentImageDetails struct {
	ContentImageID uuid.UUID `json:"content_image_id"`
	ImageID        uuid.UUID `json:"image_id"`
	FilePath       string    `json:"file_path"`
}

// SectionImage represents a section-image relationship.
type SectionImage struct {
	ID         uuid.UUID `json:"id"`
	SectionID  uuid.UUID `json:"section_id"`
	ImageID    uuid.UUID `json:"image_id"`
	IsHeader   bool      `json:"is_header"`
	IsFeatured bool      `json:"is_featured"`
	OrderNum   int       `json:"order_num"`
	CreatedAt  time.Time `json:"created_at"`
}

// SectionImageWithDetails represents a section-image with full image data.
type SectionImageWithDetails struct {
	SectionImageID uuid.UUID `json:"section_image_id"`
	SectionID      uuid.UUID `json:"section_id"`
	IsHeader       bool      `json:"is_header"`
	IsFeatured     bool      `json:"is_featured"`
	OrderNum       int       `json:"order_num"`
	ID             uuid.UUID `json:"id"`
	SiteID         uuid.UUID `json:"site_id"`
	ShortID        string    `json:"short_id"`
	FileName       string    `json:"file_name"`
	FilePath       string    `json:"file_path"`
	AltText        string    `json:"alt_text"`
	Title          string    `json:"title"`
	Width          int       `json:"width"`
	Height         int       `json:"height"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// SectionImageDetails represents minimal info for deletion.
type SectionImageDetails struct {
	SectionImageID uuid.UUID `json:"section_image_id"`
	ImageID        uuid.UUID `json:"image_id"`
	FilePath       string    `json:"file_path"`
}

// --- Contributor ---

type SocialLink struct {
	Platform string `json:"platform"`
	Handle   string `json:"handle,omitempty"`
	URL      string `json:"url,omitempty"`
}

type Contributor struct {
	ID          uuid.UUID    `json:"id"`
	SiteID      uuid.UUID    `json:"site_id"`
	ProfileID   *uuid.UUID   `json:"profile_id,omitempty"`
	ShortID     string       `json:"short_id"`
	Handle      string       `json:"handle"`
	Name        string       `json:"name"`
	Surname     string       `json:"surname"`
	Bio         string       `json:"bio"`
	SocialLinks []SocialLink `json:"social_links"`
	Role        string       `json:"role"`
	PhotoPath   string       `json:"photo_path,omitempty"`
	CreatedBy   uuid.UUID    `json:"-"`
	UpdatedBy   uuid.UUID    `json:"-"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

const ContributorRoleEditor = "editor"

func NewContributor(siteID uuid.UUID, handle, name, surname string) *Contributor {
	now := time.Now()
	return &Contributor{
		ID:          uuid.New(),
		SiteID:      siteID,
		ShortID:     uuid.New().String()[:8],
		Handle:      handle,
		Name:        name,
		Surname:     surname,
		SocialLinks: []SocialLink{},
		Role:        ContributorRoleEditor,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func (c *Contributor) FullName() string {
	if c.Surname == "" {
		return c.Name
	}
	return c.Name + " " + c.Surname
}

// --- Utility Functions ---

var nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify converts a string to a URL-friendly slug.
func Slugify(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace non-alphanumeric characters with hyphens
	s = nonAlphanumericRegex.ReplaceAllString(s, "-")

	// Trim leading/trailing hyphens
	s = strings.Trim(s, "-")

	return s
}

// Normalize removes diacritics and normalizes a string.
func Normalize(s string) string {
	// Simple normalization: lowercase and remove non-ASCII
	var result strings.Builder
	for _, r := range strings.ToLower(s) {
		if r < 128 && (unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '-') {
			result.WriteRune(r)
		}
	}
	return Slugify(result.String())
}
