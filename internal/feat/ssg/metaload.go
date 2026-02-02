package ssg

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type MetaLoader struct {
	service        Service
	profileService ProfileService
	workspace      *Workspace
}

func NewMetaLoader(service Service, profileService ProfileService, workspace *Workspace) *MetaLoader {
	return &MetaLoader{service: service, profileService: profileService, workspace: workspace}
}

type HydrationResult struct {
	LayoutsCreated      int
	SectionsCreated     int
	ContributorsCreated int
	ProfilesCreated     int
	TagsCreated         int
	ImagesCreated       int
	ImagesUpdated       int
	Errors              []string
}

func (l *MetaLoader) LoadMeta(metaPath string) (*BackupMeta, error) {
	meta := &BackupMeta{}
	meta.Errors = make([]string, 0)

	layoutsPath := filepath.Join(metaPath, "layouts.yml")
	if data, err := os.ReadFile(layoutsPath); err == nil {
		var layouts []MetaLayout
		if err := yaml.Unmarshal(data, &layouts); err != nil {
			meta.Errors = append(meta.Errors, "layouts.yml parse error: "+err.Error())
		} else {
			meta.Layouts = layouts
		}
	}

	contributorsPath := filepath.Join(metaPath, "contributors.yml")
	if data, err := os.ReadFile(contributorsPath); err == nil {
		var contributors []MetaContributor
		if err := yaml.Unmarshal(data, &contributors); err != nil {
			meta.Errors = append(meta.Errors, "contributors.yml parse error: "+err.Error())
		} else {
			meta.Contributors = contributors
		}
	}

	tagsPath := filepath.Join(metaPath, "tags.yml")
	if data, err := os.ReadFile(tagsPath); err == nil {
		var tags []MetaTag
		if err := yaml.Unmarshal(data, &tags); err == nil {
			meta.Tags = tags
		}
	}

	sectionsPath := filepath.Join(metaPath, "sections.yml")
	if data, err := os.ReadFile(sectionsPath); err == nil {
		var sections []MetaSection
		if err := yaml.Unmarshal(data, &sections); err == nil {
			meta.Sections = sections
		}
	}

	imagesPath := filepath.Join(metaPath, "images.yml")
	if data, err := os.ReadFile(imagesPath); err == nil {
		var images map[string]*MetaImage
		if err := yaml.Unmarshal(data, &images); err == nil {
			meta.Images = images
		}
	}

	contentImagesPath := filepath.Join(metaPath, "content_images.yml")
	if data, err := os.ReadFile(contentImagesPath); err == nil {
		var contentImages map[string][]MetaContentImage
		if err := yaml.Unmarshal(data, &contentImages); err == nil {
			meta.ContentImages = contentImages
		}
	}

	return meta, nil
}

func (l *MetaLoader) HydrateFromMeta(ctx context.Context, siteID uuid.UUID, meta *BackupMeta, userID uuid.UUID) (*HydrationResult, error) {
	return l.HydrateFromMetaWithPath(ctx, siteID, meta, userID, "")
}

