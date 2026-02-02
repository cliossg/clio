package ssg

// MetaContributor represents contributor metadata for YAML export.
type MetaContributor struct {
	Handle      string            `yaml:"handle"`
	Name        string            `yaml:"name"`
	Surname     string            `yaml:"surname,omitempty"`
	Bio         string            `yaml:"bio,omitempty"`
	PhotoPath   string            `yaml:"photo_path,omitempty"`
	SocialLinks map[string]string `yaml:"social_links,omitempty"`
}

// MetaTag represents tag metadata for YAML export.
type MetaTag struct {
	Name string `yaml:"name"`
	Slug string `yaml:"slug"`
}

// MetaSection represents section metadata for YAML export.
type MetaSection struct {
	Name   string `yaml:"name"`
	Path   string `yaml:"path"`
	Layout string `yaml:"layout,omitempty"`
}

// MetaLayout represents layout metadata for YAML export.
type MetaLayout struct {
	Name              string `yaml:"name"`
	Description       string `yaml:"description,omitempty"`
	ExcludeDefaultCSS bool   `yaml:"exclude_default_css,omitempty"`
}

// MetaImage represents image metadata for YAML export.
type MetaImage struct {
	Path           string `yaml:"path"`
	Alt            string `yaml:"alt,omitempty"`
	Caption        string `yaml:"caption,omitempty"`
	Attribution    string `yaml:"attribution,omitempty"`
	AttributionURL string `yaml:"attribution_url,omitempty"`
}

// MetaContentImage represents a content-image association for YAML export.
type MetaContentImage struct {
	ImagePath  string `yaml:"image_path"`
	IsHeader   bool   `yaml:"is_header,omitempty"`
	IsFeatured bool   `yaml:"is_featured,omitempty"`
	OrderNum   int    `yaml:"order_num,omitempty"`
}

// BackupMeta represents the complete metadata for a backup.
type BackupMeta struct {
	Layouts       []MetaLayout                  `yaml:"layouts,omitempty"`
	Contributors  []MetaContributor             `yaml:"contributors,omitempty"`
	Tags          []MetaTag                     `yaml:"tags,omitempty"`
	Sections      []MetaSection                 `yaml:"sections,omitempty"`
	Images        map[string]*MetaImage         `yaml:"images,omitempty"`         // keyed by path
	ContentImages map[string][]MetaContentImage `yaml:"content_images,omitempty"` // keyed by content short_id
	Errors        []string                      `yaml:"-"`                        // parse errors, not serialized
}

// ContributorToMeta converts a Contributor to MetaContributor.
func ContributorToMeta(c *Contributor) MetaContributor {
	socialLinks := make(map[string]string)
	for _, link := range c.SocialLinks {
		if link.URL != "" {
			socialLinks[link.Platform] = link.URL
		} else if link.Handle != "" {
			socialLinks[link.Platform] = link.Handle
		}
	}

	return MetaContributor{
		Handle:      c.Handle,
		Name:        c.Name,
		Surname:     c.Surname,
		Bio:         c.Bio,
		SocialLinks: socialLinks,
	}
}

// TagToMeta converts a Tag to MetaTag.
func TagToMeta(t *Tag) MetaTag {
	return MetaTag{
		Name: t.Name,
		Slug: t.Slug,
	}
}

// SectionToMeta converts a Section to MetaSection.
func SectionToMeta(s *Section) MetaSection {
	return MetaSection{
		Name:   s.Name,
		Path:   s.Path,
		Layout: s.LayoutName,
	}
}

// ImageToMeta converts an Image to MetaImage.
func ImageToMeta(i *Image) *MetaImage {
	return &MetaImage{
		Path:           i.FilePath,
		Alt:            i.AltText,
		Caption:        i.Title,
		Attribution:    i.Attribution,
		AttributionURL: i.AttributionURL,
	}
}

// LayoutToMeta converts a Layout to MetaLayout.
func LayoutToMeta(l *Layout) MetaLayout {
	return MetaLayout{
		Name:              l.Name,
		Description:       l.Description,
		ExcludeDefaultCSS: l.ExcludeDefaultCSS,
	}
}
