package ssg

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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

	// Param operations
	CreateParam(ctx context.Context, param *Param) error
	GetParam(ctx context.Context, id uuid.UUID) (*Param, error)
	GetParamByName(ctx context.Context, siteID uuid.UUID, name string) (*Param, error)
	GetParams(ctx context.Context, siteID uuid.UUID) ([]*Param, error)
	UpdateParam(ctx context.Context, param *Param) error
	DeleteParam(ctx context.Context, id uuid.UUID) error

	// Image operations
	CreateImage(ctx context.Context, image *Image) error
	GetImage(ctx context.Context, id uuid.UUID) (*Image, error)
	GetImages(ctx context.Context, siteID uuid.UUID) ([]*Image, error)
	GetContentImagesWithDetails(ctx context.Context, contentID uuid.UUID) ([]*ContentImageWithDetails, error)
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
	GetContributors(ctx context.Context, siteID uuid.UUID) ([]*Contributor, error)
	UpdateContributor(ctx context.Context, contributor *Contributor) error
	DeleteContributor(ctx context.Context, id uuid.UUID) error
}

// DBProvider provides access to the database.
type DBProvider interface {
	GetDB() *sql.DB
}

type service struct {
	dbProvider DBProvider
	queries    *sqlc.Queries
	cfg        *config.Config
	log        logger.Logger
}