func (l *MetaLoader) HydrateFromMetaWithPath(ctx context.Context, siteID uuid.UUID, meta *BackupMeta, userID uuid.UUID, metaPath string) (*HydrationResult, error) {
	result := &HydrationResult{}

	for _, ml := range meta.Layouts {
		existingLayouts, _ := l.service.GetLayouts(ctx, siteID)
		found := false
		for _, existing := range existingLayouts {
			if existing.Name == ml.Name {
				found = true
				break
			}
		}
		if !found {
			layout := NewLayout(siteID, ml.Name, ml.Description)
			layout.ExcludeDefaultCSS = ml.ExcludeDefaultCSS
			layout.CreatedBy = userID
			layout.UpdatedBy = userID

			if metaPath != "" {
				codePath := filepath.Join(metaPath, "layouts", ml.Name+".html")
				if data, err := os.ReadFile(codePath); err == nil {
					layout.Code = string(data)
				}
				cssPath := filepath.Join(metaPath, "layouts", ml.Name+".css")
				if data, err := os.ReadFile(cssPath); err == nil {
					layout.CSS = string(data)
				}
			}

			if err := l.service.CreateLayout(ctx, layout); err != nil {
				result.Errors = append(result.Errors, "layout "+ml.Name+": "+err.Error())
			} else {
				result.LayoutsCreated++
			}
		}
	}

	for _, ms := range meta.Sections {
		_, err := l.service.GetSectionByPath(ctx, siteID, ms.Path)
		if errors.Is(err, ErrNotFound) {
			section := NewSection(siteID, ms.Name, "", ms.Path)
			section.CreatedBy = userID
			section.UpdatedBy = userID

			if ms.Layout != "" {
				layouts, _ := l.service.GetLayouts(ctx, siteID)
				for _, layout := range layouts {
					if layout.Name == ms.Layout {
						section.LayoutID = layout.ID
						break
					}
				}
			}

			if err := l.service.CreateSection(ctx, section); err != nil {
				result.Errors = append(result.Errors, "section "+ms.Name+": "+err.Error())
			} else {
				result.SectionsCreated++
			}
		}
	}

	for _, mc := range meta.Contributors {
		_, err := l.service.GetContributorByHandle(ctx, siteID, mc.Handle)
		if errors.Is(err, ErrNotFound) {
			contributor := NewContributor(siteID, mc.Handle, mc.Name, mc.Surname)
			contributor.Bio = mc.Bio
			contributor.CreatedBy = userID
			contributor.UpdatedBy = userID

			if len(mc.SocialLinks) > 0 {
				for platform, value := range mc.SocialLinks {
					contributor.SocialLinks = append(contributor.SocialLinks, SocialLink{
						Platform: platform,
						URL:      value,
					})
				}
			}

			if err := l.service.CreateContributor(ctx, contributor); err != nil {
				result.Errors = append(result.Errors, "contributor "+mc.Handle+": "+err.Error())
			} else {
				result.ContributorsCreated++
			}
		}
	}

	for _, mt := range meta.Tags {
		_, err := l.service.GetTagByName(ctx, siteID, mt.Name)
		if errors.Is(err, ErrNotFound) {
			tag := NewTag(siteID, mt.Name)
			if mt.Slug != "" {
				tag.Slug = mt.Slug
			}
			tag.CreatedBy = userID
			tag.UpdatedBy = userID

			if err := l.service.CreateTag(ctx, tag); err != nil {
				result.Errors = append(result.Errors, "tag "+mt.Name+": "+err.Error())
			} else {
				result.TagsCreated++
			}
		}
	}

	return result, nil
}

