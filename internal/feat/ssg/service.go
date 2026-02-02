package ssg

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cliossg/clio/internal/db/sqlc"
	"github.com/cliossg/clio/pkg/cl/config"
	"github.com/cliossg/clio/pkg/cl/logger"
	"github.com/google/uuid"
)

var (
	ErrNotFound = errors.New("not found")
)

// Service defines the SSG service interface.
type Service interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	// Site operations
	CreateSite(ctx context.Context, site *Site) error
	GetSite(ctx context.Context, id uuid.UUID) (*Site, error)
	GetSiteBySlug(ctx context.Context, slug string) (*Site, error)
	ListSites(ctx context.Context) ([]*Site, error)
	UpdateSite(ctx context.Context, site *Site) error
	DeleteSite(ctx context.Context, id uuid.UUID) error

	// Content operations
	CreateContent(ctx context.Context, content *Content) error
	GetContent(ctx context.Context, id uuid.UUID) (*Content, error)
	GetContentWithMeta(ctx context.Context, id uuid.UUID) (*Content, error)
	GetAllContentWithMeta(ctx context.Context, siteID uuid.UUID) ([]*Content, error)
	GetContentWithPagination(ctx context.Context, siteID uuid.UUID, offset, limit int, search string) ([]*Content, int, error)
	UpdateContent(ctx context.Context, content *Content) error
	DeleteContent(ctx context.Context, id uuid.UUID) error

	// Section operations
	CreateSection(ctx context.Context, section *Section) error
	GetSection(ctx context.Context, id uuid.UUID) (*Section, error)
	GetSectionByPath(ctx context.Context, siteID uuid.UUID, path string) (*Section, error)
	GetSections(ctx context.Context, siteID uuid.UUID) ([]*Section, error)
	UpdateSection(ctx context.Context, section *Section) error
	DeleteSection(ctx context.Context, id uuid.UUID) error

	// Layout operations
	CreateLayout(ctx context.Context, layout *Layout) error
	GetLayout(ctx context.Context, id uuid.UUID) (*Layout, error)
	GetLayouts(ctx context.Context, siteID uuid.UUID) ([]*Layout, error)
	UpdateLayout(ctx context.Context, layout *Layout) error
	DeleteLayout(ctx context.Context, id uuid.UUID) error

	// Tag operations
	CreateTag(ctx context.Context, tag *Tag) error
	GetTag(ctx context.Context, id uuid.UUID) (*Tag, error)
	GetTagByName(ctx context.Context, siteID uuid.UUID, name string) (*Tag, error)
	GetTags(ctx context.Context, siteID uuid.UUID) ([]*Tag, error)
	UpdateTag(ctx context.Context, tag *Tag) error
	DeleteTag(ctx context.Context, id uuid.UUID) error
	AddTagToContent(ctx context.Context, contentID uuid.UUID, tagName string, siteID uuid.UUID) error
	AddTagToContentByID(ctx context.Context, contentID, tagID uuid.UUID) error
	RemoveTagFromContent(ctx context.Context, contentID, tagID uuid.UUID) error
	RemoveAllTagsFromContent(ctx context.Context, contentID uuid.UUID) error
	GetTagsForContent(ctx context.Context, contentID uuid.UUID) ([]*Tag, error)

	// Setting operations
	CreateSetting(ctx context.Context, param *Setting) error
	GetSetting(ctx context.Context, id uuid.UUID) (*Setting, error)
	GetSettingByName(ctx context.Context, siteID uuid.UUID, name string) (*Setting, error)
	GetSettingByRefKey(ctx context.Context, siteID uuid.UUID, refKey string) (*Setting, error)
	GetSettings(ctx context.Context, siteID uuid.UUID) ([]*Setting, error)
	UpdateSetting(ctx context.Context, param *Setting) error
	DeleteSetting(ctx context.Context, id uuid.UUID) error

	// Image operations
	CreateImage(ctx context.Context, image *Image) error
	GetImage(ctx context.Context, id uuid.UUID) (*Image, error)
	GetImages(ctx context.Context, siteID uuid.UUID) ([]*Image, error)
	GetImageByPath(ctx context.Context, siteID uuid.UUID, filePath string) (*Image, error)
	GetContentImagesWithDetails(ctx context.Context, contentID uuid.UUID) ([]*ContentImageWithDetails, error)
	GetAllContentImages(ctx context.Context, siteID uuid.UUID) (map[string][]MetaContentImage, error)
	GetContentImageDetails(ctx context.Context, contentImageID uuid.UUID) (*ContentImageDetails, error)
	LinkImageToContent(ctx context.Context, contentID, imageID uuid.UUID, isHeader bool) error
	UnlinkImageFromContent(ctx context.Context, contentImageID uuid.UUID) error
	UnlinkHeaderImageFromContent(ctx context.Context, contentID uuid.UUID) error
	GetSectionImagesWithDetails(ctx context.Context, sectionID uuid.UUID) ([]*SectionImageWithDetails, error)
	GetSectionImageDetails(ctx context.Context, sectionImageID uuid.UUID) (*SectionImageDetails, error)
	LinkImageToSection(ctx context.Context, sectionID, imageID uuid.UUID, isHeader bool) error
	UnlinkImageFromSection(ctx context.Context, sectionImageID uuid.UUID) error
	UpdateImage(ctx context.Context, image *Image) error
	DeleteImage(ctx context.Context, id uuid.UUID) error

	// Meta operations
	GetMetaByContentID(ctx context.Context, contentID uuid.UUID) (*Meta, error)
	CreateMeta(ctx context.Context, meta *Meta) error
	UpdateMeta(ctx context.Context, meta *Meta) error

	// Contributor operations
	CreateContributor(ctx context.Context, contributor *Contributor) error
	GetContributor(ctx context.Context, id uuid.UUID) (*Contributor, error)
	GetContributorByHandle(ctx context.Context, siteID uuid.UUID, handle string) (*Contributor, error)
	GetContributors(ctx context.Context, siteID uuid.UUID) ([]*Contributor, error)
	UpdateContributor(ctx context.Context, contributor *Contributor) error
	DeleteContributor(ctx context.Context, id uuid.UUID) error
	SetContributorProfile(ctx context.Context, contributorID, profileID uuid.UUID, updatedBy string) error

	// HTML generation
	GenerateHTMLForSite(ctx context.Context, siteSlug string) error
	BuildUserAuthorsMap(ctx context.Context, contents []*Content, contributors []*Contributor) map[string]*Contributor

	// Import operations
	CreateImport(ctx context.Context, imp *Import) error
	GetImport(ctx context.Context, id uuid.UUID) (*Import, error)
	GetImportByFilePath(ctx context.Context, filePath string) (*Import, error)
	GetImportByContentID(ctx context.Context, contentID uuid.UUID) (*Import, error)
	ListImports(ctx context.Context, siteID uuid.UUID) ([]*Import, error)
	UpdateImport(ctx context.Context, imp *Import) error
	UpdateImportStatus(ctx context.Context, id uuid.UUID, status string) error
	DeleteImport(ctx context.Context, id uuid.UUID) error
	ScanImportDirectory(ctx context.Context, importPath string) ([]ImportFile, error)
	ImportFile(ctx context.Context, siteID, userID uuid.UUID, file ImportFile, sectionID uuid.UUID) (*Content, *Import, error)
	ReimportFile(ctx context.Context, importID uuid.UUID, force bool) (*Content, error)
}

// DBProvider provides access to the database.
type DBProvider interface {
	GetDB() *sql.DB
}

type service struct {
	dbProvider DBProvider
	queries    *sqlc.Queries
	htmlGen    *HTMLGenerator
	cfg        *config.Config
	log        logger.Logger
}

// NewService creates a new SSG service.
func NewService(dbProvider DBProvider, htmlGen *HTMLGenerator, cfg *config.Config, log logger.Logger) Service {
	return &service{
		dbProvider: dbProvider,
		htmlGen:    htmlGen,
		cfg:        cfg,
		log:        log,
	}
}

func (s *service) Start(ctx context.Context) error {
	s.log.Info("SSG service started")
	return nil
}

func (s *service) Stop(ctx context.Context) error {
	s.log.Info("SSG service stopped")
	return nil
}

func (s *service) ensureQueries() {
	if s.queries == nil && s.dbProvider != nil {
		s.queries = sqlc.New(s.dbProvider.GetDB())
	}
}

// --- Site Operations ---