// NewService creates a new SSG service.
func NewService(dbProvider DBProvider, cfg *config.Config, log logger.Logger) Service {
	return &service{
		dbProvider: dbProvider,
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
		Mode:      site.Mode,
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
		Name:      site.Name,
		Slug:      site.Slug,
		Mode:      site.Mode,
		Active:    boolToInt(site.Active),
		UpdatedBy: site.UpdatedBy.String(),
		UpdatedAt: site.UpdatedAt,
		ID:        site.ID.String(),
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

	params := sqlc.CreateContentParams{
		ID:            content.ID.String(),
		SiteID:        content.SiteID.String(),
		UserID:        nullString(content.UserID.String()),
		ShortID:       nullString(content.ShortID),
		SectionID:     nullString(content.SectionID.String()),
		ContributorID: contributorID,
		Kind:          nullString(content.Kind),
		Heading:       content.Heading,
		Summary:       nullString(content.Summary),
		Body:          nullString(content.Body),
		Draft:         nullInt(boolToInt(content.Draft)),
		Featured:      nullInt(boolToInt(content.Featured)),
		Series:        nullString(content.Series),
		SeriesOrder:   nullInt(int64(content.SeriesOrder)),
		PublishedAt:   nullTime(content.PublishedAt),
		CreatedBy:     nullString(content.CreatedBy.String()),
		UpdatedBy:     nullString(content.UpdatedBy.String()),
		CreatedAt:     nullTime(&content.CreatedAt),
		UpdatedAt:     nullTime(&content.UpdatedAt),
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

	var contributorID sql.NullString
	if content.ContributorID != nil {
		contributorID = nullString(content.ContributorID.String())
	}

	params := sqlc.UpdateContentParams{
		SectionID:     nullString(content.SectionID.String()),
		ContributorID: contributorID,
		Kind:          nullString(content.Kind),
		Heading:       content.Heading,
		Summary:       nullString(content.Summary),
		Body:          nullString(content.Body),
		Draft:         nullInt(boolToInt(content.Draft)),
		Featured:      nullInt(boolToInt(content.Featured)),
		Series:        nullString(content.Series),
		SeriesOrder:   nullInt(int64(content.SeriesOrder)),
		PublishedAt:   nullTime(content.PublishedAt),
		UpdatedBy:     nullString(content.UpdatedBy.String()),
		UpdatedAt:     nullTime(&content.UpdatedAt),
		ID:            content.ID.String(),
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
		ID:          section.ID.String(),
		SiteID:      section.SiteID.String(),
		ShortID:     nullString(section.ShortID),
		Name:        section.Name,
		Description: nullString(section.Description),
		Path:        nullString(section.Path),
		LayoutID:    nullString(section.LayoutID.String()),
		LayoutName:  nullString(section.LayoutName),
		CreatedBy:   nullString(section.CreatedBy.String()),
		UpdatedBy:   nullString(section.UpdatedBy.String()),
		CreatedAt:   nullTime(&section.CreatedAt),
		UpdatedAt:   nullTime(&section.UpdatedAt),
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

func (s *service) GetSections(ctx context.Context, siteID uuid.UUID) ([]*Section, error) {
	s.ensureQueries()

	sqlcSections, err := s.queries.GetSectionsBySiteID(ctx, siteID.String())
	if err != nil {
		return nil, fmt.Errorf("cannot get sections: %w", err)
	}

	sections := make([]*Section, len(sqlcSections))
	for i, sqlcSection := range sqlcSections {
		sections[i] = sectionFromSQLC(sqlcSection)
	}

	return sections, nil
}

func (s *service) UpdateSection(ctx context.Context, section *Section) error {
	s.ensureQueries()

	params := sqlc.UpdateSectionParams{
		Name:        section.Name,
		Description: nullString(section.Description),
		Path:        nullString(section.Path),
		LayoutID:    nullString(section.LayoutID.String()),
		LayoutName:  nullString(section.LayoutName),
		UpdatedBy:   nullString(section.UpdatedBy.String()),
		UpdatedAt:   nullTime(&section.UpdatedAt),
		ID:          section.ID.String(),
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
		ID:            layout.ID.String(),
		SiteID:        layout.SiteID.String(),
		ShortID:       nullString(layout.ShortID),
		Name:          layout.Name,
		Description:   nullString(layout.Description),
		Code:          nullString(layout.Code),
		HeaderImageID: nullString(layout.HeaderImageID.String()),
		CreatedBy:     nullString(layout.CreatedBy.String()),
		UpdatedBy:     nullString(layout.UpdatedBy.String()),
		CreatedAt:     nullTime(&layout.CreatedAt),
		UpdatedAt:     nullTime(&layout.UpdatedAt),
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
		Name:          layout.Name,
		Description:   nullString(layout.Description),
		Code:          nullString(layout.Code),
		HeaderImageID: nullString(layout.HeaderImageID.String()),
		UpdatedBy:     nullString(layout.UpdatedBy.String()),
		UpdatedAt:     nullTime(&layout.UpdatedAt),
		ID:            layout.ID.String(),
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

// --- Param Operations ---

func (s *service) CreateParam(ctx context.Context, param *Param) error {
	s.ensureQueries()

	params := sqlc.CreateParamParams{
		ID:          param.ID.String(),
		SiteID:      param.SiteID.String(),
		ShortID:     nullString(param.ShortID),
		Name:        param.Name,
		Description: nullString(param.Description),
		Value:       nullString(param.Value),
		RefKey:      nullString(param.RefKey),
		System:      nullInt(boolToInt(param.System)),
		CreatedBy:   nullString(param.CreatedBy.String()),
		UpdatedBy:   nullString(param.UpdatedBy.String()),
		CreatedAt:   nullTime(&param.CreatedAt),
		UpdatedAt:   nullTime(&param.UpdatedAt),
	}

	_, err := s.queries.CreateParam(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot create param: %w", err)
	}

	return nil
}

func (s *service) GetParam(ctx context.Context, id uuid.UUID) (*Param, error) {
	s.ensureQueries()

	sqlcParam, err := s.queries.GetParam(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get param: %w", err)
	}

	return paramFromSQLC(sqlcParam), nil
}

func (s *service) GetParamByName(ctx context.Context, siteID uuid.UUID, name string) (*Param, error) {
	s.ensureQueries()

	sqlcParam, err := s.queries.GetParamByName(ctx, sqlc.GetParamByNameParams{
		SiteID: siteID.String(),
		Name:   name,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get param by name: %w", err)
	}

	return paramFromSQLC(sqlcParam), nil
}

func (s *service) GetParams(ctx context.Context, siteID uuid.UUID) ([]*Param, error) {
	s.ensureQueries()

	sqlcParams, err := s.queries.GetParamsBySiteID(ctx, siteID.String())
	if err != nil {
		return nil, fmt.Errorf("cannot get params: %w", err)
	}

	params := make([]*Param, len(sqlcParams))
	for i, sqlcParam := range sqlcParams {
		params[i] = paramFromSQLC(sqlcParam)
	}

	return params, nil
}

func (s *service) UpdateParam(ctx context.Context, param *Param) error {
	s.ensureQueries()

	params := sqlc.UpdateParamParams{
		Name:        param.Name,
		Description: nullString(param.Description),
		Value:       nullString(param.Value),
		RefKey:      nullString(param.RefKey),
		System:      nullInt(boolToInt(param.System)),
		UpdatedBy:   nullString(param.UpdatedBy.String()),
		UpdatedAt:   nullTime(&param.UpdatedAt),
		ID:          param.ID.String(),
	}

	_, err := s.queries.UpdateParam(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot update param: %w", err)
	}

	return nil
}

func (s *service) DeleteParam(ctx context.Context, id uuid.UUID) error {
	s.ensureQueries()

	err := s.queries.DeleteParam(ctx, id.String())
	if err != nil {
		return fmt.Errorf("cannot delete param: %w", err)
	}

	return nil
}

// --- Image Operations ---

func (s *service) CreateImage(ctx context.Context, image *Image) error {
	s.ensureQueries()

	params := sqlc.CreateImageParams{
		ID:        image.ID.String(),
		SiteID:    image.SiteID.String(),
		ShortID:   nullString(image.ShortID),
		FileName:  image.FileName,
		FilePath:  image.FilePath,
		AltText:   nullString(image.AltText),
		Title:     nullString(image.Title),
		Width:     nullInt(int64(image.Width)),
		Height:    nullInt(int64(image.Height)),
		CreatedBy: nullString(image.CreatedBy.String()),
		UpdatedBy: nullString(image.UpdatedBy.String()),
		CreatedAt: nullTime(&image.CreatedAt),
		UpdatedAt: nullTime(&image.UpdatedAt),
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

func (s *service) GetContentImagesWithDetails(ctx context.Context, contentID uuid.UUID) ([]*ContentImageWithDetails, error) {
	s.ensureQueries()

	rows, err := s.queries.GetContentImagesWithDetails(ctx, contentID.String())
	if err != nil {
		return nil, fmt.Errorf("cannot get content images: %w", err)
	}

	images := make([]*ContentImageWithDetails, len(rows))
	for i, row := range rows {
		images[i] = &ContentImageWithDetails{
			ContentImageID: uuid.MustParse(row.ContentImageID),
			ContentID:      uuid.MustParse(row.ContentID),
			IsHeader:       row.IsHeader.Int64 == 1,
			IsFeatured:     row.IsFeatured.Int64 == 1,
			OrderNum:       int(row.OrderNum.Int64),
			ID:             uuid.MustParse(row.ID),
			SiteID:         uuid.MustParse(row.SiteID),
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
		ContentImageID: uuid.MustParse(row.ContentImageID),
		ImageID:        uuid.MustParse(row.ImageID),
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
			SectionImageID: uuid.MustParse(row.SectionImageID),
			SectionID:      uuid.MustParse(row.SectionID),
			IsHeader:       row.IsHeader.Int64 == 1,
			IsFeatured:     row.IsFeatured.Int64 == 1,
			OrderNum:       int(row.OrderNum.Int64),
			ID:             uuid.MustParse(row.ID),
			SiteID:         uuid.MustParse(row.SiteID),
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
		SectionImageID: uuid.MustParse(row.SectionImageID),
		ImageID:        uuid.MustParse(row.ImageID),
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

	params := sqlc.CreateContributorParams{
		ID:          contributor.ID.String(),
		ShortID:     contributor.ShortID,
		SiteID:      contributor.SiteID.String(),
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

func (s *service) GetContributors(ctx context.Context, siteID uuid.UUID) ([]*Contributor, error) {
	s.ensureQueries()

	rows, err := s.queries.ListContributorsBySiteID(ctx, siteID.String())
	if err != nil {
		return nil, fmt.Errorf("cannot list contributors: %w", err)
	}

	contributors := make([]*Contributor, 0, len(rows))
	for _, row := range rows {
		c, err := contributorFromSQLC(row)
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

func contributorFromSQLC(row sqlc.Contributor) (*Contributor, error) {
	var socialLinks []SocialLink
	if row.SocialLinks != "" && row.SocialLinks != "[]" {
		if err := json.Unmarshal([]byte(row.SocialLinks), &socialLinks); err != nil {
			return nil, fmt.Errorf("cannot unmarshal social links: %w", err)
		}
	}

	return &Contributor{
		ID:          uuid.MustParse(row.ID),
		SiteID:      uuid.MustParse(row.SiteID),
		ShortID:     row.ShortID,
		Handle:      row.Handle,
		Name:        row.Name,
		Surname:     row.Surname,
		Bio:         row.Bio,
		SocialLinks: socialLinks,
		Role:        row.Role,
		CreatedBy:   uuid.MustParse(row.CreatedBy),
		UpdatedBy:   uuid.MustParse(row.UpdatedBy),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}