// HydrateImages copies images from backup and creates/updates DB records.
func (l *MetaLoader) HydrateImages(ctx context.Context, siteID uuid.UUID, siteSlug string, importPath string, meta *BackupMeta, userID uuid.UUID) (*ImageHydrationResult, error) {
	result := &ImageHydrationResult{}

	// Get source images directory
	srcImagesPath := filepath.Join(importPath, "images")
	if _, err := os.Stat(srcImagesPath); os.IsNotExist(err) {
		return result, nil // No images to hydrate
	}

	// Get destination images directory
	dstImagesPath := l.workspace.GetImagesPath(siteSlug)
	if err := os.MkdirAll(dstImagesPath, 0755); err != nil {
		return result, err
	}

	// Get existing images from DB
	existingImages, _ := l.service.GetImages(ctx, siteID)
	existingMap := make(map[string]*Image)
	for _, img := range existingImages {
		existingMap[img.FilePath] = img
	}

	// Walk source images directory
	err := filepath.Walk(srcImagesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Get relative path from images dir
		relPath, err := filepath.Rel(srcImagesPath, path)
		if err != nil {
			return nil
		}

		// Skip non-image files
		ext := strings.ToLower(filepath.Ext(relPath))
		if !isImageExtension(ext) {
			return nil
		}

		// Copy file to workspace
		dstPath := filepath.Join(dstImagesPath, relPath)
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			result.Errors = append(result.Errors, "mkdir "+relPath+": "+err.Error())
			return nil
		}

		if err := copyFile(path, dstPath); err != nil {
			result.Errors = append(result.Errors, "copy "+relPath+": "+err.Error())
			return nil
		}
		result.FilesCopied++

		// Check if image exists in DB
		if img, exists := existingMap[relPath]; exists {
			// Update metadata if available
			if mi, ok := meta.Images[relPath]; ok {
				img.AltText = mi.Alt
				img.Title = mi.Caption
				img.Attribution = mi.Attribution
				img.AttributionURL = mi.AttributionURL
				img.UpdatedBy = userID

				if err := l.service.UpdateImage(ctx, img); err != nil {
					result.Errors = append(result.Errors, "update "+relPath+": "+err.Error())
				} else {
					result.ImagesUpdated++
				}
			}
		} else {
			// Create new image record
			fileName := filepath.Base(relPath)
			image := NewImage(siteID, fileName, relPath)
			image.CreatedBy = userID
			image.UpdatedBy = userID

			// Apply metadata if available
			if mi, ok := meta.Images[relPath]; ok {
				image.AltText = mi.Alt
				image.Title = mi.Caption
				image.Attribution = mi.Attribution
				image.AttributionURL = mi.AttributionURL
			}

			if err := l.service.CreateImage(ctx, image); err != nil {
				result.Errors = append(result.Errors, "create "+relPath+": "+err.Error())
			} else {
				result.ImagesCreated++
			}
		}

		return nil
	})

	if err != nil {
		return result, err
	}

	return result, nil
}

// ImageHydrationResult contains the results of image hydration.
type ImageHydrationResult struct {
	FilesCopied   int
	ImagesCreated int
	ImagesUpdated int
	Errors        []string
}

func HasMetaDirectory(importPath string) bool {
	metaPath := filepath.Join(importPath, "meta")
	info, err := os.Stat(metaPath)
	return err == nil && info.IsDir()
}

func GetMetaPath(importPath string) string {
	return filepath.Join(importPath, "meta")
}

func GetContentPath(importPath string) string {
	contentPath := filepath.Join(importPath, "content")
	if info, err := os.Stat(contentPath); err == nil && info.IsDir() {
		return contentPath
	}
	return importPath
}

// GetImagesPath returns the images path from import directory.
func GetImagesPath(importPath string) string {
	return filepath.Join(importPath, "images")
}

// HasImagesDirectory checks if images directory exists.
func HasImagesDirectory(importPath string) bool {
	imagesPath := filepath.Join(importPath, "images")
	info, err := os.Stat(imagesPath)
	return err == nil && info.IsDir()
}

// isImageExtension checks if extension is a supported image format.
func isImageExtension(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".avif":
		return true
	}
	return false
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return dstFile.Sync()
}

func HasProfilesDirectory(importPath string) bool {
	profilesPath := filepath.Join(importPath, "profiles")
	info, err := os.Stat(profilesPath)
	return err == nil && info.IsDir()
}

func CopyProfiles(importPath, workspaceProfilesPath string) (int, error) {
	srcPath := filepath.Join(importPath, "profiles")
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return 0, nil
	}

	if err := os.MkdirAll(workspaceProfilesPath, 0755); err != nil {
		return 0, err
	}

	var count int
	err := filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(srcPath, path)
		if err != nil {
			return nil
		}

		dstPath := filepath.Join(workspaceProfilesPath, relPath)
		if err := copyFile(path, dstPath); err != nil {
			return nil
		}
		count++
		return nil
	})

	return count, err
}

