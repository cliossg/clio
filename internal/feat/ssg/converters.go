package ssg

import (
	"time"

	"github.com/cliossg/clio/internal/db/sqlc"
)

// Site converters

func siteFromSQLC(s sqlc.Site) *Site {
	site := &Site{
		ID:        parseUUID(s.ID),
		ShortID:   s.ShortID,
		Name:      s.Name,
		Slug:      s.Slug,
		Active:    s.Active == 1,
		CreatedBy: parseUUID(s.CreatedBy),
		UpdatedBy: parseUUID(s.UpdatedBy),
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
	if s.DefaultLayoutID.Valid {
		site.DefaultLayoutID = parseUUID(s.DefaultLayoutID.String)
	}
	if s.DefaultLayoutName.Valid {
		site.DefaultLayoutName = s.DefaultLayoutName.String
	}
	return site
}

// Content converters

func contentFromSQLC(c sqlc.Content) *Content {
	content := &Content{
		ID:            parseUUID(c.ID),
		SiteID:        parseUUID(c.SiteID),
		ShortID:       c.ShortID.String,
		Heading:       c.Heading,
		Summary:       c.Summary.String,
		Body:          c.Body.String,
		Draft:         intToBool(c.Draft.Int64),
		Featured:      intToBool(c.Featured.Int64),
		Series:        c.Series.String,
		Kind:          c.Kind.String,
		HeroTitleDark: intToBool(c.HeroTitleDark.Int64),
	}

	if c.UserID.Valid {
		content.UserID = parseUUID(c.UserID.String)
	}
	if c.SectionID.Valid {
		content.SectionID = parseUUID(c.SectionID.String)
	}
	if c.SeriesOrder.Valid {
		content.SeriesOrder = int(c.SeriesOrder.Int64)
	}
	if c.PublishedAt.Valid {
		content.PublishedAt = &c.PublishedAt.Time
	}
	if c.CreatedBy.Valid {
		content.CreatedBy = parseUUID(c.CreatedBy.String)
	}
	if c.UpdatedBy.Valid {
		content.UpdatedBy = parseUUID(c.UpdatedBy.String)
	}
	if c.CreatedAt.Valid {
		content.CreatedAt = c.CreatedAt.Time
	}
	if c.UpdatedAt.Valid {
		content.UpdatedAt = c.UpdatedAt.Time
	}
	if c.ContributorID.Valid {
		id := parseUUID(c.ContributorID.String)
		content.ContributorID = &id
	}
	if c.ImagesMeta.Valid {
		content.ImagesMeta = c.ImagesMeta.String
	}

	return content
}

func contentWithMetaFromSQLC(row sqlc.GetContentWithMetaRow) *Content {
	content := &Content{
		ID:            parseUUID(row.ID),
		SiteID:        parseUUID(row.SiteID),
		ShortID:       row.ShortID.String,
		Heading:       row.Heading,
		Summary:       row.Summary.String,
		Body:          row.Body.String,
		Draft:         intToBool(row.Draft.Int64),
		Featured:      intToBool(row.Featured.Int64),
		Series:        row.Series.String,
		Kind:          row.Kind.String,
		HeroTitleDark: intToBool(row.HeroTitleDark.Int64),
	}

	if row.UserID.Valid {
		content.UserID = parseUUID(row.UserID.String)
	}
	if row.SectionID.Valid {
		content.SectionID = parseUUID(row.SectionID.String)
	}
	if row.SeriesOrder.Valid {
		content.SeriesOrder = int(row.SeriesOrder.Int64)
	}
	if row.PublishedAt.Valid {
		content.PublishedAt = &row.PublishedAt.Time
	}
	if row.CreatedBy.Valid {
		content.CreatedBy = parseUUID(row.CreatedBy.String)
	}
	if row.UpdatedBy.Valid {
		content.UpdatedBy = parseUUID(row.UpdatedBy.String)
	}
	if row.CreatedAt.Valid {
		content.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		content.UpdatedAt = row.UpdatedAt.Time
	}

	// Joined fields
	if row.SectionPath.Valid {
		content.SectionPath = row.SectionPath.String
	}
	if row.SectionName.Valid {
		content.SectionName = row.SectionName.String
	}

	// Meta fields
	if row.MetaSummary.Valid || row.MetaDescription.Valid || row.MetaKeywords.Valid {
		content.Meta = &Meta{
			Summary:     row.MetaSummary.String,
			Description: row.MetaDescription.String,
			Keywords:    row.MetaKeywords.String,
		}
	}
	if row.ContributorID.Valid {
		id := parseUUID(row.ContributorID.String)
		content.ContributorID = &id
	}
	if row.ImagesMeta.Valid {
		content.ImagesMeta = row.ImagesMeta.String
	}
	content.ContributorHandle = row.ContributorHandle
	content.AuthorUsername = row.AuthorUsername

	return content
}

func contentWithMetaFromSQLCAll(row sqlc.GetAllContentWithMetaRow) *Content {
	content := &Content{
		ID:            parseUUID(row.ID),
		SiteID:        parseUUID(row.SiteID),
		ShortID:       row.ShortID.String,
		Heading:       row.Heading,
		Summary:       row.Summary.String,
		Body:          row.Body.String,
		Draft:         intToBool(row.Draft.Int64),
		Featured:      intToBool(row.Featured.Int64),
		Series:        row.Series.String,
		Kind:          row.Kind.String,
		HeroTitleDark: intToBool(row.HeroTitleDark.Int64),
	}

	if row.UserID.Valid {
		content.UserID = parseUUID(row.UserID.String)
	}
	if row.SectionID.Valid {
		content.SectionID = parseUUID(row.SectionID.String)
	}
	if row.SeriesOrder.Valid {
		content.SeriesOrder = int(row.SeriesOrder.Int64)
	}
	if row.PublishedAt.Valid {
		content.PublishedAt = &row.PublishedAt.Time
	}
	if row.CreatedBy.Valid {
		content.CreatedBy = parseUUID(row.CreatedBy.String)
	}
	if row.UpdatedBy.Valid {
		content.UpdatedBy = parseUUID(row.UpdatedBy.String)
	}
	if row.CreatedAt.Valid {
		content.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		content.UpdatedAt = row.UpdatedAt.Time
	}

	// Joined fields
	if row.SectionPath.Valid {
		content.SectionPath = row.SectionPath.String
	}
	if row.SectionName.Valid {
		content.SectionName = row.SectionName.String
	}

	// Meta fields
	if row.MetaSummary.Valid || row.MetaDescription.Valid || row.MetaKeywords.Valid {
		content.Meta = &Meta{
			Summary:     row.MetaSummary.String,
			Description: row.MetaDescription.String,
			Keywords:    row.MetaKeywords.String,
		}
	}
	if row.ContributorID.Valid {
		id := parseUUID(row.ContributorID.String)
		content.ContributorID = &id
	}
	content.ContributorHandle = row.ContributorHandle
	content.AuthorUsername = row.AuthorUsername

	if row.HeaderImagePath.Valid {
		content.HeaderImageURL = "/images/" + row.HeaderImagePath.String
	}
	if row.HeaderImageAlt.Valid {
		content.HeaderImageAlt = row.HeaderImageAlt.String
	}
	if row.HeaderImageCaption.Valid {
		content.HeaderImageCaption = row.HeaderImageCaption.String
	}
	if row.HeaderImageAttribution.Valid {
		content.HeaderImageAttribution = row.HeaderImageAttribution.String
	}
	if row.HeaderImageAttributionUrl.Valid {
		content.HeaderImageAttributionURL = row.HeaderImageAttributionUrl.String
	}
	if row.ImagesMeta.Valid {
		content.ImagesMeta = row.ImagesMeta.String
	}

	return content
}

// Section converters

func sectionFromSQLC(s sqlc.Section) *Section {
	section := &Section{
		ID:     parseUUID(s.ID),
		SiteID: parseUUID(s.SiteID),
		Name:   s.Name,
	}

	if s.ShortID.Valid {
		section.ShortID = s.ShortID.String
	}
	if s.Description.Valid {
		section.Description = s.Description.String
	}
	if s.Path.Valid {
		section.Path = s.Path.String
	}
	if s.LayoutID.Valid {
		section.LayoutID = parseUUID(s.LayoutID.String)
	}
	if s.LayoutName.Valid {
		section.LayoutName = s.LayoutName.String
	}
	if s.CreatedBy.Valid {
		section.CreatedBy = parseUUID(s.CreatedBy.String)
	}
	if s.UpdatedBy.Valid {
		section.UpdatedBy = parseUUID(s.UpdatedBy.String)
	}
	if s.CreatedAt.Valid {
		section.CreatedAt = s.CreatedAt.Time
	}
	if s.UpdatedAt.Valid {
		section.UpdatedAt = s.UpdatedAt.Time
	}

	return section
}

// Layout converters

func layoutFromSQLC(l sqlc.Layout) *Layout {
	layout := &Layout{
		ID:     parseUUID(l.ID),
		SiteID: parseUUID(l.SiteID),
		Name:   l.Name,
	}

	if l.ShortID.Valid {
		layout.ShortID = l.ShortID.String
	}
	if l.Description.Valid {
		layout.Description = l.Description.String
	}
	if l.Code.Valid {
		layout.Code = l.Code.String
	}
	if l.HeaderImageID.Valid {
		layout.HeaderImageID = parseUUID(l.HeaderImageID.String)
	}
	if l.CreatedBy.Valid {
		layout.CreatedBy = parseUUID(l.CreatedBy.String)
	}
	if l.UpdatedBy.Valid {
		layout.UpdatedBy = parseUUID(l.UpdatedBy.String)
	}
	if l.CreatedAt.Valid {
		layout.CreatedAt = l.CreatedAt.Time
	}
	if l.UpdatedAt.Valid {
		layout.UpdatedAt = l.UpdatedAt.Time
	}

	return layout
}

// Tag converters

func tagFromSQLC(t sqlc.Tag) *Tag {
	tag := &Tag{
		ID:     parseUUID(t.ID),
		SiteID: parseUUID(t.SiteID),
		Name:   t.Name,
		Slug:   t.Slug,
	}

	if t.ShortID.Valid {
		tag.ShortID = t.ShortID.String
	}
	if t.CreatedBy.Valid {
		tag.CreatedBy = parseUUID(t.CreatedBy.String)
	}
	if t.UpdatedBy.Valid {
		tag.UpdatedBy = parseUUID(t.UpdatedBy.String)
	}
	if t.CreatedAt.Valid {
		tag.CreatedAt = t.CreatedAt.Time
	}
	if t.UpdatedAt.Valid {
		tag.UpdatedAt = t.UpdatedAt.Time
	}

	return tag
}

// Setting converters

func settingFromSQLC(s sqlc.Setting) *Setting {
	setting := &Setting{
		ID:     parseUUID(s.ID),
		SiteID: parseUUID(s.SiteID),
		Name:   s.Name,
	}

	if s.ShortID.Valid {
		setting.ShortID = s.ShortID.String
	}
	if s.Description.Valid {
		setting.Description = s.Description.String
	}
	if s.Value.Valid {
		setting.Value = s.Value.String
	}
	if s.RefKey.Valid {
		setting.RefKey = s.RefKey.String
	}
	if s.System.Valid {
		setting.System = intToBool(s.System.Int64)
	}
	if s.CreatedBy.Valid {
		setting.CreatedBy = parseUUID(s.CreatedBy.String)
	}
	if s.UpdatedBy.Valid {
		setting.UpdatedBy = parseUUID(s.UpdatedBy.String)
	}
	if s.CreatedAt.Valid {
		setting.CreatedAt = s.CreatedAt.Time
	}
	if s.UpdatedAt.Valid {
		setting.UpdatedAt = s.UpdatedAt.Time
	}
	if s.Category.Valid {
		setting.Category = s.Category.String
	}
	if s.Position.Valid {
		setting.Position = int(s.Position.Int64)
	}

	return setting
}

// Image converters

func imageFromSQLC(i sqlc.Image) *Image {
	image := &Image{
		ID:       parseUUID(i.ID),
		SiteID:   parseUUID(i.SiteID),
		FileName: i.FileName,
		FilePath: i.FilePath,
	}

	if i.ShortID.Valid {
		image.ShortID = i.ShortID.String
	}
	if i.AltText.Valid {
		image.AltText = i.AltText.String
	}
	if i.Title.Valid {
		image.Title = i.Title.String
	}
	if i.Attribution.Valid {
		image.Attribution = i.Attribution.String
	}
	if i.AttributionUrl.Valid {
		image.AttributionURL = i.AttributionUrl.String
	}
	if i.Width.Valid {
		image.Width = int(i.Width.Int64)
	}
	if i.Height.Valid {
		image.Height = int(i.Height.Int64)
	}
	if i.CreatedBy.Valid {
		image.CreatedBy = parseUUID(i.CreatedBy.String)
	}
	if i.UpdatedBy.Valid {
		image.UpdatedBy = parseUUID(i.UpdatedBy.String)
	}
	if i.CreatedAt.Valid {
		image.CreatedAt = i.CreatedAt.Time
	}
	if i.UpdatedAt.Valid {
		image.UpdatedAt = i.UpdatedAt.Time
	}

	return image
}

// Meta converter

func metaFromSQLC(m sqlc.Meta) *Meta {
	meta := &Meta{
		ID:        parseUUID(m.ID),
		SiteID:    parseUUID(m.SiteID),
		ContentID: parseUUID(m.ContentID),
	}

	if m.ShortID.Valid {
		meta.ShortID = m.ShortID.String
	}
	if m.Summary.Valid {
		meta.Summary = m.Summary.String
	}
	if m.Excerpt.Valid {
		meta.Excerpt = m.Excerpt.String
	}
	if m.Description.Valid {
		meta.Description = m.Description.String
	}
	if m.Keywords.Valid {
		meta.Keywords = m.Keywords.String
	}
	if m.Robots.Valid {
		meta.Robots = m.Robots.String
	}
	if m.CanonicalUrl.Valid {
		meta.CanonicalURL = m.CanonicalUrl.String
	}
	if m.Sitemap.Valid {
		meta.Sitemap = m.Sitemap.String
	}
	if m.TableOfContents.Valid {
		meta.TableOfContents = m.TableOfContents.Int64 == 1
	}
	if m.Share.Valid {
		meta.Share = m.Share.Int64 == 1
	}
	if m.Comments.Valid {
		meta.Comments = m.Comments.Int64 == 1
	}
	if m.CreatedBy.Valid {
		meta.CreatedBy = parseUUID(m.CreatedBy.String)
	}
	if m.UpdatedBy.Valid {
		meta.UpdatedBy = parseUUID(m.UpdatedBy.String)
	}
	if m.CreatedAt.Valid {
		meta.CreatedAt = m.CreatedAt.Time
	}
	if m.UpdatedAt.Valid {
		meta.UpdatedAt = m.UpdatedAt.Time
	}

	return meta
}

// Time helper
func timeFromNull(t interface{ Time() (time.Time, bool) }) time.Time {
	if t, ok := t.Time(); ok {
		return t
	}
	return time.Time{}
}