func (s *service) CreateSite(ctx context.Context, site *Site) error {
	s.ensureQueries()

	if s.queries == nil {
		s.log.Error("CreateSite: queries is nil - dbProvider may not be initialized")
		return fmt.Errorf("database not initialized")
	}

	s.log.Infof("CreateSite: creating site with slug=%s, name=%s", site.Slug, site.Name)

	params := sqlc.CreateSiteParams{
		ID:        site.ID.String(),
		ShortID:   site.ShortID,
		Name:      site.Name,
		Slug:      site.Slug,
		Active:    boolToInt(site.Active),
		CreatedBy: site.CreatedBy.String(),
		UpdatedBy: site.UpdatedBy.String(),
		CreatedAt: site.CreatedAt,
		UpdatedAt: site.UpdatedAt,
	}

	_, err := s.queries.CreateSite(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot create site: %w", err)
	}

	return nil
}

func (s *service) GetSite(ctx context.Context, id uuid.UUID) (*Site, error) {
	s.ensureQueries()

	sqlcSite, err := s.queries.GetSite(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get site: %w", err)
	}

	return siteFromSQLC(sqlcSite), nil
}

func (s *service) GetSiteBySlug(ctx context.Context, slug string) (*Site, error) {
	s.ensureQueries()

	sqlcSite, err := s.queries.GetSiteBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get site by slug: %w", err)
	}

	return siteFromSQLC(sqlcSite), nil
}

func (s *service) ListSites(ctx context.Context) ([]*Site, error) {
	s.ensureQueries()

	sqlcSites, err := s.queries.ListSites(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot list sites: %w", err)
	}

	sites := make([]*Site, len(sqlcSites))
	for i, sqlcSite := range sqlcSites {
		sites[i] = siteFromSQLC(sqlcSite)
	}

	return sites, nil
}

func (s *service) UpdateSite(ctx context.Context, site *Site) error {
	s.ensureQueries()

	params := sqlc.UpdateSiteParams{
		Name:              site.Name,
		Slug:              site.Slug,
		Active:            boolToInt(site.Active),
		DefaultLayoutID:   nullString(site.DefaultLayoutID.String()),
		DefaultLayoutName: nullString(site.DefaultLayoutName),
		UpdatedBy:         site.UpdatedBy.String(),
		UpdatedAt:         site.UpdatedAt,
		ID:                site.ID.String(),
	}

	_, err := s.queries.UpdateSite(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot update site: %w", err)
	}

	return nil
}

func (s *service) DeleteSite(ctx context.Context, id uuid.UUID) error {
	s.ensureQueries()

	err := s.queries.DeleteSite(ctx, id.String())
	if err != nil {
		return fmt.Errorf("cannot delete site: %w", err)
	}

	return nil
}

// --- Content Operations ---

func (s *service) CreateContent(ctx context.Context, content *Content) error {
	s.ensureQueries()

	var contributorID sql.NullString
	if content.ContributorID != nil {
		contributorID = nullString(content.ContributorID.String())
	}

	imagesMeta := s.buildImagesMeta(ctx, content.SiteID, content.Body)

	params := sqlc.CreateContentParams{
		ID:                content.ID.String(),
		SiteID:            content.SiteID.String(),
		UserID:            nullString(content.UserID.String()),
		ShortID:           nullString(content.ShortID),
		SectionID:         nullString(content.SectionID.String()),
		ContributorID:     contributorID,
		ContributorHandle: content.ContributorHandle,
		AuthorUsername:    content.AuthorUsername,
		Kind:              nullString(content.Kind),
		Heading:           content.Heading,
		Summary:           nullString(content.Summary),
		Body:              nullString(content.Body),
		Draft:             nullInt(boolToInt(content.Draft)),
		Featured:          nullInt(boolToInt(content.Featured)),
		Series:            nullString(content.Series),
		SeriesOrder:       nullInt(int64(content.SeriesOrder)),
		PublishedAt:       nullTime(content.PublishedAt),
		HeroTitleDark:     nullInt(boolToInt(content.HeroTitleDark)),
		ImagesMeta:        nullString(imagesMeta),
		CreatedBy:         nullString(content.CreatedBy.String()),
		UpdatedBy:         nullString(content.UpdatedBy.String()),
		CreatedAt:         nullTime(&content.CreatedAt),
		UpdatedAt:         nullTime(&content.UpdatedAt),
	}

	_, err := s.queries.CreateContent(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot create content: %w", err)
	}

	return nil
}

func (s *service) GetContent(ctx context.Context, id uuid.UUID) (*Content, error) {
	s.ensureQueries()

	sqlcContent, err := s.queries.GetContent(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get content: %w", err)
	}

	return contentFromSQLC(sqlcContent), nil
}

func (s *service) GetContentWithMeta(ctx context.Context, id uuid.UUID) (*Content, error) {
	s.ensureQueries()

	row, err := s.queries.GetContentWithMeta(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get content with meta: %w", err)
	}

	content := contentWithMetaFromSQLC(row)

	// Get tags
	tags, err := s.GetTagsForContent(ctx, id)
	if err == nil {
		content.Tags = tags
	}

	return content, nil
}

func (s *service) GetAllContentWithMeta(ctx context.Context, siteID uuid.UUID) ([]*Content, error) {
	s.ensureQueries()

	rows, err := s.queries.GetAllContentWithMeta(ctx, siteID.String())
	if err != nil {
		return nil, fmt.Errorf("cannot get all content: %w", err)
	}

	contents := make([]*Content, len(rows))
	for i, row := range rows {
		contents[i] = contentWithMetaFromSQLCAll(row)
		// Load tags for each content
		tags, err := s.GetTagsForContent(ctx, contents[i].ID)
		if err == nil {
			contents[i].Tags = tags
		}
	}

	return contents, nil
}

func (s *service) GetContentWithPagination(ctx context.Context, siteID uuid.UUID, offset, limit int, search string) ([]*Content, int, error) {
	s.ensureQueries()

	var contents []*Content
	var total int64

	if search != "" {
		searchPattern := "%" + search + "%"
		rows, err := s.queries.SearchContent(ctx, sqlc.SearchContentParams{
			SiteID:  siteID.String(),
			Heading: searchPattern,
			Limit:   int64(limit),
			Offset:  int64(offset),
		})
		if err != nil {
			return nil, 0, fmt.Errorf("cannot search content: %w", err)
		}

		contents = make([]*Content, len(rows))
		for i, row := range rows {
			contents[i] = contentFromSQLC(row)
		}

		total, _ = s.queries.CountSearchContent(ctx, sqlc.CountSearchContentParams{
			SiteID:  siteID.String(),
			Heading: searchPattern,
		})
	} else {
		rows, err := s.queries.GetContentWithPagination(ctx, sqlc.GetContentWithPaginationParams{
			SiteID: siteID.String(),
			Limit:  int64(limit),
			Offset: int64(offset),
		})
		if err != nil {
			return nil, 0, fmt.Errorf("cannot get content: %w", err)
		}

		contents = make([]*Content, len(rows))
		for i, row := range rows {
			contents[i] = contentFromSQLC(row)
		}

		total, _ = s.queries.CountContent(ctx, siteID.String())
	}

	return contents, int(total), nil
}

func (s *service) UpdateContent(ctx context.Context, content *Content) error {
	s.ensureQueries()

	content.UpdatedAt = time.Now()

	var contributorID sql.NullString
	if content.ContributorID != nil {
		contributorID = nullString(content.ContributorID.String())
	}

	imagesMeta := s.buildImagesMeta(ctx, content.SiteID, content.Body)

	params := sqlc.UpdateContentParams{
		SectionID:         nullString(content.SectionID.String()),
		ContributorID:     contributorID,
		ContributorHandle: content.ContributorHandle,
		AuthorUsername:    content.AuthorUsername,
		Kind:              nullString(content.Kind),
		Heading:           content.Heading,
		Summary:           nullString(content.Summary),
		Body:              nullString(content.Body),
		Draft:             nullInt(boolToInt(content.Draft)),
		Featured:          nullInt(boolToInt(content.Featured)),
		Series:            nullString(content.Series),
		SeriesOrder:       nullInt(int64(content.SeriesOrder)),
		PublishedAt:       nullTime(content.PublishedAt),
		HeroTitleDark:     nullInt(boolToInt(content.HeroTitleDark)),
		ImagesMeta:        nullString(imagesMeta),
		UpdatedBy:         nullString(content.UpdatedBy.String()),
		UpdatedAt:         nullTime(&content.UpdatedAt),
		ID:                content.ID.String(),
	}

	_, err := s.queries.UpdateContent(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot update content: %w", err)
	}

	return nil
}

func (s *service) DeleteContent(ctx context.Context, id uuid.UUID) error {
	s.ensureQueries()

	err := s.queries.DeleteContent(ctx, id.String())
	if err != nil {
		return fmt.Errorf("cannot delete content: %w", err)
	}

	return nil
}

// --- Section Operations ---

