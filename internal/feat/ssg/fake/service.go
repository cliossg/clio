package fake

import (
	"context"

	"github.com/cliossg/clio/internal/feat/ssg"
	"github.com/google/uuid"
)

type Service struct {
	Sites        []*ssg.Site
	Contents     map[uuid.UUID][]*ssg.Content
	Sections     map[uuid.UUID][]*ssg.Section
	Layouts      map[uuid.UUID][]*ssg.Layout
	Settings     map[uuid.UUID][]*ssg.Setting
	Contributors map[uuid.UUID][]*ssg.Contributor

	UpdateSiteCalls []*ssg.Site
	UpdateSiteErr   error

	ListSitesErr           error
	GetAllContentErr       error
	GetSectionsErr         error
	GetSettingByRefKeyFunc func(siteID uuid.UUID, refKey string) (*ssg.Setting, error)
}

func NewService() *Service {
	return &Service{
		Contents:     make(map[uuid.UUID][]*ssg.Content),
		Sections:     make(map[uuid.UUID][]*ssg.Section),
		Layouts:      make(map[uuid.UUID][]*ssg.Layout),
		Settings:     make(map[uuid.UUID][]*ssg.Setting),
		Contributors: make(map[uuid.UUID][]*ssg.Contributor),
	}
}

func (s *Service) Start(_ context.Context) error { return nil }
func (s *Service) Stop(_ context.Context) error  { return nil }

func (s *Service) ListSites(_ context.Context) ([]*ssg.Site, error) {
	return s.Sites, s.ListSitesErr
}

func (s *Service) GetSettingByRefKey(_ context.Context, siteID uuid.UUID, refKey string) (*ssg.Setting, error) {
	if s.GetSettingByRefKeyFunc != nil {
		return s.GetSettingByRefKeyFunc(siteID, refKey)
	}
	for _, st := range s.Settings[siteID] {
		if st.RefKey == refKey {
			return st, nil
		}
	}
	return nil, nil
}

func (s *Service) GetAllContentWithMeta(_ context.Context, siteID uuid.UUID) ([]*ssg.Content, error) {
	return s.Contents[siteID], s.GetAllContentErr
}

func (s *Service) GetSections(_ context.Context, siteID uuid.UUID) ([]*ssg.Section, error) {
	return s.Sections[siteID], s.GetSectionsErr
}

func (s *Service) GetLayouts(_ context.Context, siteID uuid.UUID) ([]*ssg.Layout, error) {
	return s.Layouts[siteID], nil
}

func (s *Service) GetSettings(_ context.Context, siteID uuid.UUID) ([]*ssg.Setting, error) {
	return s.Settings[siteID], nil
}

func (s *Service) GetContributors(_ context.Context, siteID uuid.UUID) ([]*ssg.Contributor, error) {
	return s.Contributors[siteID], nil
}

func (s *Service) BuildUserAuthorsMap(_ context.Context, _ []*ssg.Content, _ []*ssg.Contributor) map[string]*ssg.Contributor {
	return make(map[string]*ssg.Contributor)
}

func (s *Service) UpdateSite(_ context.Context, site *ssg.Site) error {
	s.UpdateSiteCalls = append(s.UpdateSiteCalls, site)
	return s.UpdateSiteErr
}

// Unused methods required by Service interface.

