package ssg

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// MetaGenerator handles generation of meta YAML files for backup.
type MetaGenerator struct {
	workspace *Workspace
}

// NewMetaGenerator creates a new meta generator.
func NewMetaGenerator(workspace *Workspace) *MetaGenerator {
	return &MetaGenerator{
		workspace: workspace,
	}
}

// GenerateMetaResult contains the result of meta generation.
type GenerateMetaResult struct {
	LayoutsFile        string
	ContributorsFile   string
	TagsFile           string
	SectionsFile       string
	ImagesFile         string
	ContentImagesFile  string
	Errors             []string
}

// GenerateMeta generates all meta YAML files for a site backup.
func (g *MetaGenerator) GenerateMeta(
	siteSlug string,
	layouts []*Layout,
	contributors []*Contributor,
	tags []*Tag,
	sections []*Section,
	images []*Image,
	contentImages map[string][]MetaContentImage,
	contributorPhotoPaths map[string]string,
) (*GenerateMetaResult, error) {
	result := &GenerateMetaResult{}

	basePath := g.workspace.GetMetaPath(siteSlug)

	if err := CleanDir(basePath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to clean meta directory: %w", err)
	}

	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create meta directory: %w", err)
	}

	if len(layouts) > 0 {
		metaLayouts := make([]MetaLayout, 0, len(layouts))
		for _, l := range layouts {
			metaLayouts = append(metaLayouts, LayoutToMeta(l))
		}

		filePath := filepath.Join(basePath, "layouts.yml")
		if err := g.writeYAMLFile(filePath, metaLayouts); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("layouts: %v", err))
		} else {
			result.LayoutsFile = filePath
		}

		layoutsDir := filepath.Join(basePath, "layouts")
		if err := os.MkdirAll(layoutsDir, 0755); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("layouts dir: %v", err))
		} else {
			for _, l := range layouts {
				if l.Code != "" {
					codePath := filepath.Join(layoutsDir, l.Name+".html")
					if err := os.WriteFile(codePath, []byte(l.Code), 0644); err != nil {
						result.Errors = append(result.Errors, fmt.Sprintf("layout %s code: %v", l.Name, err))
					}
				}
				if l.CSS != "" {
					cssPath := filepath.Join(layoutsDir, l.Name+".css")
					if err := os.WriteFile(cssPath, []byte(l.CSS), 0644); err != nil {
						result.Errors = append(result.Errors, fmt.Sprintf("layout %s css: %v", l.Name, err))
					}
				}
			}
		}
	}

	// Generate contributors.yml
	if len(contributors) > 0 {
		metaContributors := make([]MetaContributor, 0, len(contributors))
		for _, c := range contributors {
			mc := ContributorToMeta(c)
			if photoPath, ok := contributorPhotoPaths[c.Handle]; ok {
				mc.PhotoPath = photoPath
			}
			metaContributors = append(metaContributors, mc)
		}

		filePath := filepath.Join(basePath, "contributors.yml")
		if err := g.writeYAMLFile(filePath, metaContributors); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("contributors: %v", err))
		} else {
			result.ContributorsFile = filePath
		}
	}

	// Generate tags.yml
	if len(tags) > 0 {
		metaTags := make([]MetaTag, 0, len(tags))
		for _, t := range tags {
			metaTags = append(metaTags, TagToMeta(t))
		}

		filePath := filepath.Join(basePath, "tags.yml")
		if err := g.writeYAMLFile(filePath, metaTags); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("tags: %v", err))
		} else {
			result.TagsFile = filePath
		}
	}

	// Generate sections.yml
	if len(sections) > 0 {
		metaSections := make([]MetaSection, 0, len(sections))
		for _, s := range sections {
			metaSections = append(metaSections, SectionToMeta(s))
		}

		filePath := filepath.Join(basePath, "sections.yml")
		if err := g.writeYAMLFile(filePath, metaSections); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("sections: %v", err))
		} else {
			result.SectionsFile = filePath
		}
	}

	// Generate images.yml (only images with metadata)
	if len(images) > 0 {
		metaImages := make(map[string]*MetaImage)
		for _, img := range images {
			// Only include images that have metadata
			if img.AltText != "" || img.Title != "" || img.Attribution != "" || img.AttributionURL != "" {
				metaImages[img.FilePath] = ImageToMeta(img)
			}
		}

		if len(metaImages) > 0 {
			filePath := filepath.Join(basePath, "images.yml")
			if err := g.writeYAMLFile(filePath, metaImages); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("images: %v", err))
			} else {
				result.ImagesFile = filePath
			}
		}
	}

	if len(contentImages) > 0 {
		filePath := filepath.Join(basePath, "content_images.yml")
		if err := g.writeYAMLFile(filePath, contentImages); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("content_images: %v", err))
		} else {
			result.ContentImagesFile = filePath
		}
	}

	return result, nil
}

// writeYAMLFile writes data to a YAML file.
func (g *MetaGenerator) writeYAMLFile(filePath string, data interface{}) error {
	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("cannot marshal YAML: %w", err)
	}

	if err := os.WriteFile(filePath, yamlBytes, 0644); err != nil {
		return fmt.Errorf("cannot write file: %w", err)
	}

	return nil
}