func (l *MetaLoader) HydrateProfiles(ctx context.Context, siteID uuid.UUID, importPath string, meta *BackupMeta, userID uuid.UUID) (*ProfileHydrationResult, error) {
	result := &ProfileHydrationResult{}

	contributors, err := l.service.GetContributors(ctx, siteID)
	if err != nil {
		return result, err
	}

	profilesPath := filepath.Join(importPath, "profiles")

	for _, contributor := range contributors {
		if contributor.ProfileID != nil {
			continue
		}

		var mc *MetaContributor
		for i := range meta.Contributors {
			if meta.Contributors[i].Handle == contributor.Handle {
				mc = &meta.Contributors[i]
				break
			}
		}
		if mc == nil {
			continue
		}

		// Use photo path from metadata, or try to find by handle as fallback
		photoPath := mc.PhotoPath
		if photoPath == "" {
			photoPath = findProfilePhoto(profilesPath, contributor.Handle)
		} else {
			// Verify the file exists at the metadata-specified path
			fullPath := filepath.Join(profilesPath, photoPath)
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				photoPath = "" // File doesn't exist, clear the path
			}
		}

		socialLinks := "[]"
		if len(mc.SocialLinks) > 0 {
			var links []map[string]string
			for platform, url := range mc.SocialLinks {
				links = append(links, map[string]string{
					"platform": platform,
					"url":      url,
				})
			}
			if data, err := json.Marshal(links); err == nil {
				socialLinks = string(data)
			}
		}

		newProfile, err := l.profileService.CreateProfile(
			ctx,
			siteID,
			contributor.Handle,
			contributor.Name,
			contributor.Surname,
			contributor.Bio,
			socialLinks,
			photoPath,
			userID.String(),
		)
		if err != nil {
			result.Errors = append(result.Errors, "profile "+contributor.Handle+": "+err.Error())
			continue
		}

		if err := l.service.SetContributorProfile(ctx, contributor.ID, newProfile.ID, userID.String()); err != nil {
			result.Errors = append(result.Errors, "link profile "+contributor.Handle+": "+err.Error())
			continue
		}

		result.ProfilesCreated++
	}

	return result, nil
}

type ProfileHydrationResult struct {
	ProfilesCreated int
	Errors          []string
}

func findProfilePhoto(profilesPath, handle string) string {
	exts := []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}
	for _, ext := range exts {
		photoPath := filepath.Join(profilesPath, handle+ext)
		if _, err := os.Stat(photoPath); err == nil {
			return handle + ext
		}
	}
	return ""
}

func (l *MetaLoader) HydrateContentImages(ctx context.Context, siteID uuid.UUID, meta *BackupMeta) (*ContentImageHydrationResult, error) {
	result := &ContentImageHydrationResult{}

	if len(meta.ContentImages) == 0 {
		return result, nil
	}

	contents, err := l.service.GetAllContentWithMeta(ctx, siteID)
	if err != nil {
		return result, err
	}
	contentMap := make(map[string]uuid.UUID)
	for _, c := range contents {
		if c.ShortID != "" {
			contentMap[c.ShortID] = c.ID
		}
	}

	images, err := l.service.GetImages(ctx, siteID)
	if err != nil {
		return result, err
	}
	imageMap := make(map[string]uuid.UUID)
	for _, img := range images {
		imageMap[img.FilePath] = img.ID
	}

	for shortID, contentImgs := range meta.ContentImages {
		contentID, ok := contentMap[shortID]
		if !ok {
			continue
		}

		for _, ci := range contentImgs {
			imageID, ok := imageMap[ci.ImagePath]
			if !ok {
				result.Errors = append(result.Errors, "image not found: "+ci.ImagePath)
				continue
			}

			if err := l.service.LinkImageToContent(ctx, contentID, imageID, ci.IsHeader); err != nil {
				if !strings.Contains(err.Error(), "UNIQUE constraint") {
					result.Errors = append(result.Errors, "link "+shortID+": "+err.Error())
				}
				continue
			}
			result.LinksCreated++
		}
	}

	return result, nil
}

type ContentImageHydrationResult struct {
	LinksCreated int
	Errors       []string
}