func (s *service) CreateSection(ctx context.Context, section *Section) error {
	s.ensureQueries()

	params := sqlc.CreateSectionParams{
		ID:            section.ID.String(),
		SiteID:        section.SiteID.String(),
		ShortID:       nullString(section.ShortID),
		Name:          section.Name,
		Description:   nullString(section.Description),
		Path:          sql.NullString{String: section.Path, Valid: true},
		LayoutID:      nullString(section.LayoutID.String()),
		LayoutName:    nullString(section.LayoutName),
		HeroTitleDark: nullInt(boolToInt(section.HeroTitleDark)),
		CreatedBy:     nullString(section.CreatedBy.String()),
		UpdatedBy:     nullString(section.UpdatedBy.String()),
		CreatedAt:     nullTime(&section.CreatedAt),
		UpdatedAt:     nullTime(&section.UpdatedAt),
	}

	_, err := s.queries.CreateSection(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot create section: %w", err)
	}

	return nil
}

func (s *service) GetSection(ctx context.Context, id uuid.UUID) (*Section, error) {
	s.ensureQueries()

	sqlcSection, err := s.queries.GetSection(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get section: %w", err)
	}

	return sectionFromSQLC(sqlcSection), nil
}

func (s *service) GetSectionByPath(ctx context.Context, siteID uuid.UUID, path string) (*Section, error) {
	s.ensureQueries()

	sqlcSection, err := s.queries.GetSectionByPath(ctx, sqlc.GetSectionByPathParams{
		SiteID: siteID.String(),
		Path:   sql.NullString{String: path, Valid: true},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get section by path: %w", err)
	}

	return sectionFromSQLC(sqlcSection), nil
}

func (s *service) GetSections(ctx context.Context, siteID uuid.UUID) ([]*Section, error) {
	s.ensureQueries()

	rows, err := s.queries.GetSectionsWithHeaderImage(ctx, siteID.String())
	if err != nil {
		return nil, fmt.Errorf("cannot get sections: %w", err)
	}

	sections := make([]*Section, len(rows))
	for i, row := range rows {
		section := &Section{
			ID:     parseUUID(row.ID),
			SiteID: parseUUID(row.SiteID),
			Name:   row.Name,
		}
		if row.ShortID.Valid {
			section.ShortID = row.ShortID.String
		}
		if row.Description.Valid {
			section.Description = row.Description.String
		}
		if row.Path.Valid {
			section.Path = row.Path.String
		}
		if row.LayoutID.Valid {
			section.LayoutID = parseUUID(row.LayoutID.String)
		}
		if row.LayoutName.Valid {
			section.LayoutName = row.LayoutName.String
		}
		if row.HeaderImagePath.Valid {
			section.HeaderImageURL = "/images/" + row.HeaderImagePath.String
		}
		if row.HeroTitleDark.Valid {
			section.HeroTitleDark = row.HeroTitleDark.Int64 == 1
		}
		if row.CreatedAt.Valid {
			section.CreatedAt = row.CreatedAt.Time
		}
		if row.UpdatedAt.Valid {
			section.UpdatedAt = row.UpdatedAt.Time
		}
		sections[i] = section
	}

	return sections, nil
}

func (s *service) UpdateSection(ctx context.Context, section *Section) error {
	s.ensureQueries()

	params := sqlc.UpdateSectionParams{
		Name:          section.Name,
		Description:   nullString(section.Description),
		Path:          sql.NullString{String: section.Path, Valid: true},
		LayoutID:      nullString(section.LayoutID.String()),
		LayoutName:    nullString(section.LayoutName),
		HeroTitleDark: nullInt(boolToInt(section.HeroTitleDark)),
		UpdatedBy:     nullString(section.UpdatedBy.String()),
		UpdatedAt:     nullTime(&section.UpdatedAt),
		ID:            section.ID.String(),
	}

	_, err := s.queries.UpdateSection(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot update section: %w", err)
	}

	return nil
}

func (s *service) DeleteSection(ctx context.Context, id uuid.UUID) error {
	s.ensureQueries()

	err := s.queries.DeleteSection(ctx, id.String())
	if err != nil {
		return fmt.Errorf("cannot delete section: %w", err)
	}

	return nil
}

// --- Layout Operations ---

func (s *service) CreateLayout(ctx context.Context, layout *Layout) error {
	s.ensureQueries()

	params := sqlc.CreateLayoutParams{
		ID:                layout.ID.String(),
		SiteID:            layout.SiteID.String(),
		ShortID:           nullString(layout.ShortID),
		Name:              layout.Name,
		Description:       nullString(layout.Description),
		Code:              nullString(layout.Code),
		Css:               nullString(layout.CSS),
		ExcludeDefaultCss: nullInt(boolToInt(layout.ExcludeDefaultCSS)),
		HeaderImageID:     nullString(layout.HeaderImageID.String()),
		CreatedBy:         nullString(layout.CreatedBy.String()),
		UpdatedBy:         nullString(layout.UpdatedBy.String()),
		CreatedAt:         nullTime(&layout.CreatedAt),
		UpdatedAt:         nullTime(&layout.UpdatedAt),
	}

	_, err := s.queries.CreateLayout(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot create layout: %w", err)
	}

	return nil
}

func (s *service) GetLayout(ctx context.Context, id uuid.UUID) (*Layout, error) {
	s.ensureQueries()

	sqlcLayout, err := s.queries.GetLayout(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get layout: %w", err)
	}

	return layoutFromSQLC(sqlcLayout), nil
}

func (s *service) GetLayouts(ctx context.Context, siteID uuid.UUID) ([]*Layout, error) {
	s.ensureQueries()

	sqlcLayouts, err := s.queries.GetLayoutsBySiteID(ctx, siteID.String())
	if err != nil {
		return nil, fmt.Errorf("cannot get layouts: %w", err)
	}

	layouts := make([]*Layout, len(sqlcLayouts))
	for i, sqlcLayout := range sqlcLayouts {
		layouts[i] = layoutFromSQLC(sqlcLayout)
	}

	return layouts, nil
}

func (s *service) UpdateLayout(ctx context.Context, layout *Layout) error {
	s.ensureQueries()

	params := sqlc.UpdateLayoutParams{
		Name:              layout.Name,
		Description:       nullString(layout.Description),
		Code:              nullString(layout.Code),
		Css:               nullString(layout.CSS),
		ExcludeDefaultCss: nullInt(boolToInt(layout.ExcludeDefaultCSS)),
		HeaderImageID:     nullString(layout.HeaderImageID.String()),
		UpdatedBy:         nullString(layout.UpdatedBy.String()),
		UpdatedAt:         nullTime(&layout.UpdatedAt),
		ID:                layout.ID.String(),
	}

	_, err := s.queries.UpdateLayout(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot update layout: %w", err)
	}

	return nil
}

func (s *service) DeleteLayout(ctx context.Context, id uuid.UUID) error {
	s.ensureQueries()

	err := s.queries.DeleteLayout(ctx, id.String())
	if err != nil {
		return fmt.Errorf("cannot delete layout: %w", err)
	}

	return nil
}

// --- Tag Operations ---

func (s *service) CreateTag(ctx context.Context, tag *Tag) error {
	s.ensureQueries()

	params := sqlc.CreateTagParams{
		ID:        tag.ID.String(),
		SiteID:    tag.SiteID.String(),
		ShortID:   nullString(tag.ShortID),
		Name:      tag.Name,
		Slug:      tag.Slug,
		CreatedBy: nullString(tag.CreatedBy.String()),
		UpdatedBy: nullString(tag.UpdatedBy.String()),
		CreatedAt: nullTime(&tag.CreatedAt),
		UpdatedAt: nullTime(&tag.UpdatedAt),
	}

	_, err := s.queries.CreateTag(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot create tag: %w", err)
	}

	return nil
}

func (s *service) GetTag(ctx context.Context, id uuid.UUID) (*Tag, error) {
	s.ensureQueries()

	sqlcTag, err := s.queries.GetTag(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get tag: %w", err)
	}

	return tagFromSQLC(sqlcTag), nil
}

func (s *service) GetTagByName(ctx context.Context, siteID uuid.UUID, name string) (*Tag, error) {
	s.ensureQueries()

	sqlcTag, err := s.queries.GetTagByName(ctx, sqlc.GetTagByNameParams{
		SiteID: siteID.String(),
		Name:   name,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get tag by name: %w", err)
	}

	return tagFromSQLC(sqlcTag), nil
}

func (s *service) GetTags(ctx context.Context, siteID uuid.UUID) ([]*Tag, error) {
	s.ensureQueries()

	sqlcTags, err := s.queries.GetTagsBySiteID(ctx, siteID.String())
	if err != nil {
		return nil, fmt.Errorf("cannot get tags: %w", err)
	}

	tags := make([]*Tag, len(sqlcTags))
	for i, sqlcTag := range sqlcTags {
		tags[i] = tagFromSQLC(sqlcTag)
	}

	return tags, nil
}

func (s *service) UpdateTag(ctx context.Context, tag *Tag) error {
	s.ensureQueries()

	params := sqlc.UpdateTagParams{
		Name:      tag.Name,
		Slug:      tag.Slug,
		UpdatedBy: nullString(tag.UpdatedBy.String()),
		UpdatedAt: nullTime(&tag.UpdatedAt),
		ID:        tag.ID.String(),
	}

	_, err := s.queries.UpdateTag(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot update tag: %w", err)
	}

	return nil
}

func (s *service) DeleteTag(ctx context.Context, id uuid.UUID) error {
	s.ensureQueries()

	err := s.queries.DeleteTag(ctx, id.String())
	if err != nil {
		return fmt.Errorf("cannot delete tag: %w", err)
	}

	return nil
}

func (s *service) AddTagToContent(ctx context.Context, contentID uuid.UUID, tagName string, siteID uuid.UUID) error {
	s.ensureQueries()

	// Get or create tag
	tag, err := s.GetTagByName(ctx, siteID, tagName)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			tag = NewTag(siteID, tagName)
			if err := s.CreateTag(ctx, tag); err != nil {
				return fmt.Errorf("cannot create tag: %w", err)
			}
		} else {
			return fmt.Errorf("cannot get tag: %w", err)
		}
	}

	err = s.queries.AddTagToContent(ctx, sqlc.AddTagToContentParams{
		ID:        uuid.New().String(),
		ContentID: contentID.String(),
		TagID:     tag.ID.String(),
		CreatedAt: nullTime(timePtr(time.Now())),
	})
	if err != nil {
		return fmt.Errorf("cannot add tag to content: %w", err)
	}

	return nil
}