func (s *Service) CreateSite(_ context.Context, _ *ssg.Site) error                    { return nil }
func (s *Service) GetSite(_ context.Context, _ uuid.UUID) (*ssg.Site, error)          { return nil, nil }
func (s *Service) GetSiteBySlug(_ context.Context, _ string) (*ssg.Site, error)       { return nil, nil }
func (s *Service) DeleteSite(_ context.Context, _ uuid.UUID) error                    { return nil }
func (s *Service) CreateContent(_ context.Context, _ *ssg.Content) error              { return nil }
func (s *Service) GetContent(_ context.Context, _ uuid.UUID) (*ssg.Content, error)    { return nil, nil }
func (s *Service) GetContentWithMeta(_ context.Context, _ uuid.UUID) (*ssg.Content, error) {
	return nil, nil
}
func (s *Service) GetContentWithPagination(_ context.Context, _ uuid.UUID, _, _ int, _ string) ([]*ssg.Content, int, error) {
	return nil, 0, nil
}
func (s *Service) UpdateContent(_ context.Context, _ *ssg.Content) error { return nil }
func (s *Service) DeleteContent(_ context.Context, _ uuid.UUID) error    { return nil }
func (s *Service) CreateSection(_ context.Context, _ *ssg.Section) error { return nil }
func (s *Service) GetSection(_ context.Context, _ uuid.UUID) (*ssg.Section, error) {
	return nil, nil
}
func (s *Service) GetSectionByPath(_ context.Context, _ uuid.UUID, _ string) (*ssg.Section, error) {
	return nil, nil
}
func (s *Service) UpdateSection(_ context.Context, _ *ssg.Section) error { return nil }
func (s *Service) DeleteSection(_ context.Context, _ uuid.UUID) error    { return nil }
func (s *Service) CreateLayout(_ context.Context, _ *ssg.Layout) error   { return nil }
func (s *Service) GetLayout(_ context.Context, _ uuid.UUID) (*ssg.Layout, error) {
	return nil, nil
}
func (s *Service) UpdateLayout(_ context.Context, _ *ssg.Layout) error { return nil }
func (s *Service) DeleteLayout(_ context.Context, _ uuid.UUID) error   { return nil }
func (s *Service) CreateTag(_ context.Context, _ *ssg.Tag) error       { return nil }
func (s *Service) GetTag(_ context.Context, _ uuid.UUID) (*ssg.Tag, error) {
	return nil, nil
}
func (s *Service) GetTagByName(_ context.Context, _ uuid.UUID, _ string) (*ssg.Tag, error) {
	return nil, nil
}
func (s *Service) GetTags(_ context.Context, _ uuid.UUID) ([]*ssg.Tag, error) { return nil, nil }
func (s *Service) UpdateTag(_ context.Context, _ *ssg.Tag) error              { return nil }
func (s *Service) DeleteTag(_ context.Context, _ uuid.UUID) error             { return nil }
func (s *Service) AddTagToContent(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID) error {
	return nil
}
func (s *Service) AddTagToContentByID(_ context.Context, _, _ uuid.UUID) error    { return nil }
func (s *Service) RemoveTagFromContent(_ context.Context, _, _ uuid.UUID) error   { return nil }
func (s *Service) RemoveAllTagsFromContent(_ context.Context, _ uuid.UUID) error  { return nil }
func (s *Service) GetTagsForContent(_ context.Context, _ uuid.UUID) ([]*ssg.Tag, error) {
	return nil, nil
}
func (s *Service) CreateSetting(_ context.Context, _ *ssg.Setting) error { return nil }
func (s *Service) GetSetting(_ context.Context, _ uuid.UUID) (*ssg.Setting, error) {
	return nil, nil
}
func (s *Service) GetSettingByName(_ context.Context, _ uuid.UUID, _ string) (*ssg.Setting, error) {
	return nil, nil
}
func (s *Service) UpdateSetting(_ context.Context, _ *ssg.Setting) error { return nil }
func (s *Service) DeleteSetting(_ context.Context, _ uuid.UUID) error    { return nil }
func (s *Service) CreateImage(_ context.Context, _ *ssg.Image) error     { return nil }
func (s *Service) GetImage(_ context.Context, _ uuid.UUID) (*ssg.Image, error) {
	return nil, nil
}
func (s *Service) GetImages(_ context.Context, _ uuid.UUID) ([]*ssg.Image, error) {
	return nil, nil
}
func (s *Service) GetImageByPath(_ context.Context, _ uuid.UUID, _ string) (*ssg.Image, error) {
	return nil, nil
}
func (s *Service) GetContentImagesWithDetails(_ context.Context, _ uuid.UUID) ([]*ssg.ContentImageWithDetails, error) {
	return nil, nil
}
func (s *Service) GetAllContentImages(_ context.Context, _ uuid.UUID) (map[string][]ssg.MetaContentImage, error) {
	return nil, nil
}
func (s *Service) GetContentImageDetails(_ context.Context, _ uuid.UUID) (*ssg.ContentImageDetails, error) {
	return nil, nil
}
func (s *Service) LinkImageToContent(_ context.Context, _, _ uuid.UUID, _ bool) error   { return nil }
func (s *Service) UnlinkImageFromContent(_ context.Context, _ uuid.UUID) error          { return nil }
func (s *Service) UnlinkHeaderImageFromContent(_ context.Context, _ uuid.UUID) error    { return nil }
func (s *Service) GetSectionImagesWithDetails(_ context.Context, _ uuid.UUID) ([]*ssg.SectionImageWithDetails, error) {
	return nil, nil
}
func (s *Service) GetSectionImageDetails(_ context.Context, _ uuid.UUID) (*ssg.SectionImageDetails, error) {
	return nil, nil
}
func (s *Service) LinkImageToSection(_ context.Context, _, _ uuid.UUID, _ bool) error { return nil }
func (s *Service) UnlinkImageFromSection(_ context.Context, _ uuid.UUID) error        { return nil }
func (s *Service) UpdateImage(_ context.Context, _ *ssg.Image) error                  { return nil }
func (s *Service) DeleteImage(_ context.Context, _ uuid.UUID) error                   { return nil }
func (s *Service) GetMetaByContentID(_ context.Context, _ uuid.UUID) (*ssg.Meta, error) {
	return nil, nil
}
func (s *Service) CreateMeta(_ context.Context, _ *ssg.Meta) error          { return nil }
func (s *Service) UpdateMeta(_ context.Context, _ *ssg.Meta) error          { return nil }
func (s *Service) CreateContributor(_ context.Context, _ *ssg.Contributor) error { return nil }
func (s *Service) GetContributor(_ context.Context, _ uuid.UUID) (*ssg.Contributor, error) {
	return nil, nil
}
func (s *Service) GetContributorByHandle(_ context.Context, _ uuid.UUID, _ string) (*ssg.Contributor, error) {
	return nil, nil
}
func (s *Service) UpdateContributor(_ context.Context, _ *ssg.Contributor) error       { return nil }
func (s *Service) DeleteContributor(_ context.Context, _ uuid.UUID) error              { return nil }
func (s *Service) SetContributorProfile(_ context.Context, _, _ uuid.UUID, _ string) error {
	return nil
}
func (s *Service) GenerateHTMLForSite(_ context.Context, _ string) error { return nil }
func (s *Service) CreateImport(_ context.Context, _ *ssg.Import) error   { return nil }
func (s *Service) GetImport(_ context.Context, _ uuid.UUID) (*ssg.Import, error) {
	return nil, nil
}
func (s *Service) GetImportByFilePath(_ context.Context, _ string) (*ssg.Import, error) {
	return nil, nil
}
func (s *Service) GetImportByContentID(_ context.Context, _ uuid.UUID) (*ssg.Import, error) {
	return nil, nil
}
func (s *Service) ListImports(_ context.Context, _ uuid.UUID) ([]*ssg.Import, error) {
	return nil, nil
}
func (s *Service) UpdateImport(_ context.Context, _ *ssg.Import) error            { return nil }
func (s *Service) UpdateImportStatus(_ context.Context, _ uuid.UUID, _ string) error { return nil }
func (s *Service) DeleteImport(_ context.Context, _ uuid.UUID) error              { return nil }
func (s *Service) ScanImportDirectory(_ context.Context, _ string) ([]ssg.ImportFile, error) {
	return nil, nil
}
func (s *Service) ImportFile(_ context.Context, _, _ uuid.UUID, _ ssg.ImportFile, _ uuid.UUID) (*ssg.Content, *ssg.Import, error) {
	return nil, nil, nil
}
func (s *Service) ReimportFile(_ context.Context, _ uuid.UUID, _ bool) (*ssg.Content, error) {
	return nil, nil
}