func (s *service) RemoveTagFromContent(ctx context.Context, contentID, tagID uuid.UUID) error {
	s.ensureQueries()

	err := s.queries.RemoveTagFromContent(ctx, sqlc.RemoveTagFromContentParams{
		ContentID: contentID.String(),
		TagID:     tagID.String(),
	})
	if err != nil {
		return fmt.Errorf("cannot remove tag from content: %w", err)
	}

	return nil
}

func (s *service) AddTagToContentByID(ctx context.Context, contentID, tagID uuid.UUID) error {
	s.ensureQueries()

	id := uuid.New()
	err := s.queries.AddTagToContent(ctx, sqlc.AddTagToContentParams{
		ID:        id.String(),
		ContentID: contentID.String(),
		TagID:     tagID.String(),
		CreatedAt: nullTime(timePtr(time.Now())),
	})
	if err != nil {
		return fmt.Errorf("cannot add tag to content: %w", err)
	}

	return nil
}

func (s *service) RemoveAllTagsFromContent(ctx context.Context, contentID uuid.UUID) error {
	s.ensureQueries()

	err := s.queries.RemoveAllTagsFromContent(ctx, contentID.String())
	if err != nil {
		return fmt.Errorf("cannot remove all tags from content: %w", err)
	}

	return nil
}

func (s *service) GetTagsForContent(ctx context.Context, contentID uuid.UUID) ([]*Tag, error) {
	s.ensureQueries()

	sqlcTags, err := s.queries.GetTagsForContent(ctx, contentID.String())
	if err != nil {
		return nil, fmt.Errorf("cannot get tags for content: %w", err)
	}

	tags := make([]*Tag, len(sqlcTags))
	for i, sqlcTag := range sqlcTags {
		tags[i] = tagFromSQLC(sqlcTag)
	}

	return tags, nil
}

// --- Setting Operations ---

func (s *service) CreateSetting(ctx context.Context, param *Setting) error {
	s.ensureQueries()

	params := sqlc.CreateSettingParams{
		ID:          param.ID.String(),
		SiteID:      param.SiteID.String(),
		ShortID:     nullString(param.ShortID),
		Name:        param.Name,
		Description: nullString(param.Description),
		Value:       nullString(param.Value),
		RefKey:      nullString(param.RefKey),
		Category:    nullString(param.Category),
		Position:    nullInt(int64(param.Position)),
		System:      nullInt(boolToInt(param.System)),
		CreatedBy:   nullString(param.CreatedBy.String()),
		UpdatedBy:   nullString(param.UpdatedBy.String()),
		CreatedAt:   nullTime(&param.CreatedAt),
		UpdatedAt:   nullTime(&param.UpdatedAt),
	}

	_, err := s.queries.CreateSetting(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot create param: %w", err)
	}

	return nil
}

func (s *service) GetSetting(ctx context.Context, id uuid.UUID) (*Setting, error) {
	s.ensureQueries()

	sqlcParam, err := s.queries.GetSetting(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get param: %w", err)
	}

	return settingFromSQLC(sqlcParam), nil
}

func (s *service) GetSettingByName(ctx context.Context, siteID uuid.UUID, name string) (*Setting, error) {
	s.ensureQueries()

	sqlcParam, err := s.queries.GetSettingByName(ctx, sqlc.GetSettingByNameParams{
		SiteID: siteID.String(),
		Name:   name,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get param by name: %w", err)
	}

	return settingFromSQLC(sqlcParam), nil
}

func (s *service) GetSettingByRefKey(ctx context.Context, siteID uuid.UUID, refKey string) (*Setting, error) {
	s.ensureQueries()

	sqlcParam, err := s.queries.GetSettingByRefKey(ctx, sqlc.GetSettingByRefKeyParams{
		SiteID: siteID.String(),
		RefKey: sql.NullString{String: refKey, Valid: true},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get param by ref key: %w", err)
	}

	return settingFromSQLC(sqlcParam), nil
}

func (s *service) GetSettings(ctx context.Context, siteID uuid.UUID) ([]*Setting, error) {
	s.ensureQueries()

	sqlcParams, err := s.queries.GetSettingsBySiteID(ctx, siteID.String())
	if err != nil {
		return nil, fmt.Errorf("cannot get params: %w", err)
	}

	params := make([]*Setting, len(sqlcParams))
	for i, sqlcParam := range sqlcParams {
		params[i] = settingFromSQLC(sqlcParam)
	}

	return params, nil
}

func (s *service) UpdateSetting(ctx context.Context, param *Setting) error {
	s.ensureQueries()

	params := sqlc.UpdateSettingParams{
		Name:        param.Name,
		Description: nullString(param.Description),
		Value:       nullString(param.Value),
		RefKey:      nullString(param.RefKey),
		Category:    nullString(param.Category),
		Position:    nullInt(int64(param.Position)),
		System:      nullInt(boolToInt(param.System)),
		UpdatedBy:   nullString(param.UpdatedBy.String()),
		UpdatedAt:   nullTime(&param.UpdatedAt),
		ID:          param.ID.String(),
	}

	_, err := s.queries.UpdateSetting(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot update param: %w", err)
	}

	return nil
}

func (s *service) DeleteSetting(ctx context.Context, id uuid.UUID) error {
	s.ensureQueries()

	err := s.queries.DeleteSetting(ctx, id.String())
	if err != nil {
		return fmt.Errorf("cannot delete param: %w", err)
	}

	return nil
}

// --- Image Operations ---

func (s *service) CreateImage(ctx context.Context, image *Image) error {
	s.ensureQueries()

	params := sqlc.CreateImageParams{
		ID:             image.ID.String(),
		SiteID:         image.SiteID.String(),
		ShortID:        nullString(image.ShortID),
		FileName:       image.FileName,
		FilePath:       image.FilePath,
		AltText:        nullString(image.AltText),
		Title:          nullString(image.Title),
		Attribution:    nullString(image.Attribution),
		AttributionUrl: nullString(image.AttributionURL),
		Width:          nullInt(int64(image.Width)),
		Height:         nullInt(int64(image.Height)),
		CreatedBy:      nullString(image.CreatedBy.String()),
		UpdatedBy:      nullString(image.UpdatedBy.String()),
		CreatedAt:      nullTime(&image.CreatedAt),
		UpdatedAt:      nullTime(&image.UpdatedAt),
	}

	_, err := s.queries.CreateImage(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot create image: %w", err)
	}

	return nil
}

func (s *service) GetImage(ctx context.Context, id uuid.UUID) (*Image, error) {
	s.ensureQueries()

	sqlcImage, err := s.queries.GetImage(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get image: %w", err)
	}

	return imageFromSQLC(sqlcImage), nil
}

func (s *service) GetImages(ctx context.Context, siteID uuid.UUID) ([]*Image, error) {
	s.ensureQueries()

	sqlcImages, err := s.queries.GetImagesBySiteID(ctx, siteID.String())
	if err != nil {
		return nil, fmt.Errorf("cannot get images: %w", err)
	}

	images := make([]*Image, len(sqlcImages))
	for i, sqlcImage := range sqlcImages {
		images[i] = imageFromSQLC(sqlcImage)
	}

	return images, nil
}

func (s *service) GetImageByPath(ctx context.Context, siteID uuid.UUID, filePath string) (*Image, error) {
	s.ensureQueries()

	sqlcImage, err := s.queries.GetImageByPath(ctx, sqlc.GetImageByPathParams{
		SiteID:   siteID.String(),
		FilePath: filePath,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get image by path: %w", err)
	}

	return imageFromSQLC(sqlcImage), nil
}

func (s *service) GetContentImagesWithDetails(ctx context.Context, contentID uuid.UUID) ([]*ContentImageWithDetails, error) {
	s.ensureQueries()

	rows, err := s.queries.GetContentImagesWithDetails(ctx, contentID.String())
	if err != nil {
		return nil, fmt.Errorf("cannot get content images: %w", err)
	}

	images := make([]*ContentImageWithDetails, len(rows))
	for i, row := range rows {
		images[i] = &ContentImageWithDetails{
			ContentImageID: parseUUID(row.ContentImageID),
			ContentID:      parseUUID(row.ContentID),
			IsHeader:       row.IsHeader.Int64 == 1,
			IsFeatured:     row.IsFeatured.Int64 == 1,
			OrderNum:       int(row.OrderNum.Int64),
			ID:             parseUUID(row.ID),
			SiteID:         parseUUID(row.SiteID),
			ShortID:        row.ShortID.String,
			FileName:       row.FileName,
			FilePath:       row.FilePath,
			AltText:        row.AltText.String,
			Title:          row.Title.String,
			Width:          int(row.Width.Int64),
			Height:         int(row.Height.Int64),
			CreatedAt:      row.CreatedAt.Time,
			UpdatedAt:      row.UpdatedAt.Time,
		}
	}

	return images, nil
}

func (s *service) GetAllContentImages(ctx context.Context, siteID uuid.UUID) (map[string][]MetaContentImage, error) {
	s.ensureQueries()

	rows, err := s.queries.GetAllContentImagesBySiteID(ctx, siteID.String())
	if err != nil {
		return nil, fmt.Errorf("cannot get all content images: %w", err)
	}

	result := make(map[string][]MetaContentImage)
	for _, row := range rows {
		if !row.ContentShortID.Valid {
			continue
		}
		shortID := row.ContentShortID.String
		result[shortID] = append(result[shortID], MetaContentImage{
			ImagePath:  row.ImagePath,
			IsHeader:   row.IsHeader.Int64 == 1,
			IsFeatured: row.IsFeatured.Int64 == 1,
			OrderNum:   int(row.OrderNum.Int64),
		})
	}

	return result, nil
}

func (s *service) LinkImageToContent(ctx context.Context, contentID, imageID uuid.UUID, isHeader bool) error {
	s.ensureQueries()

	isHeaderInt := int64(0)
	if isHeader {
		isHeaderInt = 1
	}

	params := sqlc.CreateContentImageParams{
		ID:        uuid.New().String(),
		ContentID: contentID.String(),
		ImageID:   imageID.String(),
		IsHeader:  sql.NullInt64{Int64: isHeaderInt, Valid: true},
		IsFeatured: sql.NullInt64{Int64: 0, Valid: true},
		OrderNum:  sql.NullInt64{Int64: 0, Valid: true},
		CreatedAt: sql.NullTime{Time: time.Now(), Valid: true},
	}

	if err := s.queries.CreateContentImage(ctx, params); err != nil {
		return fmt.Errorf("cannot link image to content: %w", err)
	}

	return nil
}

func (s *service) GetContentImageDetails(ctx context.Context, contentImageID uuid.UUID) (*ContentImageDetails, error) {
	s.ensureQueries()

	row, err := s.queries.GetContentImageWithDetails(ctx, contentImageID.String())
	if err != nil {
		return nil, fmt.Errorf("cannot get content image details: %w", err)
	}

	return &ContentImageDetails{
		ContentImageID: parseUUID(row.ContentImageID),
		ImageID:        parseUUID(row.ImageID),
		FilePath:       row.FilePath,
	}, nil
}

func (s *service) UnlinkImageFromContent(ctx context.Context, contentImageID uuid.UUID) error {
	s.ensureQueries()

	if err := s.queries.DeleteContentImage(ctx, contentImageID.String()); err != nil {
		return fmt.Errorf("cannot unlink image from content: %w", err)
	}

	return nil
}

func (s *service) UnlinkHeaderImageFromContent(ctx context.Context, contentID uuid.UUID) error {
	s.ensureQueries()

	images, err := s.GetContentImagesWithDetails(ctx, contentID)
	if err != nil {
		return fmt.Errorf("cannot get content images: %w", err)
	}

	for _, img := range images {
		if img.IsHeader {
			if err := s.queries.DeleteContentImage(ctx, img.ContentImageID.String()); err != nil {
				return fmt.Errorf("cannot unlink header image: %w", err)
			}
			break
		}
	}

	return nil
}

func (s *service) GetSectionImagesWithDetails(ctx context.Context, sectionID uuid.UUID) ([]*SectionImageWithDetails, error) {
	s.ensureQueries()

	rows, err := s.queries.GetSectionImagesWithDetails(ctx, sectionID.String())
	if err != nil {
		return nil, fmt.Errorf("cannot get section images: %w", err)
	}

	images := make([]*SectionImageWithDetails, len(rows))
	for i, row := range rows {
		images[i] = &SectionImageWithDetails{
			SectionImageID: parseUUID(row.SectionImageID),
			SectionID:      parseUUID(row.SectionID),
			IsHeader:       row.IsHeader.Int64 == 1,
			IsFeatured:     row.IsFeatured.Int64 == 1,
			OrderNum:       int(row.OrderNum.Int64),
			ID:             parseUUID(row.ID),
			SiteID:         parseUUID(row.SiteID),
			ShortID:        row.ShortID.String,
			FileName:       row.FileName,
			FilePath:       row.FilePath,
			AltText:        row.AltText.String,
			Title:          row.Title.String,
			Width:          int(row.Width.Int64),
			Height:         int(row.Height.Int64),
			CreatedAt:      row.CreatedAt.Time,
			UpdatedAt:      row.UpdatedAt.Time,
		}
	}

	return images, nil
}

func (s *service) GetSectionImageDetails(ctx context.Context, sectionImageID uuid.UUID) (*SectionImageDetails, error) {
	s.ensureQueries()

	row, err := s.queries.GetSectionImageWithDetails(ctx, sectionImageID.String())
	if err != nil {
		return nil, fmt.Errorf("cannot get section image details: %w", err)
	}

	return &SectionImageDetails{
		SectionImageID: parseUUID(row.SectionImageID),
		ImageID:        parseUUID(row.ImageID),
		FilePath:       row.FilePath,
	}, nil
}

func (s *service) LinkImageToSection(ctx context.Context, sectionID, imageID uuid.UUID, isHeader bool) error {
	s.ensureQueries()

	isHeaderInt := int64(0)
	if isHeader {
		isHeaderInt = 1
	}

	params := sqlc.CreateSectionImageParams{
		ID:         uuid.New().String(),
		SectionID:  sectionID.String(),
		ImageID:    imageID.String(),
		IsHeader:   sql.NullInt64{Int64: isHeaderInt, Valid: true},
		IsFeatured: sql.NullInt64{Int64: 0, Valid: true},
		OrderNum:   sql.NullInt64{Int64: 0, Valid: true},
		CreatedAt:  sql.NullTime{Time: time.Now(), Valid: true},
	}

	if err := s.queries.CreateSectionImage(ctx, params); err != nil {
		return fmt.Errorf("cannot link image to section: %w", err)
	}

	return nil
}

func (s *service) UnlinkImageFromSection(ctx context.Context, sectionImageID uuid.UUID) error {
	s.ensureQueries()

	if err := s.queries.DeleteSectionImage(ctx, sectionImageID.String()); err != nil {
		return fmt.Errorf("cannot unlink image from section: %w", err)
	}

	return nil
}

func (s *service) UpdateImage(ctx context.Context, image *Image) error {
	s.ensureQueries()

	params := sqlc.UpdateImageParams{
		FileName:  image.FileName,
		FilePath:  image.FilePath,
		AltText:   nullString(image.AltText),
		Title:     nullString(image.Title),
		Width:     nullInt(int64(image.Width)),
		Height:    nullInt(int64(image.Height)),
		UpdatedBy: nullString(image.UpdatedBy.String()),
		UpdatedAt: nullTime(&image.UpdatedAt),
		ID:        image.ID.String(),
	}

	_, err := s.queries.UpdateImage(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot update image: %w", err)
	}

	return nil
}

func (s *service) DeleteImage(ctx context.Context, id uuid.UUID) error {
	s.ensureQueries()

	err := s.queries.DeleteImage(ctx, id.String())
	if err != nil {
		return fmt.Errorf("cannot delete image: %w", err)
	}

	return nil
}

// --- Meta Operations ---

func (s *service) GetMetaByContentID(ctx context.Context, contentID uuid.UUID) (*Meta, error) {
	s.ensureQueries()

	sqlcMeta, err := s.queries.GetMetaByContentID(ctx, contentID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No meta yet, not an error
		}
		return nil, fmt.Errorf("cannot get meta: %w", err)
	}

	return metaFromSQLC(sqlcMeta), nil
}

func (s *service) CreateMeta(ctx context.Context, meta *Meta) error {
	s.ensureQueries()

	params := sqlc.CreateMetaParams{
		ID:              meta.ID.String(),
		SiteID:          meta.SiteID.String(),
		ShortID:         nullString(meta.ShortID),
		ContentID:       meta.ContentID.String(),
		Summary:         nullString(meta.Summary),
		Excerpt:         nullString(meta.Excerpt),
		Description:     nullString(meta.Description),
		Keywords:        nullString(meta.Keywords),
		Robots:          nullString(meta.Robots),
		CanonicalUrl:    nullString(meta.CanonicalURL),
		Sitemap:         nullString(meta.Sitemap),
		TableOfContents: nullInt(boolToInt(meta.TableOfContents)),
		Share:           nullInt(boolToInt(meta.Share)),
		Comments:        nullInt(boolToInt(meta.Comments)),
		CreatedBy:       nullString(meta.CreatedBy.String()),
		UpdatedBy:       nullString(meta.UpdatedBy.String()),
		CreatedAt:       nullTime(&meta.CreatedAt),
		UpdatedAt:       nullTime(&meta.UpdatedAt),
	}

	_, err := s.queries.CreateMeta(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot create meta: %w", err)
	}

	return nil
}

func (s *service) UpdateMeta(ctx context.Context, meta *Meta) error {
	s.ensureQueries()

	params := sqlc.UpdateMetaParams{
		Summary:         nullString(meta.Summary),
		Excerpt:         nullString(meta.Excerpt),
		Description:     nullString(meta.Description),
		Keywords:        nullString(meta.Keywords),
		Robots:          nullString(meta.Robots),
		CanonicalUrl:    nullString(meta.CanonicalURL),
		Sitemap:         nullString(meta.Sitemap),
		TableOfContents: nullInt(boolToInt(meta.TableOfContents)),
		Share:           nullInt(boolToInt(meta.Share)),
		Comments:        nullInt(boolToInt(meta.Comments)),
		UpdatedBy:       nullString(meta.UpdatedBy.String()),
		UpdatedAt:       nullTime(&meta.UpdatedAt),
		ID:              meta.ID.String(),
	}

	_, err := s.queries.UpdateMeta(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot update meta: %w", err)
	}

	return nil
}

// --- Helper Functions ---

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func intToBool(i int64) bool {
	return i != 0
}

func nullString(s string) sql.NullString {
	if s == "" || s == "00000000-0000-0000-0000-000000000000" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullInt(i int64) sql.NullInt64 {
	return sql.NullInt64{Int64: i, Valid: true}
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func parseUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}

// --- Contributor Operations ---

func (s *service) CreateContributor(ctx context.Context, contributor *Contributor) error {
	s.ensureQueries()

	socialLinksJSON, err := json.Marshal(contributor.SocialLinks)
	if err != nil {
		return fmt.Errorf("cannot marshal social links: %w", err)
	}

	var profileID sql.NullString
	if contributor.ProfileID != nil {
		profileID = sql.NullString{String: contributor.ProfileID.String(), Valid: true}
	}

	params := sqlc.CreateContributorParams{
		ID:          contributor.ID.String(),
		ShortID:     contributor.ShortID,
		SiteID:      contributor.SiteID.String(),
		ProfileID:   profileID,
		Handle:      contributor.Handle,
		Name:        contributor.Name,
		Surname:     contributor.Surname,
		Bio:         contributor.Bio,
		SocialLinks: string(socialLinksJSON),
		Role:        contributor.Role,
		CreatedBy:   contributor.CreatedBy.String(),
		UpdatedBy:   contributor.UpdatedBy.String(),
		CreatedAt:   contributor.CreatedAt,
		UpdatedAt:   contributor.UpdatedAt,
	}

	_, err = s.queries.CreateContributor(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot create contributor: %w", err)
	}

	return nil
}

func (s *service) GetContributor(ctx context.Context, id uuid.UUID) (*Contributor, error) {
	s.ensureQueries()

	row, err := s.queries.GetContributor(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get contributor: %w", err)
	}

	return contributorFromSQLC(row)
}

func (s *service) GetContributorByHandle(ctx context.Context, siteID uuid.UUID, handle string) (*Contributor, error) {
	s.ensureQueries()

	row, err := s.queries.GetContributorByHandle(ctx, sqlc.GetContributorByHandleParams{
		SiteID: siteID.String(),
		Handle: handle,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get contributor by handle: %w", err)
	}

	return contributorFromSQLC(row)
}

func (s *service) GetContributors(ctx context.Context, siteID uuid.UUID) ([]*Contributor, error) {
	s.ensureQueries()

	rows, err := s.queries.ListContributorsWithProfile(ctx, siteID.String())
	if err != nil {
		return nil, fmt.Errorf("cannot list contributors: %w", err)
	}

	contributors := make([]*Contributor, 0, len(rows))
	for _, row := range rows {
		c, err := contributorWithProfileFromSQLC(row)
		if err != nil {
			return nil, err
		}
		contributors = append(contributors, c)
	}

	return contributors, nil
}

func (s *service) UpdateContributor(ctx context.Context, contributor *Contributor) error {
	s.ensureQueries()

	socialLinksJSON, err := json.Marshal(contributor.SocialLinks)
	if err != nil {
		return fmt.Errorf("cannot marshal social links: %w", err)
	}

	params := sqlc.UpdateContributorParams{
		ID:          contributor.ID.String(),
		Handle:      contributor.Handle,
		Name:        contributor.Name,
		Surname:     contributor.Surname,
		Bio:         contributor.Bio,
		SocialLinks: string(socialLinksJSON),
		Role:        contributor.Role,
		UpdatedBy:   contributor.UpdatedBy.String(),
		UpdatedAt:   contributor.UpdatedAt,
	}

	_, err = s.queries.UpdateContributor(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot update contributor: %w", err)
	}

	return nil
}

func (s *service) DeleteContributor(ctx context.Context, id uuid.UUID) error {
	s.ensureQueries()

	if err := s.queries.DeleteContributor(ctx, id.String()); err != nil {
		return fmt.Errorf("cannot delete contributor: %w", err)
	}

	return nil
}

func (s *service) SetContributorProfile(ctx context.Context, contributorID, profileID uuid.UUID, updatedBy string) error {
	s.ensureQueries()

	err := s.queries.SetContributorProfile(ctx, sqlc.SetContributorProfileParams{
		ProfileID: nullString(profileID.String()),
		UpdatedBy: updatedBy,
		UpdatedAt: time.Now(),
		ID:        contributorID.String(),
	})
	if err != nil {
		return fmt.Errorf("cannot set contributor profile: %w", err)
	}

	return nil
}

func contributorFromSQLC(row sqlc.Contributor) (*Contributor, error) {
	var socialLinks []SocialLink
	if row.SocialLinks != "" && row.SocialLinks != "[]" {
		if err := json.Unmarshal([]byte(row.SocialLinks), &socialLinks); err != nil {
			return nil, fmt.Errorf("cannot unmarshal social links: %w", err)
		}
	}

	var profileID *uuid.UUID
	if row.ProfileID.Valid {
		id := parseUUID(row.ProfileID.String)
		profileID = &id
	}

	return &Contributor{
		ID:          parseUUID(row.ID),
		SiteID:      parseUUID(row.SiteID),
		ProfileID:   profileID,
		ShortID:     row.ShortID,
		Handle:      row.Handle,
		Name:        row.Name,
		Surname:     row.Surname,
		Bio:         row.Bio,
		SocialLinks: socialLinks,
		Role:        row.Role,
		CreatedBy:   parseUUID(row.CreatedBy),
		UpdatedBy:   parseUUID(row.UpdatedBy),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func contributorWithProfileFromSQLC(row sqlc.ListContributorsWithProfileRow) (*Contributor, error) {
	var socialLinks []SocialLink
	if row.SocialLinks != "" && row.SocialLinks != "[]" {
		if err := json.Unmarshal([]byte(row.SocialLinks), &socialLinks); err != nil {
			return nil, fmt.Errorf("cannot unmarshal social links: %w", err)
		}
	}

	var profileID *uuid.UUID
	if row.ProfileID.Valid {
		id := parseUUID(row.ProfileID.String)
		profileID = &id
	}

	var photoPath string
	if row.ProfilePhotoPath.Valid {
		photoPath = row.ProfilePhotoPath.String
	}

	return &Contributor{
		ID:          parseUUID(row.ID),
		SiteID:      parseUUID(row.SiteID),
		ProfileID:   profileID,
		ShortID:     row.ShortID,
		Handle:      row.Handle,
		Name:        row.Name,
		Surname:     row.Surname,
		Bio:         row.Bio,
		SocialLinks: socialLinks,
		Role:        row.Role,
		PhotoPath:   photoPath,
		CreatedBy:   parseUUID(row.CreatedBy),
		UpdatedBy:   parseUUID(row.UpdatedBy),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func (s *service) GenerateHTMLForSite(ctx context.Context, siteSlug string) error {
	site, err := s.GetSiteBySlug(ctx, siteSlug)
	if err != nil {
		return fmt.Errorf("cannot get site: %w", err)
	}

	contents, err := s.GetAllContentWithMeta(ctx, site.ID)
	if err != nil {
		return fmt.Errorf("cannot get contents: %w", err)
	}

	for _, c := range contents {
		tags, err := s.GetTagsForContent(ctx, c.ID)
		if err == nil {
			c.Tags = tags
		}
		if c.ContributorID != nil {
			contributor, err := s.GetContributor(ctx, *c.ContributorID)
			if err == nil {
				c.Contributor = contributor
			}
		}
	}

	sections, err := s.GetSections(ctx, site.ID)
	if err != nil {
		return fmt.Errorf("cannot get sections: %w", err)
	}

	layouts, err := s.GetLayouts(ctx, site.ID)
	if err != nil {
		layouts = []*Layout{}
	}

	params, err := s.GetSettings(ctx, site.ID)
	if err != nil {
		params = []*Setting{}
	}

	contributors, err := s.GetContributors(ctx, site.ID)
	if err != nil {
		contributors = []*Contributor{}
	}

	userAuthors := s.BuildUserAuthorsMap(ctx, contents, contributors)

	_, err = s.htmlGen.GenerateHTML(ctx, site, contents, sections, layouts, params, contributors, userAuthors)
	if err != nil {
		return fmt.Errorf("cannot generate HTML: %w", err)
	}

	return nil
}

func (s *service) BuildUserAuthorsMap(ctx context.Context, contents []*Content, contributors []*Contributor) map[string]*Contributor {
	contributorHandles := make(map[string]bool)
	for _, c := range contributors {
		contributorHandles[c.Handle] = true
	}

	usernames := make(map[string]bool)
	for _, c := range contents {
		if c.AuthorUsername != "" && !contributorHandles[c.AuthorUsername] {
			usernames[c.AuthorUsername] = true
		}
	}

	result := make(map[string]*Contributor)
	for username := range usernames {
		row, err := s.queries.GetUserWithProfile(ctx, username)
		if err != nil {
			result[username] = &Contributor{
				Handle: username,
				Name:   username,
			}
			continue
		}

		author := &Contributor{
			Handle:  row.Name,
			Name:    row.ProfileName.String,
			Surname: row.ProfileSurname.String,
			Bio:     row.ProfileBio.String,
		}
		if author.Name == "" {
			author.Name = username
		}
		if row.ProfilePhotoPath.Valid && row.ProfilePhotoPath.String != "" {
			author.PhotoPath = row.ProfilePhotoPath.String
		}
		if row.ProfileSocialLinks.Valid && row.ProfileSocialLinks.String != "" {
			var links []SocialLink
			if err := json.Unmarshal([]byte(row.ProfileSocialLinks.String), &links); err == nil {
				author.SocialLinks = links
			}
		}
		result[username] = author
	}

	return result
}

func (s *service) buildImagesMeta(ctx context.Context, siteID uuid.UUID, body string) string {
	if body == "" {
		return ""
	}

	imgPathRegex := regexp.MustCompile(`/images/([^)"'\s]+\.(png|jpg|jpeg|gif|webp))`)
	matches := imgPathRegex.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return ""
	}

	meta := make(map[string]ImageMeta)
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		filePath := match[1]
		fullPath := "/images/" + filePath

		if seen[filePath] {
			continue
		}
		seen[filePath] = true

		img, err := s.queries.GetImageByPath(ctx, sqlc.GetImageByPathParams{
			SiteID:   siteID.String(),
			FilePath: filePath,
		})
		if err != nil {
			continue
		}

		if img.Attribution.Valid && img.Attribution.String != "" {
			meta[fullPath] = ImageMeta{
				Title:          img.Title.String,
				Alt:            img.AltText.String,
				Attribution:    img.Attribution.String,
				AttributionURL: img.AttributionUrl.String,
			}
		}
	}

	if len(meta) == 0 {
		return ""
	}

	jsonBytes, err := json.Marshal(meta)
	if err != nil {
		return ""
	}

	return string(jsonBytes)
}

// --- Import Operations ---

func (s *service) CreateImport(ctx context.Context, imp *Import) error {
	s.ensureQueries()

	params := sqlc.CreateImportParams{
		ID:        imp.ID.String(),
		ShortID:   imp.ShortID,
		FilePath:  imp.FilePath,
		FileHash:  nullString(imp.FileHash),
		FileMtime: nullTime(imp.FileMtime),
		SiteID:    imp.SiteID.String(),
		UserID:    imp.UserID.String(),
		Status:    imp.Status,
		CreatedAt: imp.CreatedAt,
		UpdatedAt: imp.UpdatedAt,
	}

	if imp.ContentID != nil {
		params.ContentID = nullString(imp.ContentID.String())
	}
	if imp.ImportedAt != nil {
		params.ImportedAt = nullTime(imp.ImportedAt)
	}

	_, err := s.queries.CreateImport(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot create import: %w", err)
	}

	return nil
}

func (s *service) GetImport(ctx context.Context, id uuid.UUID) (*Import, error) {
	s.ensureQueries()

	row, err := s.queries.GetImport(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get import: %w", err)
	}

	return importFromSQLC(row), nil
}

func (s *service) GetImportByFilePath(ctx context.Context, filePath string) (*Import, error) {
	s.ensureQueries()

	row, err := s.queries.GetImportByFilePath(ctx, filePath)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get import by file path: %w", err)
	}

	return importFromSQLC(row), nil
}

func (s *service) GetImportByContentID(ctx context.Context, contentID uuid.UUID) (*Import, error) {
	s.ensureQueries()

	row, err := s.queries.GetImportByContentID(ctx, nullString(contentID.String()))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get import by content ID: %w", err)
	}

	return importFromSQLC(row), nil
}

func (s *service) ListImports(ctx context.Context, siteID uuid.UUID) ([]*Import, error) {
	s.ensureQueries()

	rows, err := s.queries.ListImportsBySiteID(ctx, siteID.String())
	if err != nil {
		return nil, fmt.Errorf("cannot list imports: %w", err)
	}

	imports := make([]*Import, len(rows))
	for i, row := range rows {
		imports[i] = importWithContentFromSQLC(row)
	}

	return imports, nil
}

func (s *service) UpdateImport(ctx context.Context, imp *Import) error {
	s.ensureQueries()

	params := sqlc.UpdateImportParams{
		ID:        imp.ID.String(),
		FileHash:  nullString(imp.FileHash),
		FileMtime: nullTime(imp.FileMtime),
		Status:    imp.Status,
		UpdatedAt: imp.UpdatedAt,
	}

	if imp.ContentID != nil {
		params.ContentID = nullString(imp.ContentID.String())
	}
	if imp.ImportedAt != nil {
		params.ImportedAt = nullTime(imp.ImportedAt)
	}

	_, err := s.queries.UpdateImport(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot update import: %w", err)
	}

	return nil
}

func (s *service) UpdateImportStatus(ctx context.Context, id uuid.UUID, status string) error {
	s.ensureQueries()

	params := sqlc.UpdateImportStatusParams{
		ID:        id.String(),
		Status:    status,
		UpdatedAt: time.Now(),
	}

	_, err := s.queries.UpdateImportStatus(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot update import status: %w", err)
	}

	return nil
}

func (s *service) DeleteImport(ctx context.Context, id uuid.UUID) error {
	s.ensureQueries()

	err := s.queries.DeleteImport(ctx, id.String())
	if err != nil {
		return fmt.Errorf("cannot delete import: %w", err)
	}

	return nil
}

func (s *service) ScanImportDirectory(ctx context.Context, importPath string) ([]ImportFile, error) {
	scanner := NewImportScanner([]string{importPath})
	files, err := scanner.ScanFiles()
	if err != nil {
		return nil, fmt.Errorf("cannot scan import directory: %w", err)
	}

	return files, nil
}

func (s *service) ImportFile(ctx context.Context, siteID, userID uuid.UUID, file ImportFile, sectionID uuid.UUID) (*Content, *Import, error) {
	s.ensureQueries()

	existing, err := s.GetImportByFilePath(ctx, file.Path)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, nil, fmt.Errorf("cannot check existing import: %w", err)
	}
	if existing != nil && existing.ContentID != nil && existing.SiteID == siteID {
		return nil, nil, fmt.Errorf("file already imported")
	}

	resolvedSectionID := sectionID
	var typedFM *ImportFrontmatter

	if len(file.Frontmatter) > 0 {
		fm := ParseImportFrontmatter(file.Frontmatter)
		typedFM, _, _ = ParseTypedFrontmatter("---\n" + joinFrontmatter(file.Frontmatter) + "\n---\n")

		if sectionPath, ok := file.Frontmatter["section"]; ok && sectionPath != "" {
			if section, err := s.GetSectionByPath(ctx, siteID, sectionPath); err == nil {
				resolvedSectionID = section.ID
			}
		}

		content := NewContent(siteID, resolvedSectionID, file.Title, file.Body)
		content.UserID = userID
		content.CreatedBy = userID
		content.UpdatedBy = userID

		if typedFM != nil {
			if typedFM.CreatedAt != nil {
				content.CreatedAt = *typedFM.CreatedAt
			}
			if typedFM.PublishedAt != nil {
				content.PublishedAt = typedFM.PublishedAt
			}
		}

		if fm.ShortID != "" {
			content.ShortID = fm.ShortID
		}

		if fm.Summary != "" {
			content.Summary = fm.Summary
		}
		if fm.Kind != "" {
			content.Kind = fm.Kind
		}
		content.Draft = fm.Draft
		content.Featured = fm.Featured
		if fm.Series != "" {
			content.Series = fm.Series
			content.SeriesOrder = fm.SeriesOrder
		}
		if fm.Author != "" {
			content.AuthorUsername = fm.Author
		}
		if fm.Contributor != "" {
			content.ContributorHandle = fm.Contributor
			if contributor, err := s.GetContributorByHandle(ctx, siteID, fm.Contributor); err == nil {
				content.ContributorID = &contributor.ID
			}
		}

		if err := s.CreateContent(ctx, content); err != nil {
			return nil, nil, fmt.Errorf("cannot create content: %w", err)
		}
		if typedFM != nil && len(typedFM.Tags) > 0 {
			for _, tagName := range typedFM.Tags {
				_ = s.AddTagToContent(ctx, content.ID, tagName, siteID)
			}
		}

		if fm.Description != "" || fm.Robots != "" || fm.Keywords != "" ||
			fm.CanonicalURL != "" || fm.Sitemap != "" || fm.TableOfContents || fm.Comments || fm.Share {
			meta := NewMeta(siteID, content.ID)
			meta.Description = fm.Description
			meta.Robots = fm.Robots
			meta.Keywords = fm.Keywords
			meta.CanonicalURL = fm.CanonicalURL
			meta.Sitemap = fm.Sitemap
			meta.TableOfContents = fm.TableOfContents
			meta.Comments = fm.Comments
			meta.Share = fm.Share
			meta.CreatedBy = userID
			meta.UpdatedBy = userID
			_ = s.CreateMeta(ctx, meta)
		}

		if fm.Image != "" {
			imgPath := strings.TrimPrefix(fm.Image, "/images/")
			if img, err := s.GetImageByPath(ctx, siteID, imgPath); err == nil {
				_ = s.LinkImageToContent(ctx, content.ID, img.ID, true)
			}
		}

		imagePaths := ExtractImagePaths(file.Body)
		for _, imgPath := range imagePaths {
			if img, err := s.GetImageByPath(ctx, siteID, imgPath); err == nil {
				_ = s.LinkImageToContent(ctx, content.ID, img.ID, false)
			}
		}

		now := time.Now()
		imp := NewImport(siteID, userID, file.Path)
		imp.FileHash = file.Hash
		imp.FileMtime = &file.Mtime
		imp.ContentID = &content.ID
		imp.Status = ImportStatusImported
		imp.ImportedAt = &now

		if err := s.CreateImport(ctx, imp); err != nil {
			_ = s.DeleteContent(ctx, content.ID)
			return nil, nil, fmt.Errorf("cannot create import: %w", err)
		}

		return content, imp, nil
	}

	content := NewContent(siteID, resolvedSectionID, file.Title, file.Body)
	content.UserID = userID
	content.CreatedBy = userID
	content.UpdatedBy = userID

	if err := s.CreateContent(ctx, content); err != nil {
		return nil, nil, fmt.Errorf("cannot create content: %w", err)
	}

	imagePaths := ExtractImagePaths(file.Body)
	for _, imgPath := range imagePaths {
		if img, err := s.GetImageByPath(ctx, siteID, imgPath); err == nil {
			_ = s.LinkImageToContent(ctx, content.ID, img.ID, false)
		}
	}

	now := time.Now()
	imp := NewImport(siteID, userID, file.Path)
	imp.FileHash = file.Hash
	imp.FileMtime = &file.Mtime
	imp.ContentID = &content.ID
	imp.Status = ImportStatusImported
	imp.ImportedAt = &now

	if err := s.CreateImport(ctx, imp); err != nil {
		_ = s.DeleteContent(ctx, content.ID)
		return nil, nil, fmt.Errorf("cannot create import: %w", err)
	}

	return content, imp, nil
}

func joinFrontmatter(fm map[string]string) string {
	var lines []string
	for k, v := range fm {
		if strings.ContainsAny(v, ":\"'#[]{}") || strings.HasPrefix(v, " ") || strings.HasSuffix(v, " ") {
			v = "'" + strings.ReplaceAll(v, "'", "''") + "'"
		}
		lines = append(lines, k+": "+v)
	}
	return strings.Join(lines, "\n")
}

func (s *service) ReimportFile(ctx context.Context, importID uuid.UUID, force bool) (*Content, error) {
	s.ensureQueries()

	// Get the import
	imp, err := s.GetImport(ctx, importID)
	if err != nil {
		return nil, fmt.Errorf("cannot get import: %w", err)
	}

	if imp.ContentID == nil {
		return nil, fmt.Errorf("import has no associated content")
	}

	// Get the content
	content, err := s.GetContent(ctx, *imp.ContentID)
	if err != nil {
		return nil, fmt.Errorf("cannot get content: %w", err)
	}

	// Re-scan the file
	scanner := NewImportScanner([]string{})
	fileInfo, err := scanner.parseFile(imp.FilePath, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot re-parse file: %w", err)
	}

	// Check for conflicts
	if imp.ContentUpdatedAt != nil && imp.FileMtime != nil {
		if imp.ContentUpdatedAt.After(*imp.FileMtime) && !force {
			return nil, fmt.Errorf("conflict detected: content was modified in web UI after last import, use force=true to override")
		}
	}

	// Update content
	content.Heading = fileInfo.Title
	content.Body = fileInfo.Body
	content.UpdatedAt = time.Now()

	// Apply frontmatter updates
	if len(fileInfo.Frontmatter) > 0 {
		fm := ParseImportFrontmatter(fileInfo.Frontmatter)
		if fm.Summary != "" {
			content.Summary = fm.Summary
		}
		if fm.Kind != "" {
			content.Kind = fm.Kind
		}
		content.Draft = fm.Draft
		content.Featured = fm.Featured
		if fm.Series != "" {
			content.Series = fm.Series
			content.SeriesOrder = fm.SeriesOrder
		}
	}

	if err := s.UpdateContent(ctx, content); err != nil {
		return nil, fmt.Errorf("cannot update content: %w", err)
	}

	// Update import
	now := time.Now()
	imp.FileHash = fileInfo.Hash
	imp.FileMtime = &fileInfo.Mtime
	imp.Status = ImportStatusImported
	imp.ImportedAt = &now
	imp.UpdatedAt = now

	if err := s.UpdateImport(ctx, imp); err != nil {
		return nil, fmt.Errorf("cannot update import: %w", err)
	}

	return content, nil
}
