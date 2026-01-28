package ssg

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cliossg/clio/internal/feat/profile"
	"github.com/cliossg/clio/pkg/cl/config"
	"github.com/cliossg/clio/pkg/cl/git"
	"github.com/cliossg/clio/pkg/cl/logger"
	"github.com/cliossg/clio/pkg/cl/middleware"
	"github.com/cliossg/clio/pkg/cl/render"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ProfileService interface {
	CreateProfile(ctx context.Context, slug, name, surname, bio, socialLinks, photoPath, createdBy string) (*profile.Profile, error)
	GetProfile(ctx context.Context, id uuid.UUID) (*profile.Profile, error)
	UpdateProfile(ctx context.Context, p *profile.Profile) error
	DeleteProfile(ctx context.Context, id uuid.UUID) error
}

type Handler struct {
	service        Service
	profileService ProfileService
	workspace      *Workspace
	generator      *Generator
	htmlGen        *HTMLGenerator
	publisher      *Publisher
	siteCtxMw      func(http.Handler) http.Handler
	sessionMw      func(http.Handler) http.Handler
	userNameFn     func(context.Context) string
	userRolesFn    func(context.Context) string
	templatesFS    embed.FS
	ssgAssetsFS    embed.FS
	cfg            *config.Config
	log            logger.Logger
}

func NewHandler(service Service, profileService ProfileService, siteCtxMw, sessionMw func(http.Handler) http.Handler, userNameFn func(context.Context) string, userRolesFn func(context.Context) string, templatesFS, ssgAssetsFS embed.FS, cfg *config.Config, log logger.Logger) *Handler {
	workspace := NewWorkspace(cfg.SSG.SitesBasePath)
	gitClient := git.NewClient(log)
	return &Handler{
		service:        service,
		profileService: profileService,
		workspace:      workspace,
		generator:      NewGenerator(workspace),
		htmlGen:        NewHTMLGenerator(workspace, ssgAssetsFS),
		publisher:      NewPublisher(workspace, gitClient),
		siteCtxMw:      siteCtxMw,
		sessionMw:      sessionMw,
		userNameFn:     userNameFn,
		userRolesFn:    userRolesFn,
		templatesFS:    templatesFS,
		ssgAssetsFS:    ssgAssetsFS,
		cfg:            cfg,
		log:            log,
	}
}

// Start initializes templates and other resources.
func (h *Handler) Start(ctx context.Context) error {
	h.log.Info("SSG handler started")
	return nil
}

type tagifyEntry struct {
	Value string `json:"value"`
	ID    string `json:"id"`
}

func (h *Handler) processTagifyTags(ctx context.Context, siteID, contentID uuid.UUID, tagsStr string) {
	if tagsStr == "" {
		h.log.Debugf("processTagifyTags: empty tags string")
		return
	}

	h.log.Debugf("processTagifyTags: received tags string: %s", tagsStr)

	var entries []tagifyEntry
	jsonStr := "[" + tagsStr + "]"
	if err := json.Unmarshal([]byte(jsonStr), &entries); err != nil {
		h.log.Errorf("Failed to parse tagify tags: %v (input: %s)", err, tagsStr)
		return
	}

	h.log.Debugf("processTagifyTags: parsed %d entries", len(entries))

	for _, entry := range entries {
		if entry.Value == "" {
			continue
		}

		var tagID uuid.UUID
		if entry.ID != "" {
			if parsed, err := uuid.Parse(entry.ID); err == nil {
				tagID = parsed
			}
		}

		if tagID == uuid.Nil {
			existing, err := h.service.GetTagByName(ctx, siteID, entry.Value)
			if err == nil && existing != nil {
				tagID = existing.ID
			} else {
				slug := strings.ToLower(strings.ReplaceAll(entry.Value, " ", "-"))
				tag := &Tag{
					ID:     uuid.New(),
					SiteID: siteID,
					Name:   entry.Value,
					Slug:   slug,
				}
				if err := h.service.CreateTag(ctx, tag); err != nil {
					h.log.Errorf("Failed to create tag %s: %v", entry.Value, err)
					continue
				}
				tagID = tag.ID
			}
		}

		_ = h.service.AddTagToContentByID(ctx, contentID, tagID)
	}
}

func (h *Handler) requireEditor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		roles := ""
		if h.userRolesFn != nil {
			roles = h.userRolesFn(r.Context())
		}
		isEditor := false
		for _, role := range strings.Split(roles, ",") {
			r := strings.TrimSpace(role)
			if r == "admin" || r == "editor" {
				isEditor = true
				break
			}
		}
		if !isEditor {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		roles := ""
		if h.userRolesFn != nil {
			roles = h.userRolesFn(r.Context())
		}
		isAdmin := false
		for _, role := range strings.Split(roles, ",") {
			if strings.TrimSpace(role) == "admin" {
				isAdmin = true
				break
			}
		}
		if !isAdmin {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RegisterRoutes registers SSG routes.
func (h *Handler) RegisterRoutes(r chi.Router) {
	h.log.Info("Registering SSG routes")

	// Serve workspace images (public, no auth required for preview)
	r.Get("/ssg/workspace/{slug}/images/{filename}", h.HandleServeWorkspaceImage)

	r.Group(func(r chi.Router) {
		r.Use(h.sessionMw)

		// Sites - read
		r.Get("/ssg/list-sites", h.HandleListSites)
		r.Get("/ssg/get-site", h.HandleShowSite)

		// Sites - write (admin only)
		r.Group(func(r chi.Router) {
			r.Use(h.requireAdmin)
			r.Get("/ssg/new-site", h.HandleNewSite)
			r.Post("/ssg/create-site", h.HandleCreateSite)
			r.Get("/ssg/edit-site", h.HandleEditSite)
			r.Post("/ssg/update-site", h.HandleUpdateSite)
			r.Post("/ssg/delete-site", h.HandleDeleteSite)
		})

		// Routes that need site context middleware
		r.Group(func(r chi.Router) {
			r.Use(h.siteCtxMw)

			// Read-only routes (viewer+)
			r.Get("/ssg/list-contents", h.HandleListContents)
			r.Get("/ssg/get-content", h.HandleShowContent)
			r.Get("/ssg/list-tags", h.HandleListTags)
			r.Get("/ssg/get-tag", h.HandleShowTag)
			r.Get("/ssg/list-images", h.HandleListImages)
			r.Get("/ssg/get-image", h.HandleShowImage)

			// Editor routes (editor+)
			r.Group(func(r chi.Router) {
				r.Use(h.requireEditor)

				// Contents
				r.Get("/ssg/new-content", h.HandleNewContent)
				r.Post("/ssg/create-content", h.HandleCreateContent)
				r.Get("/ssg/edit-content", h.HandleEditContent)
				r.Post("/ssg/update-content", h.HandleUpdateContent)
				r.Post("/ssg/autosave-content", h.HandleAutosaveContent)
				r.Post("/ssg/delete-content", h.HandleDeleteContent)

				// Tags
				r.Get("/ssg/new-tag", h.HandleNewTag)
				r.Post("/ssg/create-tag", h.HandleCreateTag)
				r.Get("/ssg/edit-tag", h.HandleEditTag)
				r.Post("/ssg/update-tag", h.HandleUpdateTag)
				r.Post("/ssg/delete-tag", h.HandleDeleteTag)

				// Images
				r.Get("/ssg/new-image", h.HandleNewImage)
				r.Post("/ssg/create-image", h.HandleCreateImage)
				r.Get("/ssg/edit-image", h.HandleEditImage)
				r.Post("/ssg/update-image", h.HandleUpdateImage)
				r.Post("/ssg/delete-image", h.HandleDeleteImage)

				// Content Images
				r.Post("/ssg/upload-content-image", h.HandleUploadContentImage)
				r.Post("/ssg/delete-content-image", h.HandleDeleteContentImage)
				r.Post("/ssg/remove-header-image", h.HandleRemoveHeaderImage)

				// Meta
				r.Post("/ssg/update-meta", h.HandleUpdateMeta)

				// Generation
				r.Post("/ssg/backup-markdown", h.HandleBackupMarkdown)
				r.Post("/ssg/generate-html", h.HandleGenerateHTML)
				r.Post("/ssg/publish", h.HandlePublish)
			})

			// Admin-only routes
			r.Group(func(r chi.Router) {
				r.Use(h.requireAdmin)

				// Params
				r.Get("/ssg/list-params", h.HandleListParams)
				r.Get("/ssg/get-param", h.HandleShowParam)
				r.Get("/ssg/new-param", h.HandleNewParam)
				r.Post("/ssg/create-param", h.HandleCreateParam)
				r.Get("/ssg/edit-param", h.HandleEditParam)
				r.Post("/ssg/update-param", h.HandleUpdateParam)
				r.Post("/ssg/delete-param", h.HandleDeleteParam)

				// Sections
				r.Get("/ssg/list-sections", h.HandleListSections)
				r.Get("/ssg/new-section", h.HandleNewSection)
				r.Post("/ssg/create-section", h.HandleCreateSection)
				r.Get("/ssg/get-section", h.HandleShowSection)
				r.Get("/ssg/edit-section", h.HandleEditSection)
				r.Post("/ssg/update-section", h.HandleUpdateSection)
				r.Post("/ssg/delete-section", h.HandleDeleteSection)

				// Layouts
				r.Get("/ssg/list-layouts", h.HandleListLayouts)
				r.Get("/ssg/new-layout", h.HandleNewLayout)
				r.Post("/ssg/create-layout", h.HandleCreateLayout)
				r.Get("/ssg/get-layout", h.HandleShowLayout)
				r.Get("/ssg/edit-layout", h.HandleEditLayout)
				r.Post("/ssg/update-layout", h.HandleUpdateLayout)
				r.Post("/ssg/delete-layout", h.HandleDeleteLayout)

				// Section Images
				r.Post("/ssg/upload-section-image", h.HandleUploadSectionImage)
				r.Post("/ssg/delete-section-image", h.HandleDeleteSectionImage)

				// Contributors
				r.Get("/ssg/list-contributors", h.HandleListContributors)
				r.Get("/ssg/new-contributor", h.HandleNewContributor)
				r.Post("/ssg/create-contributor", h.HandleCreateContributor)
				r.Get("/ssg/get-contributor", h.HandleShowContributor)
				r.Get("/ssg/edit-contributor", h.HandleEditContributor)
				r.Post("/ssg/update-contributor", h.HandleUpdateContributor)
				r.Post("/ssg/delete-contributor", h.HandleDeleteContributor)
				r.Get("/ssg/edit-contributor-profile", h.HandleEditContributorProfile)
				r.Post("/ssg/update-contributor-profile", h.HandleUpdateContributorProfile)
				r.Post("/ssg/upload-contributor-photo", h.HandleUploadContributorPhoto)
				r.Post("/ssg/remove-contributor-photo", h.HandleRemoveContributorPhoto)
			})
		})
	})
}

// PageData holds common page data for templates.
type PageData struct {
	Title            string
	Template         string
	HideNav          bool
	AuthPage         bool
	CurrentUserName  string
	CurrentUserRoles string
	Site             *Site
	Sites           []*Site
	Section         *Section
	Sections        []*Section
	Content         *Content
	Contents        []*Content
	Layout          *Layout
	Layouts         []*Layout
	Tag             *Tag
	Tags            []*Tag
	Param           *Param
	Params          []*Param
	Image           *Image
	Images          []*Image
	Contributor          *Contributor
	Contributors         []*Contributor
	ContributorProfile   *profile.Profile
	ProfileSocialLinks   map[string]string
	HeaderImage     *ContentImageWithDetails
	ContentImages   []*ContentImageWithDetails
	SectionImages   []*SectionImageWithDetails
	SectionHeader   *SectionImageWithDetails
	Meta            *Meta
	Error           string
	Success         string
	CSRFToken       string
	CurrentPage     int
	TotalPages      int
	HasPrev         bool
	HasNext         bool
	Search          string
}

func (h *Handler) render(w http.ResponseWriter, r *http.Request, templateName string, data PageData) {
	funcMap := render.MergeFuncMaps(render.FuncMap(), template.FuncMap{
		"add":      func(a, b int) int { return a + b },
		"subtract": func(a, b int) int { return a - b },
		"multiply": func(a, b int) int { return a * b },
		"deref": func(p *uuid.UUID) uuid.UUID {
			if p == nil {
				return uuid.Nil
			}
			return *p
		},
		"hasRole": func(roles, role string) bool {
			for _, r := range strings.Split(roles, ",") {
				if strings.TrimSpace(r) == role {
					return true
				}
			}
			return false
		},
	})

	if data.CurrentUserName == "" && h.userNameFn != nil {
		data.CurrentUserName = h.userNameFn(r.Context())
	}
	if data.CurrentUserRoles == "" && h.userRolesFn != nil {
		data.CurrentUserRoles = h.userRolesFn(r.Context())
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(h.templatesFS,
		"assets/templates/base.html",
		"assets/templates/"+templateName+".html",
	)
	if err != nil {
		h.log.Errorf("Template parse error for %s: %v", templateName, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		h.log.Errorf("Template execute error for %s: %v", templateName, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) renderError(w http.ResponseWriter, r *http.Request, status int, message string) {
	h.log.Errorf("HTTP %d: %s", status, message)
	w.WriteHeader(status)
	h.render(w, r, "error", PageData{
		Title: "Error",
		Error: message,
	})
}

func (h *Handler) siteRedirect(w http.ResponseWriter, r *http.Request, path string) {
	site := getSiteFromContext(r.Context())
	if site != nil {
		if strings.Contains(path, "?") {
			path += "&site_id=" + site.ID.String()
		} else {
			path += "?site_id=" + site.ID.String()
		}
	}
	http.Redirect(w, r, path, http.StatusSeeOther)
}

// --- Site Handlers ---

func (h *Handler) HandleListSites(w http.ResponseWriter, r *http.Request) {
	sites, err := h.service.ListSites(r.Context())
	if err != nil {
		h.log.Errorf("Cannot list sites: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Cannot load sites")
		return
	}

	if len(sites) == 0 {
		http.Redirect(w, r, "/ssg/new-site", http.StatusSeeOther)
		return
	}

	h.render(w, r, "ssg/sites/list", PageData{
		Title: "Sites",
		Sites: sites,
	})
}

func (h *Handler) HandleNewSite(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "ssg/sites/new", PageData{
		Title: "New Site",
	})
}

func (h *Handler) HandleCreateSite(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	name := r.FormValue("name")
	slug := r.FormValue("slug")
	mode := r.FormValue("mode")
	if mode == "" {
		mode = "blog"
	}

	site := NewSite(name, slug, mode)

	// Get user ID from context
	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			site.CreatedBy = userID
			site.UpdatedBy = userID
		}
	}

	if err := h.service.CreateSite(r.Context(), site); err != nil {
		h.log.Errorf("Cannot create site: %v", err)
		h.render(w, r, "ssg/sites/new", PageData{
			Title: "New Site",
			Site:  site,
			Error: "Cannot create site",
		})
		return
	}

	// Create site directories
	if err := h.workspace.CreateSiteDirectories(site.Slug); err != nil {
		h.log.Errorf("Cannot create site directories: %v", err)
		// Rollback: delete site from DB
		_ = h.service.DeleteSite(r.Context(), site.ID)
		h.render(w, r, "ssg/sites/new", PageData{
			Title: "New Site",
			Site:  site,
			Error: "Cannot create site directories",
		})
		return
	}

	// Create root section
	rootSection := NewSection(site.ID, "/ (root)", "Root section for top-level content", "")
	if err := h.service.CreateSection(r.Context(), rootSection); err != nil {
		h.log.Errorf("Cannot create root section: %v", err)
	}

	h.log.Infof("Created site %s with directories", site.Slug)
	http.Redirect(w, r, "/ssg/list-sites", http.StatusSeeOther)
}

func (h *Handler) HandleShowSite(w http.ResponseWriter, r *http.Request) {
	siteID, err := uuid.Parse(r.URL.Query().Get("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid site ID")
		return
	}

	site, err := h.service.GetSite(r.Context(), siteID)
	if err != nil {
		h.log.Errorf("Cannot get site: %v", err)
		h.renderError(w, r, http.StatusNotFound, "Site not found")
		return
	}

	data := PageData{
		Title: site.Name,
		Site:  site,
	}

	switch r.URL.Query().Get("success") {
	case "markdown":
		data.Success = "Markdown files generated successfully"
	case "backup":
		data.Success = "Markdown backed up to git repository"
	case "backup_no_changes":
		data.Success = "No changes to backup"
	case "html":
		data.Success = "HTML site generated successfully"
	case "publish":
		data.Success = "Site published successfully"
	case "publish_no_changes":
		data.Success = "No changes to publish"
	}

	switch r.URL.Query().Get("error") {
	case "backup_failed":
		data.Error = "Failed to backup markdown to git repository"
	case "publish_not_configured":
		data.Error = "Publish repository not configured"
	case "publish_failed":
		data.Error = "Failed to publish site to git repository"
	}

	h.render(w, r, "ssg/sites/show", data)
}

func (h *Handler) HandleEditSite(w http.ResponseWriter, r *http.Request) {
	siteID, err := uuid.Parse(r.URL.Query().Get("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid site ID")
		return
	}

	site, err := h.service.GetSite(r.Context(), siteID)
	if err != nil {
		h.log.Errorf("Cannot get site: %v", err)
		h.renderError(w, r, http.StatusNotFound, "Site not found")
		return
	}

	h.render(w, r, "ssg/sites/edit", PageData{
		Title: "Edit " + site.Name,
		Site:  site,
	})
}

func (h *Handler) HandleUpdateSite(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	siteID, err := uuid.Parse(r.FormValue("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid site ID")
		return
	}

	site, err := h.service.GetSite(r.Context(), siteID)
	if err != nil {
		h.log.Errorf("Cannot get site: %v", err)
		h.renderError(w, r, http.StatusNotFound, "Site not found")
		return
	}

	site.Name = r.FormValue("name")
	site.Slug = r.FormValue("slug")
	site.Mode = r.FormValue("mode")
	site.Active = r.FormValue("active") == "on"

	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			site.UpdatedBy = userID
		}
	}

	if err := h.service.UpdateSite(r.Context(), site); err != nil {
		h.log.Errorf("Cannot update site: %v", err)
		h.render(w, r, "ssg/sites/edit", PageData{
			Title: "Edit " + site.Name,
			Site:  site,
			Error: "Cannot update site",
		})
		return
	}

	http.Redirect(w, r, "/ssg/get-site?id="+site.ID.String(), http.StatusSeeOther)
}

func (h *Handler) HandleDeleteSite(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	siteID, err := uuid.Parse(r.FormValue("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid site ID")
		return
	}

	// Get site first to get the slug for directory deletion
	site, err := h.service.GetSite(r.Context(), siteID)
	if err != nil {
		h.log.Errorf("Cannot get site for deletion: %v", err)
		h.renderError(w, r, http.StatusNotFound, "Site not found")
		return
	}

	if err := h.service.DeleteSite(r.Context(), siteID); err != nil {
		h.log.Errorf("Cannot delete site: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Cannot delete site")
		return
	}

	// Delete site directories (errors logged but not fatal)
	if err := h.workspace.DeleteSiteDirectories(site.Slug); err != nil {
		h.log.Errorf("Cannot delete site directories: %v", err)
	}

	h.log.Infof("Deleted site %s", site.Slug)
	http.Redirect(w, r, "/ssg/list-sites", http.StatusSeeOther)
}

// --- Section Handlers ---

func (h *Handler) HandleListSections(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	sections, err := h.service.GetSections(r.Context(), site.ID)
	if err != nil {
		h.log.Errorf("Cannot list sections: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Cannot load sections")
		return
	}

	h.render(w, r, "ssg/sections/list", PageData{
		Title:    "Sections",
		Site:     site,
		Sections: sections,
	})
}

func (h *Handler) HandleNewSection(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	layouts, _ := h.service.GetLayouts(r.Context(), site.ID)

	h.render(w, r, "ssg/sections/new", PageData{
		Title:   "New Section",
		Site:    site,
		Layouts: layouts,
	})
}

func (h *Handler) HandleCreateSection(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	section := NewSection(site.ID, r.FormValue("name"), r.FormValue("description"), r.FormValue("path"))

	if layoutID := r.FormValue("layout_id"); layoutID != "" {
		if id, err := uuid.Parse(layoutID); err == nil {
			section.LayoutID = id
		}
	}

	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			section.CreatedBy = userID
			section.UpdatedBy = userID
		}
	}

	if err := h.service.CreateSection(r.Context(), section); err != nil {
		h.log.Errorf("Cannot create section: %v", err)
		layouts, _ := h.service.GetLayouts(r.Context(), site.ID)
		h.render(w, r, "ssg/sections/new", PageData{
			Title:   "New Section",
			Site:    site,
			Section: section,
			Layouts: layouts,
			Error:   "Cannot create section",
		})
		return
	}

	h.siteRedirect(w, r, "/ssg/get-section?id="+section.ID.String())
}

func (h *Handler) HandleShowSection(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	sectionID, err := uuid.Parse(r.URL.Query().Get("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid section ID")
		return
	}

	section, err := h.service.GetSection(r.Context(), sectionID)
	if err != nil {
		h.log.Errorf("Cannot get section: %v", err)
		h.renderError(w, r, http.StatusNotFound, "Section not found")
		return
	}

	h.render(w, r, "ssg/sections/show", PageData{
		Title:   section.Name,
		Site:    site,
		Section: section,
	})
}

func (h *Handler) HandleEditSection(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	sectionID, err := uuid.Parse(r.URL.Query().Get("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid section ID")
		return
	}

	section, err := h.service.GetSection(r.Context(), sectionID)
	if err != nil {
		h.log.Errorf("Cannot get section: %v", err)
		h.renderError(w, r, http.StatusNotFound, "Section not found")
		return
	}

	layouts, _ := h.service.GetLayouts(r.Context(), site.ID)

	allImages, _ := h.service.GetSectionImagesWithDetails(r.Context(), sectionID)
	var sectionHeader *SectionImageWithDetails
	var sectionImages []*SectionImageWithDetails
	for _, img := range allImages {
		if img.IsHeader {
			sectionHeader = img
		} else {
			sectionImages = append(sectionImages, img)
		}
	}

	h.render(w, r, "ssg/sections/edit", PageData{
		Title:         "Edit " + section.Name,
		Site:          site,
		Section:       section,
		Layouts:       layouts,
		SectionHeader: sectionHeader,
		SectionImages: sectionImages,
	})
}

func (h *Handler) HandleUpdateSection(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	sectionID, err := uuid.Parse(r.FormValue("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid section ID")
		return
	}

	section, err := h.service.GetSection(r.Context(), sectionID)
	if err != nil {
		h.log.Errorf("Cannot get section: %v", err)
		h.renderError(w, r, http.StatusNotFound, "Section not found")
		return
	}

	section.Name = r.FormValue("name")
	section.Description = r.FormValue("description")
	section.Path = normalizePath(r.FormValue("path"))

	if layoutID := r.FormValue("layout_id"); layoutID != "" {
		if id, err := uuid.Parse(layoutID); err == nil {
			section.LayoutID = id
		}
	}

	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			section.UpdatedBy = userID
		}
	}

	if err := h.service.UpdateSection(r.Context(), section); err != nil {
		h.log.Errorf("Cannot update section: %v", err)
		layouts, _ := h.service.GetLayouts(r.Context(), site.ID)
		h.render(w, r, "ssg/sections/edit", PageData{
			Title:   "Edit " + section.Name,
			Site:    site,
			Section: section,
			Layouts: layouts,
			Error:   "Cannot update section",
		})
		return
	}

	h.siteRedirect(w, r, "/ssg/get-section?id="+section.ID.String())
}

func (h *Handler) HandleDeleteSection(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	sectionID, err := uuid.Parse(r.FormValue("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid section ID")
		return
	}

	if err := h.service.DeleteSection(r.Context(), sectionID); err != nil {
		h.log.Errorf("Cannot delete section: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Cannot delete section")
		return
	}

	h.siteRedirect(w, r, "/ssg/list-sections")
}

// --- Content Handlers ---

func (h *Handler) HandleListContents(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	limit := 25
	offset := (page - 1) * limit
	search := r.URL.Query().Get("q")

	contents, total, err := h.service.GetContentWithPagination(r.Context(), site.ID, offset, limit, search)
	if err != nil {
		h.log.Errorf("Cannot list contents: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Cannot load contents")
		return
	}

	totalPages := (total + limit - 1) / limit

	h.render(w, r, "ssg/contents/list", PageData{
		Title:       "Contents",
		Site:        site,
		Contents:    contents,
		CurrentPage: page,
		TotalPages:  totalPages,
		HasPrev:     page > 1,
		HasNext:     page < totalPages,
		Search:      search,
	})
}

func (h *Handler) HandleNewContent(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	sections, _ := h.service.GetSections(r.Context(), site.ID)
	tags, _ := h.service.GetTags(r.Context(), site.ID)
	contributors, _ := h.service.GetContributors(r.Context(), site.ID)

	h.render(w, r, "ssg/contents/new", PageData{
		Title:        "New Content",
		Site:         site,
		Sections:     sections,
		Tags:         tags,
		Contributors: contributors,
	})
}

func (h *Handler) HandleCreateContent(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	var sectionID uuid.UUID
	if sid := r.FormValue("section_id"); sid != "" {
		if id, err := uuid.Parse(sid); err == nil {
			sectionID = id
		}
	}

	content := NewContent(site.ID, sectionID, r.FormValue("heading"), r.FormValue("body"))
	content.Summary = r.FormValue("summary")
	content.Kind = r.FormValue("kind")
	if content.Kind == "" {
		content.Kind = "post"
	}
	content.Draft = r.FormValue("draft") == "on"
	content.Featured = r.FormValue("featured") == "on"
	content.Series = r.FormValue("series")

	if cid := r.FormValue("contributor_id"); cid != "" {
		if id, err := uuid.Parse(cid); err == nil {
			content.ContributorID = &id
			if contributor, err := h.service.GetContributor(r.Context(), id); err == nil {
				content.ContributorHandle = contributor.Handle
			}
		}
	}

	if order := r.FormValue("series_order"); order != "" {
		if o, err := strconv.Atoi(order); err == nil {
			content.SeriesOrder = o
		}
	}

	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			content.UserID = userID
			content.CreatedBy = userID
			content.UpdatedBy = userID
		}
	}
	content.AuthorUsername = h.userNameFn(r.Context())

	if err := h.service.CreateContent(r.Context(), content); err != nil {
		h.log.Errorf("Cannot create content: %v", err)
		sections, _ := h.service.GetSections(r.Context(), site.ID)
		tags, _ := h.service.GetTags(r.Context(), site.ID)
		contributors, _ := h.service.GetContributors(r.Context(), site.ID)
		h.render(w, r, "ssg/contents/new", PageData{
			Title:        "New Content",
			Site:         site,
			Content:      content,
			Sections:     sections,
			Tags:         tags,
			Contributors: contributors,
			Error:        "Cannot create content",
		})
		return
	}

	// Handle tags (Tagify format)
	h.processTagifyTags(r.Context(), site.ID, content.ID, r.FormValue("tags"))

	h.siteRedirect(w, r, "/ssg/get-content?id="+content.ID.String())
}

func (h *Handler) HandleShowContent(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	contentID, err := uuid.Parse(r.URL.Query().Get("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid content ID")
		return
	}

	content, err := h.service.GetContent(r.Context(), contentID)
	if err != nil {
		h.log.Errorf("Cannot get content: %v", err)
		h.renderError(w, r, http.StatusNotFound, "Content not found")
		return
	}

	// Load tags
	content.Tags, _ = h.service.GetTagsForContent(r.Context(), contentID)

	h.render(w, r, "ssg/contents/show", PageData{
		Title:   content.Heading,
		Site:    site,
		Content: content,
	})
}

func (h *Handler) HandleEditContent(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	contentID, err := uuid.Parse(r.URL.Query().Get("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid content ID")
		return
	}

	content, err := h.service.GetContent(r.Context(), contentID)
	if err != nil {
		h.log.Errorf("Cannot get content: %v", err)
		h.renderError(w, r, http.StatusNotFound, "Content not found")
		return
	}

	content.Tags, _ = h.service.GetTagsForContent(r.Context(), contentID)
	sections, _ := h.service.GetSections(r.Context(), site.ID)
	tags, _ := h.service.GetTags(r.Context(), site.ID)
	contributors, _ := h.service.GetContributors(r.Context(), site.ID)

	// Get content images and separate header from content images
	allImages, _ := h.service.GetContentImagesWithDetails(r.Context(), contentID)
	var headerImage *ContentImageWithDetails
	var contentImages []*ContentImageWithDetails
	for _, img := range allImages {
		if img.IsHeader {
			headerImage = img
		} else {
			contentImages = append(contentImages, img)
		}
	}

	// Get meta for SEO/settings
	meta, _ := h.service.GetMetaByContentID(r.Context(), contentID)

	h.render(w, r, "ssg/contents/edit", PageData{
		Title:         "Edit " + content.Heading,
		Site:          site,
		Content:       content,
		Sections:      sections,
		Tags:          tags,
		Contributors:  contributors,
		HeaderImage:   headerImage,
		ContentImages: contentImages,
		Meta:          meta,
	})
}

func (h *Handler) HandleUpdateContent(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	contentID, err := uuid.Parse(r.FormValue("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid content ID")
		return
	}

	content, err := h.service.GetContent(r.Context(), contentID)
	if err != nil {
		h.log.Errorf("Cannot get content: %v", err)
		h.renderError(w, r, http.StatusNotFound, "Content not found")
		return
	}

	content.Heading = r.FormValue("heading")
	content.Summary = r.FormValue("summary")
	content.Body = r.FormValue("body")
	content.Kind = r.FormValue("kind")
	content.Draft = r.FormValue("draft") == "on"
	content.Featured = r.FormValue("featured") == "on"
	content.Series = r.FormValue("series")

	if sid := r.FormValue("section_id"); sid != "" {
		if id, err := uuid.Parse(sid); err == nil {
			content.SectionID = id
		}
	}

	if cid := r.FormValue("contributor_id"); cid != "" {
		if id, err := uuid.Parse(cid); err == nil {
			content.ContributorID = &id
			if contributor, err := h.service.GetContributor(r.Context(), id); err == nil {
				content.ContributorHandle = contributor.Handle
			}
		}
	} else {
		content.ContributorID = nil
		content.ContributorHandle = ""
	}

	if order := r.FormValue("series_order"); order != "" {
		if o, err := strconv.Atoi(order); err == nil {
			content.SeriesOrder = o
		}
	}

	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			content.UpdatedBy = userID
		}
	}

	if err := h.service.UpdateContent(r.Context(), content); err != nil {
		h.log.Errorf("Cannot update content: %v", err)
		sections, _ := h.service.GetSections(r.Context(), site.ID)
		tags, _ := h.service.GetTags(r.Context(), site.ID)
		contributors, _ := h.service.GetContributors(r.Context(), site.ID)
		h.render(w, r, "ssg/contents/edit", PageData{
			Title:        "Edit " + content.Heading,
			Site:         site,
			Content:      content,
			Sections:     sections,
			Tags:         tags,
			Contributors: contributors,
			Error:        "Cannot update content",
		})
		return
	}

	// Update tags (Tagify format)
	_ = h.service.RemoveAllTagsFromContent(r.Context(), content.ID)
	h.processTagifyTags(r.Context(), site.ID, content.ID, r.FormValue("tags"))

	h.siteRedirect(w, r, "/ssg/get-content?id="+content.ID.String())
}

func (h *Handler) HandleAutosaveContent(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<div id="save-status" class="save-status error">Site context required</div>`))
		return
	}

	if err := r.ParseForm(); err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<div id="save-status" class="save-status error">Error parsing form</div>`))
		return
	}

	var content *Content
	var isNew bool

	contentIDStr := r.FormValue("id")
	if contentIDStr == "" {
		isNew = true
		var sectionID uuid.UUID
		if sid := r.FormValue("section_id"); sid != "" {
			sectionID, _ = uuid.Parse(sid)
		}
		content = NewContent(site.ID, sectionID, r.FormValue("heading"), r.FormValue("body"))
	} else {
		contentID, err := uuid.Parse(contentIDStr)
		if err != nil {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<div id="save-status" class="save-status error">Invalid content ID</div>`))
			return
		}
		content, err = h.service.GetContent(r.Context(), contentID)
		if err != nil {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<div id="save-status" class="save-status error">Content not found</div>`))
			return
		}
	}

	content.Heading = r.FormValue("heading")
	content.Summary = r.FormValue("summary")
	content.Body = r.FormValue("body")
	content.Draft = r.FormValue("draft") == "on"
	content.Featured = r.FormValue("featured") == "on"
	content.Series = r.FormValue("series")
	content.Kind = r.FormValue("kind")

	if sectionID := r.FormValue("section_id"); sectionID != "" {
		if id, err := uuid.Parse(sectionID); err == nil {
			content.SectionID = id
		}
	}

	if cid := r.FormValue("contributor_id"); cid != "" {
		if id, err := uuid.Parse(cid); err == nil {
			content.ContributorID = &id
			if contributor, err := h.service.GetContributor(r.Context(), id); err == nil {
				content.ContributorHandle = contributor.Handle
			}
		}
	} else {
		content.ContributorID = nil
		content.ContributorHandle = ""
	}

	if seriesOrder := r.FormValue("series_order"); seriesOrder != "" {
		if order, err := strconv.Atoi(seriesOrder); err == nil {
			content.SeriesOrder = order
		}
	}

	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			if isNew {
				content.CreatedBy = userID
				content.AuthorUsername = h.userNameFn(r.Context())
			}
			content.UpdatedBy = userID
		}
	}

	if isNew {
		if err := h.service.CreateContent(r.Context(), content); err != nil {
			h.log.Errorf("Autosave create failed: %v", err)
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<div id="save-status" class="save-status error">Save failed</div>`))
			return
		}
	} else {
		if err := h.service.UpdateContent(r.Context(), content); err != nil {
			h.log.Errorf("Autosave update failed: %v", err)
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<div id="save-status" class="save-status error">Save failed</div>`))
			return
		}
	}

	_ = h.service.RemoveAllTagsFromContent(r.Context(), content.ID)
	h.processTagifyTags(r.Context(), site.ID, content.ID, r.FormValue("tags"))

	w.Header().Set("Content-Type", "text/html")
	timestamp := time.Now().Unix()
	w.Write([]byte(fmt.Sprintf(`<div id="save-status" class="save-status saved" data-saved-at="%d" data-content-id="%s"><span id="save-indicator" class="htmx-indicator">Saving...</span><span id="save-text">Saved just now</span></div>`, timestamp, content.ID.String())))
}

func (h *Handler) HandleDeleteContent(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	contentID, err := uuid.Parse(r.FormValue("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid content ID")
		return
	}

	if err := h.service.DeleteContent(r.Context(), contentID); err != nil {
		h.log.Errorf("Cannot delete content: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Cannot delete content")
		return
	}

	h.siteRedirect(w, r, "/ssg/list-contents")
}

// --- Layout Handlers ---

func (h *Handler) HandleListLayouts(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	layouts, err := h.service.GetLayouts(r.Context(), site.ID)
	if err != nil {
		h.log.Errorf("Cannot list layouts: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Cannot load layouts")
		return
	}

	h.render(w, r, "ssg/layouts/list", PageData{
		Title:   "Layouts",
		Site:    site,
		Layouts: layouts,
	})
}

func (h *Handler) HandleNewLayout(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	h.render(w, r, "ssg/layouts/new", PageData{
		Title: "New Layout",
		Site:  site,
	})
}

func (h *Handler) HandleCreateLayout(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	layout := NewLayout(site.ID, r.FormValue("name"), r.FormValue("description"))
	layout.Code = r.FormValue("code")

	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			layout.CreatedBy = userID
			layout.UpdatedBy = userID
		}
	}

	if err := h.service.CreateLayout(r.Context(), layout); err != nil {
		h.log.Errorf("Cannot create layout: %v", err)
		h.render(w, r, "ssg/layouts/new", PageData{
			Title:  "New Layout",
			Site:   site,
			Layout: layout,
			Error:  "Cannot create layout",
		})
		return
	}

	h.siteRedirect(w, r, "/ssg/get-layout?id="+layout.ID.String())
}

func (h *Handler) HandleShowLayout(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	layoutID, err := uuid.Parse(r.URL.Query().Get("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid layout ID")
		return
	}

	layout, err := h.service.GetLayout(r.Context(), layoutID)
	if err != nil {
		h.log.Errorf("Cannot get layout: %v", err)
		h.renderError(w, r, http.StatusNotFound, "Layout not found")
		return
	}

	h.render(w, r, "ssg/layouts/show", PageData{
		Title:  layout.Name,
		Site:   site,
		Layout: layout,
	})
}

func (h *Handler) HandleEditLayout(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	layoutID, err := uuid.Parse(r.URL.Query().Get("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid layout ID")
		return
	}

	layout, err := h.service.GetLayout(r.Context(), layoutID)
	if err != nil {
		h.log.Errorf("Cannot get layout: %v", err)
		h.renderError(w, r, http.StatusNotFound, "Layout not found")
		return
	}

	h.render(w, r, "ssg/layouts/edit", PageData{
		Title:  "Edit " + layout.Name,
		Site:   site,
		Layout: layout,
	})
}

func (h *Handler) HandleUpdateLayout(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	layoutID, err := uuid.Parse(r.FormValue("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid layout ID")
		return
	}

	layout, err := h.service.GetLayout(r.Context(), layoutID)
	if err != nil {
		h.log.Errorf("Cannot get layout: %v", err)
		h.renderError(w, r, http.StatusNotFound, "Layout not found")
		return
	}

	layout.Name = r.FormValue("name")
	layout.Description = r.FormValue("description")
	layout.Code = r.FormValue("code")

	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			layout.UpdatedBy = userID
		}
	}

	if err := h.service.UpdateLayout(r.Context(), layout); err != nil {
		h.log.Errorf("Cannot update layout: %v", err)
		h.render(w, r, "ssg/layouts/edit", PageData{
			Title:  "Edit " + layout.Name,
			Site:   site,
			Layout: layout,
			Error:  "Cannot update layout",
		})
		return
	}

	h.siteRedirect(w, r, "/ssg/get-layout?id="+layout.ID.String())
}

func (h *Handler) HandleDeleteLayout(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	layoutID, err := uuid.Parse(r.FormValue("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid layout ID")
		return
	}

	if err := h.service.DeleteLayout(r.Context(), layoutID); err != nil {
		h.log.Errorf("Cannot delete layout: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Cannot delete layout")
		return
	}

	h.siteRedirect(w, r, "/ssg/list-layouts")
}

// --- Tag Handlers ---

func (h *Handler) HandleListTags(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	tags, err := h.service.GetTags(r.Context(), site.ID)
	if err != nil {
		h.log.Errorf("Cannot list tags: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Cannot load tags")
		return
	}

	h.render(w, r, "ssg/tags/list", PageData{
		Title: "Tags",
		Site:  site,
		Tags:  tags,
	})
}

func (h *Handler) HandleNewTag(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	h.render(w, r, "ssg/tags/new", PageData{
		Title: "New Tag",
		Site:  site,
	})
}

func (h *Handler) HandleCreateTag(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	tag := NewTag(site.ID, r.FormValue("name"))

	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			tag.CreatedBy = userID
			tag.UpdatedBy = userID
		}
	}

	if err := h.service.CreateTag(r.Context(), tag); err != nil {
		h.log.Errorf("Cannot create tag: %v", err)
		h.render(w, r, "ssg/tags/new", PageData{
			Title: "New Tag",
			Site:  site,
			Tag:   tag,
			Error: "Cannot create tag",
		})
		return
	}

	h.siteRedirect(w, r, "/ssg/list-tags")
}

func (h *Handler) HandleShowTag(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	tagID, err := uuid.Parse(r.URL.Query().Get("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid tag ID")
		return
	}

	tag, err := h.service.GetTag(r.Context(), tagID)
	if err != nil {
		h.log.Errorf("Cannot get tag: %v", err)
		h.renderError(w, r, http.StatusNotFound, "Tag not found")
		return
	}

	h.render(w, r, "ssg/tags/show", PageData{
		Title: tag.Name,
		Site:  site,
		Tag:   tag,
	})
}

func (h *Handler) HandleEditTag(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	tagID, err := uuid.Parse(r.URL.Query().Get("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid tag ID")
		return
	}

	tag, err := h.service.GetTag(r.Context(), tagID)
	if err != nil {
		h.log.Errorf("Cannot get tag: %v", err)
		h.renderError(w, r, http.StatusNotFound, "Tag not found")
		return
	}

	h.render(w, r, "ssg/tags/edit", PageData{
		Title: "Edit " + tag.Name,
		Site:  site,
		Tag:   tag,
	})
}

func (h *Handler) HandleUpdateTag(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	tagID, err := uuid.Parse(r.FormValue("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid tag ID")
		return
	}

	tag, err := h.service.GetTag(r.Context(), tagID)
	if err != nil {
		h.log.Errorf("Cannot get tag: %v", err)
		h.renderError(w, r, http.StatusNotFound, "Tag not found")
		return
	}

	tag.Name = r.FormValue("name")
	tag.Slug = Slugify(tag.Name)

	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			tag.UpdatedBy = userID
		}
	}

	if err := h.service.UpdateTag(r.Context(), tag); err != nil {
		h.log.Errorf("Cannot update tag: %v", err)
		h.render(w, r, "ssg/tags/edit", PageData{
			Title: "Edit " + tag.Name,
			Site:  site,
			Tag:   tag,
			Error: "Cannot update tag",
		})
		return
	}

	h.siteRedirect(w, r, "/ssg/list-tags")
}

func (h *Handler) HandleDeleteTag(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	tagID, err := uuid.Parse(r.FormValue("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid tag ID")
		return
	}

	if err := h.service.DeleteTag(r.Context(), tagID); err != nil {
		h.log.Errorf("Cannot delete tag: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Cannot delete tag")
		return
	}

	h.siteRedirect(w, r, "/ssg/list-tags")
}

// --- Param Handlers ---

func (h *Handler) HandleListParams(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	params, err := h.service.GetParams(r.Context(), site.ID)
	if err != nil {
		h.log.Errorf("Cannot list params: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Cannot load params")
		return
	}

	h.render(w, r, "ssg/params/list", PageData{
		Title:  "Parameters",
		Site:   site,
		Params: params,
	})
}

func (h *Handler) HandleNewParam(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	h.render(w, r, "ssg/params/new", PageData{
		Title: "New Parameter",
		Site:  site,
	})
}

func (h *Handler) HandleCreateParam(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	param := NewParam(site.ID, r.FormValue("name"), r.FormValue("value"))
	param.Description = r.FormValue("description")
	param.RefKey = r.FormValue("ref_key")

	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			param.CreatedBy = userID
			param.UpdatedBy = userID
		}
	}

	if err := h.service.CreateParam(r.Context(), param); err != nil {
		h.log.Errorf("Cannot create param: %v", err)
		h.render(w, r, "ssg/params/new", PageData{
			Title: "New Parameter",
			Site:  site,
			Param: param,
			Error: "Cannot create parameter",
		})
		return
	}

	h.siteRedirect(w, r, "/ssg/list-params")
}

func (h *Handler) HandleShowParam(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	paramID, err := uuid.Parse(r.URL.Query().Get("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid param ID")
		return
	}

	param, err := h.service.GetParam(r.Context(), paramID)
	if err != nil {
		h.log.Errorf("Cannot get param: %v", err)
		h.renderError(w, r, http.StatusNotFound, "Parameter not found")
		return
	}

	h.render(w, r, "ssg/params/show", PageData{
		Title: param.Name,
		Site:  site,
		Param: param,
	})
}

func (h *Handler) HandleEditParam(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	paramID, err := uuid.Parse(r.URL.Query().Get("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid param ID")
		return
	}

	param, err := h.service.GetParam(r.Context(), paramID)
	if err != nil {
		h.log.Errorf("Cannot get param: %v", err)
		h.renderError(w, r, http.StatusNotFound, "Parameter not found")
		return
	}

	h.render(w, r, "ssg/params/edit", PageData{
		Title: "Edit " + param.Name,
		Site:  site,
		Param: param,
	})
}

func (h *Handler) HandleUpdateParam(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	paramID, err := uuid.Parse(r.FormValue("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid param ID")
		return
	}

	param, err := h.service.GetParam(r.Context(), paramID)
	if err != nil {
		h.log.Errorf("Cannot get param: %v", err)
		h.renderError(w, r, http.StatusNotFound, "Parameter not found")
		return
	}

	param.Name = r.FormValue("name")
	param.Description = r.FormValue("description")
	param.Value = r.FormValue("value")
	// RefKey is immutable after creation

	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			param.UpdatedBy = userID
		}
	}

	if err := h.service.UpdateParam(r.Context(), param); err != nil {
		h.log.Errorf("Cannot update param: %v", err)
		h.render(w, r, "ssg/params/edit", PageData{
			Title: "Edit " + param.Name,
			Site:  site,
			Param: param,
			Error: "Cannot update parameter",
		})
		return
	}

	h.siteRedirect(w, r, "/ssg/list-params")
}

func (h *Handler) HandleDeleteParam(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	paramID, err := uuid.Parse(r.FormValue("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid param ID")
		return
	}

	if err := h.service.DeleteParam(r.Context(), paramID); err != nil {
		h.log.Errorf("Cannot delete param: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Cannot delete parameter")
		return
	}

	h.siteRedirect(w, r, "/ssg/list-params")
}

// --- Image Handlers ---

func (h *Handler) HandleListImages(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	images, err := h.service.GetImages(r.Context(), site.ID)
	if err != nil {
		h.log.Errorf("Cannot list images: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Cannot load images")
		return
	}

	h.render(w, r, "ssg/images/list", PageData{
		Title:  "Images",
		Site:   site,
		Images: images,
	})
}

func (h *Handler) HandleNewImage(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	h.render(w, r, "ssg/images/new", PageData{
		Title: "New Image",
		Site:  site,
	})
}

func (h *Handler) HandleCreateImage(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		h.log.Errorf("Cannot parse multipart form: %v", err)
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	// Get uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		h.log.Errorf("Cannot get uploaded file: %v", err)
		h.render(w, r, "ssg/images/new", PageData{
			Title: "Upload Image",
			Site:  site,
			Error: "Please select a file to upload",
		})
		return
	}
	defer file.Close()

	// Get form values
	title := r.FormValue("title")
	altText := r.FormValue("alt_text")

	// Determine target path
	imagesPath := h.workspace.GetImagesPath(site.Slug)
	if err := os.MkdirAll(imagesPath, 0755); err != nil {
		h.log.Errorf("Cannot create images directory: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Cannot create images directory")
		return
	}

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	uniqueID := uuid.New().String()[:8]
	fileName := Slugify(strings.TrimSuffix(header.Filename, ext)) + "-" + uniqueID + ext
	filePath := filepath.Join(imagesPath, fileName)

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		h.log.Errorf("Cannot create file: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Cannot save file")
		return
	}
	defer dst.Close()

	// Copy uploaded file
	if _, err := io.Copy(dst, file); err != nil {
		h.log.Errorf("Cannot write file: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Cannot save file")
		return
	}

	// Create image record
	image := NewImage(site.ID, header.Filename, fileName)
	image.Title = title
	image.AltText = altText

	// Get user ID from context
	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			image.CreatedBy = userID
			image.UpdatedBy = userID
		}
	}

	if err := h.service.CreateImage(r.Context(), image); err != nil {
		h.log.Errorf("Cannot create image record: %v", err)
		// Delete uploaded file on error
		os.Remove(filePath)
		h.render(w, r, "ssg/images/new", PageData{
			Title: "Upload Image",
			Site:  site,
			Error: "Cannot save image record",
		})
		return
	}

	h.log.Infof("Image uploaded: %s", fileName)
	h.siteRedirect(w, r, "/ssg/list-images")
}

func (h *Handler) HandleShowImage(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	imageID, err := uuid.Parse(r.URL.Query().Get("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid image ID")
		return
	}

	image, err := h.service.GetImage(r.Context(), imageID)
	if err != nil {
		h.log.Errorf("Cannot get image: %v", err)
		h.renderError(w, r, http.StatusNotFound, "Image not found")
		return
	}

	h.render(w, r, "ssg/images/show", PageData{
		Title: image.FileName,
		Site:  site,
		Image: image,
	})
}

func (h *Handler) HandleEditImage(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	imageID, err := uuid.Parse(r.URL.Query().Get("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid image ID")
		return
	}

	image, err := h.service.GetImage(r.Context(), imageID)
	if err != nil {
		h.log.Errorf("Cannot get image: %v", err)
		h.renderError(w, r, http.StatusNotFound, "Image not found")
		return
	}

	h.render(w, r, "ssg/images/edit", PageData{
		Title: "Edit " + image.FileName,
		Site:  site,
		Image: image,
	})
}

func (h *Handler) HandleUpdateImage(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	imageID, err := uuid.Parse(r.FormValue("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid image ID")
		return
	}

	image, err := h.service.GetImage(r.Context(), imageID)
	if err != nil {
		h.log.Errorf("Cannot get image: %v", err)
		h.renderError(w, r, http.StatusNotFound, "Image not found")
		return
	}

	image.AltText = r.FormValue("alt_text")
	image.Title = r.FormValue("title")

	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			image.UpdatedBy = userID
		}
	}

	if err := h.service.UpdateImage(r.Context(), image); err != nil {
		h.log.Errorf("Cannot update image: %v", err)
		h.render(w, r, "ssg/images/edit", PageData{
			Title: "Edit " + image.FileName,
			Site:  site,
			Image: image,
			Error: "Cannot update image",
		})
		return
	}

	h.siteRedirect(w, r, "/ssg/list-images")
}

func (h *Handler) HandleDeleteImage(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	imageID, err := uuid.Parse(r.FormValue("id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid image ID")
		return
	}

	if err := h.service.DeleteImage(r.Context(), imageID); err != nil {
		h.log.Errorf("Cannot delete image: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Cannot delete image")
		return
	}

	h.siteRedirect(w, r, "/ssg/list-images")
}

// --- Workspace File Handlers ---

// HandleServeWorkspaceImage serves images from the workspace directory.
func (h *Handler) HandleServeWorkspaceImage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	filename := chi.URLParam(r, "filename")

	if slug == "" || filename == "" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Prevent directory traversal
	if strings.Contains(slug, "..") || strings.Contains(filename, "..") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Build the file path
	filePath := filepath.Join(h.workspace.GetImagesPath(slug), filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}

	// Serve the file
	http.ServeFile(w, r, filePath)
}

// --- Content Image Handlers ---

func (h *Handler) HandleUploadContentImage(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		http.Error(w, "Site context required", http.StatusBadRequest)
		return
	}

	contentID, err := uuid.Parse(r.URL.Query().Get("content_id"))
	if err != nil {
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		h.log.Errorf("Cannot parse multipart form: %v", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Get uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		h.log.Errorf("Cannot get uploaded file: %v", err)
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get form values
	altText := r.FormValue("alt_text")
	title := r.FormValue("title")
	purpose := r.FormValue("purpose") // "header" or "content"
	isHeader := purpose == "header"

	// Determine target path
	imagesPath := h.workspace.GetImagesPath(site.Slug)
	if err := os.MkdirAll(imagesPath, 0755); err != nil {
		h.log.Errorf("Cannot create images directory: %v", err)
		http.Error(w, "Cannot create images directory", http.StatusInternalServerError)
		return
	}

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	uniqueID := uuid.New().String()[:8]
	fileName := Slugify(strings.TrimSuffix(header.Filename, ext)) + "-" + uniqueID + ext
	filePath := filepath.Join(imagesPath, fileName)

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		h.log.Errorf("Cannot create file: %v", err)
		http.Error(w, "Cannot save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy uploaded file
	if _, err := io.Copy(dst, file); err != nil {
		h.log.Errorf("Cannot write file: %v", err)
		http.Error(w, "Cannot save file", http.StatusInternalServerError)
		return
	}

	// Create image record
	image := NewImage(site.ID, header.Filename, fileName)
	image.AltText = altText
	image.Title = title

	// Get user ID from context
	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			image.CreatedBy = userID
			image.UpdatedBy = userID
		}
	}

	if err := h.service.CreateImage(r.Context(), image); err != nil {
		h.log.Errorf("Cannot create image record: %v", err)
		os.Remove(filePath)
		http.Error(w, "Cannot save image record", http.StatusInternalServerError)
		return
	}

	// If this is a header image, remove existing header first
	if isHeader {
		_ = h.service.UnlinkHeaderImageFromContent(r.Context(), contentID)
	}

	// Link image to content
	if err := h.service.LinkImageToContent(r.Context(), contentID, image.ID, isHeader); err != nil {
		h.log.Errorf("Cannot link image to content: %v", err)
		http.Error(w, "Cannot link image to content", http.StatusInternalServerError)
		return
	}

	h.log.Infof("Content image uploaded: %s (header: %v)", fileName, isHeader)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) HandleDeleteContentImage(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		http.Error(w, "Site context required", http.StatusBadRequest)
		return
	}

	contentImageID, err := uuid.Parse(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "Invalid content image ID", http.StatusBadRequest)
		return
	}

	// Get image details before deleting
	imageDetails, err := h.service.GetContentImageDetails(r.Context(), contentImageID)
	if err != nil {
		h.log.Errorf("Cannot get content image details: %v", err)
		http.Error(w, "Cannot find content image", http.StatusNotFound)
		return
	}

	// Delete the content_images link
	if err := h.service.UnlinkImageFromContent(r.Context(), contentImageID); err != nil {
		h.log.Errorf("Cannot delete content image link: %v", err)
		http.Error(w, "Cannot delete content image", http.StatusInternalServerError)
		return
	}

	// Delete the image record
	if err := h.service.DeleteImage(r.Context(), imageDetails.ImageID); err != nil {
		h.log.Errorf("Cannot delete image record: %v", err)
	}

	// Delete the physical file
	filePath := filepath.Join(h.workspace.GetImagesPath(site.Slug), imageDetails.FilePath)
	if err := os.Remove(filePath); err != nil {
		h.log.Errorf("Cannot delete image file: %v", err)
	}

	h.log.Infof("Content image fully deleted: %s", contentImageID)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) HandleRemoveHeaderImage(w http.ResponseWriter, r *http.Request) {
	contentID, err := uuid.Parse(r.URL.Query().Get("content_id"))
	if err != nil {
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	if err := h.service.UnlinkHeaderImageFromContent(r.Context(), contentID); err != nil {
		h.log.Errorf("Cannot remove header image: %v", err)
		http.Error(w, "Cannot remove header image", http.StatusInternalServerError)
		return
	}

	h.log.Infof("Header image removed from content: %s", contentID)
	w.WriteHeader(http.StatusOK)
}

// --- Section Image Handlers ---

func (h *Handler) HandleUploadSectionImage(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.log.Errorf("HandleUploadSectionImage: site context required")
		http.Error(w, "Site context required", http.StatusBadRequest)
		return
	}

	sectionID, err := uuid.Parse(r.URL.Query().Get("section_id"))
	if err != nil {
		h.log.Errorf("HandleUploadSectionImage: invalid section ID: %v", err)
		http.Error(w, "Invalid section ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		h.log.Errorf("Cannot parse multipart form: %v", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		h.log.Errorf("Cannot get uploaded file: %v", err)
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	altText := r.FormValue("alt_text")
	title := r.FormValue("title")
	purpose := r.FormValue("purpose")
	isHeader := purpose == "header"

	imagesPath := h.workspace.GetImagesPath(site.Slug)
	if err := os.MkdirAll(imagesPath, 0755); err != nil {
		h.log.Errorf("Cannot create images directory: %v", err)
		http.Error(w, "Cannot create images directory", http.StatusInternalServerError)
		return
	}

	ext := filepath.Ext(header.Filename)
	uniqueID := uuid.New().String()[:8]
	fileName := Slugify(strings.TrimSuffix(header.Filename, ext)) + "-" + uniqueID + ext
	filePath := filepath.Join(imagesPath, fileName)

	dst, err := os.Create(filePath)
	if err != nil {
		h.log.Errorf("Cannot create file: %v", err)
		http.Error(w, "Cannot save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		h.log.Errorf("Cannot write file: %v", err)
		http.Error(w, "Cannot save file", http.StatusInternalServerError)
		return
	}

	image := NewImage(site.ID, header.Filename, fileName)
	image.AltText = altText
	image.Title = title

	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			image.CreatedBy = userID
			image.UpdatedBy = userID
		}
	}

	if err := h.service.CreateImage(r.Context(), image); err != nil {
		h.log.Errorf("Cannot create image record: %v", err)
		os.Remove(filePath)
		http.Error(w, "Cannot save image record", http.StatusInternalServerError)
		return
	}

	if err := h.service.LinkImageToSection(r.Context(), sectionID, image.ID, isHeader); err != nil {
		h.log.Errorf("Cannot link image to section: %v", err)
		http.Error(w, "Cannot link image to section", http.StatusInternalServerError)
		return
	}

	h.log.Infof("Section image uploaded: %s (header: %v)", fileName, isHeader)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) HandleDeleteSectionImage(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.log.Errorf("HandleDeleteSectionImage: site context required")
		http.Error(w, "Site context required", http.StatusBadRequest)
		return
	}

	sectionImageID, err := uuid.Parse(r.URL.Query().Get("id"))
	if err != nil {
		h.log.Errorf("HandleDeleteSectionImage: invalid section image ID: %v", err)
		http.Error(w, "Invalid section image ID", http.StatusBadRequest)
		return
	}

	imageDetails, err := h.service.GetSectionImageDetails(r.Context(), sectionImageID)
	if err != nil {
		h.log.Errorf("Cannot get section image details: %v", err)
		http.Error(w, "Cannot find section image", http.StatusNotFound)
		return
	}

	if err := h.service.UnlinkImageFromSection(r.Context(), sectionImageID); err != nil {
		h.log.Errorf("Cannot delete section image link: %v", err)
		http.Error(w, "Cannot delete section image", http.StatusInternalServerError)
		return
	}

	if err := h.service.DeleteImage(r.Context(), imageDetails.ImageID); err != nil {
		h.log.Errorf("Cannot delete image record: %v", err)
	}

	filePath := filepath.Join(h.workspace.GetImagesPath(site.Slug), imageDetails.FilePath)
	if err := os.Remove(filePath); err != nil {
		h.log.Errorf("Cannot delete image file: %v", err)
	}

	h.log.Infof("Section image fully deleted: %s", sectionImageID)
	w.WriteHeader(http.StatusOK)
}

// --- Meta Handlers ---

func (h *Handler) HandleUpdateMeta(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.log.Errorf("HandleUpdateMeta: site context required")
		http.Error(w, "Site context required", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.log.Errorf("HandleUpdateMeta: invalid form data: %v", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	contentID, err := uuid.Parse(r.FormValue("content_id"))
	if err != nil {
		h.log.Errorf("HandleUpdateMeta: invalid content ID: %v", err)
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	// Get existing meta or create new one
	meta, _ := h.service.GetMetaByContentID(r.Context(), contentID)
	isNew := meta == nil

	if isNew {
		meta = &Meta{
			ID:        uuid.New(),
			SiteID:    site.ID,
			ContentID: contentID,
			CreatedAt: time.Now(),
		}
	}

	// Update fields from form
	meta.Description = r.FormValue("description")
	meta.Keywords = r.FormValue("keywords")
	meta.Robots = r.FormValue("robots")
	meta.CanonicalURL = r.FormValue("canonical_url")
	meta.TableOfContents = r.FormValue("table_of_contents") == "on"
	meta.Share = r.FormValue("share") == "on"
	meta.Comments = r.FormValue("comments") == "on"
	meta.UpdatedAt = time.Now()

	// Get user ID from context
	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			meta.UpdatedBy = userID
			if isNew {
				meta.CreatedBy = userID
			}
		}
	}

	var saveErr error
	if isNew {
		saveErr = h.service.CreateMeta(r.Context(), meta)
	} else {
		saveErr = h.service.UpdateMeta(r.Context(), meta)
	}

	if saveErr != nil {
		h.log.Errorf("Cannot save meta: %v", saveErr)
		http.Error(w, "Cannot save meta", http.StatusInternalServerError)
		return
	}

	h.log.Infof("Meta updated for content: %s", contentID)
	w.WriteHeader(http.StatusOK)
}

// --- Contributor Handlers ---

func (h *Handler) HandleListContributors(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	contributors, err := h.service.GetContributors(r.Context(), site.ID)
	if err != nil {
		h.renderError(w, r, http.StatusInternalServerError, "Cannot load contributors")
		return
	}

	h.render(w, r, "ssg/contributors/list", PageData{
		Title:        "Contributors",
		Site:         site,
		Contributors: contributors,
	})
}

func (h *Handler) HandleNewContributor(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	h.render(w, r, "ssg/contributors/new", PageData{
		Title: "New Contributor",
		Site:  site,
	})
}

func (h *Handler) HandleCreateContributor(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	handle := r.FormValue("handle")
	name := r.FormValue("name")
	surname := r.FormValue("surname")
	bio := r.FormValue("bio")

	if handle == "" || name == "" {
		h.render(w, r, "ssg/contributors/new", PageData{
			Title: "New Contributor",
			Site:  site,
			Error: "Handle and name are required",
		})
		return
	}

	userIDStr := middleware.GetUserID(r.Context())

	contributorProfile, err := h.profileService.CreateProfile(
		r.Context(),
		normalizeSlug(handle),
		name,
		surname,
		bio,
		"[]",
		"",
		userIDStr,
	)
	if err != nil {
		h.render(w, r, "ssg/contributors/new", PageData{
			Title: "New Contributor",
			Site:  site,
			Error: "Cannot create profile: " + err.Error(),
		})
		return
	}

	contributor := NewContributor(site.ID, handle, name, surname)
	contributor.Bio = bio
	contributor.ProfileID = &contributorProfile.ID
	contributor.CreatedBy = parseUUID(userIDStr)
	contributor.UpdatedBy = contributor.CreatedBy

	if err := h.service.CreateContributor(r.Context(), contributor); err != nil {
		h.profileService.DeleteProfile(r.Context(), contributorProfile.ID)
		h.render(w, r, "ssg/contributors/new", PageData{
			Title: "New Contributor",
			Site:  site,
			Error: "Cannot create contributor: " + err.Error(),
		})
		return
	}

	h.siteRedirect(w, r, "/ssg/edit-contributor?id="+contributor.ID.String())
}

func (h *Handler) HandleShowContributor(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid contributor ID")
		return
	}

	contributor, err := h.service.GetContributor(r.Context(), id)
	if err != nil {
		h.renderError(w, r, http.StatusNotFound, "Contributor not found")
		return
	}

	h.render(w, r, "ssg/contributors/show", PageData{
		Title:       contributor.FullName(),
		Site:        site,
		Contributor: contributor,
	})
}

func (h *Handler) HandleEditContributor(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid contributor ID")
		return
	}

	contributor, err := h.service.GetContributor(r.Context(), id)
	if err != nil {
		h.renderError(w, r, http.StatusNotFound, "Contributor not found")
		return
	}

	var contributorProfile *profile.Profile
	var socialLinksMap map[string]string
	if contributor.ProfileID != nil {
		contributorProfile, _ = h.profileService.GetProfile(r.Context(), *contributor.ProfileID)
		if contributorProfile != nil {
			socialLinksMap = parseSocialLinksToMap(contributorProfile.SocialLinks)
		}
	}

	h.render(w, r, "ssg/contributors/edit", PageData{
		Title:              "Edit " + contributor.FullName(),
		Site:               site,
		Contributor:        contributor,
		ContributorProfile: contributorProfile,
		ProfileSocialLinks: socialLinksMap,
	})
}

func (h *Handler) HandleUpdateContributor(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	idStr := r.FormValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid contributor ID")
		return
	}

	contributor, err := h.service.GetContributor(r.Context(), id)
	if err != nil {
		h.renderError(w, r, http.StatusNotFound, "Contributor not found")
		return
	}

	userID := middleware.GetUserID(r.Context())

	contributor.Handle = r.FormValue("handle")
	contributor.Name = r.FormValue("name")
	contributor.Surname = r.FormValue("surname")
	contributor.Bio = r.FormValue("bio")
	contributor.UpdatedBy = parseUUID(userID)
	contributor.UpdatedAt = time.Now()

	if err := h.service.UpdateContributor(r.Context(), contributor); err != nil {
		h.render(w, r, "ssg/contributors/edit", PageData{
			Title:       "Edit " + contributor.FullName(),
			Site:        site,
			Contributor: contributor,
			Error:       "Cannot update contributor",
		})
		return
	}

	profileSlug := normalizeSlug(r.FormValue("profile_slug"))
	profileName := strings.TrimSpace(r.FormValue("profile_name"))

	if profileSlug != "" && profileName != "" {
		profileSurname := strings.TrimSpace(r.FormValue("profile_surname"))
		profileBio := strings.TrimSpace(r.FormValue("profile_bio"))
		socialLinks := buildSocialLinksJSON(r)

		if contributor.ProfileID != nil {
			existingProfile, err := h.profileService.GetProfile(r.Context(), *contributor.ProfileID)
			if err == nil && existingProfile != nil {
				existingProfile.Slug = profileSlug
				existingProfile.Name = profileName
				existingProfile.Surname = profileSurname
				existingProfile.Bio = profileBio
				existingProfile.SocialLinks = socialLinks
				existingProfile.UpdatedBy = userID
				existingProfile.UpdatedAt = time.Now()
				h.profileService.UpdateProfile(r.Context(), existingProfile)
			}
		} else {
			newProfile, err := h.profileService.CreateProfile(r.Context(), profileSlug, profileName, profileSurname, profileBio, socialLinks, "", userID)
			if err == nil && newProfile != nil {
				h.service.SetContributorProfile(r.Context(), contributor.ID, newProfile.ID, userID)
			}
		}
	}

	h.siteRedirect(w, r, "/ssg/get-contributor?id="+id.String())
}

func (h *Handler) HandleDeleteContributor(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	idStr := r.FormValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid contributor ID")
		return
	}

	if err := h.service.DeleteContributor(r.Context(), id); err != nil {
		h.renderError(w, r, http.StatusInternalServerError, "Cannot delete contributor")
		return
	}

	h.siteRedirect(w, r, "/ssg/list-contributors")
}

func (h *Handler) HandleEditContributorProfile(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid contributor ID")
		return
	}

	contributor, err := h.service.GetContributor(r.Context(), id)
	if err != nil {
		h.renderError(w, r, http.StatusNotFound, "Contributor not found")
		return
	}

	if contributor.ProfileID == nil {
		h.renderError(w, r, http.StatusNotFound, "Contributor has no profile")
		return
	}

	contributorProfile, err := h.profileService.GetProfile(r.Context(), *contributor.ProfileID)
	if err != nil {
		h.renderError(w, r, http.StatusNotFound, "Profile not found")
		return
	}

	h.render(w, r, "ssg/contributors/edit-profile", PageData{
		Title:              "Edit Profile: " + contributor.FullName(),
		Site:               site,
		Contributor:        contributor,
		ContributorProfile: contributorProfile,
		ProfileSocialLinks: parseSocialLinksToMap(contributorProfile.SocialLinks),
	})
}

func (h *Handler) HandleUpdateContributorProfile(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	idStr := r.FormValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid contributor ID")
		return
	}

	contributor, err := h.service.GetContributor(r.Context(), id)
	if err != nil {
		h.renderError(w, r, http.StatusNotFound, "Contributor not found")
		return
	}

	if contributor.ProfileID == nil {
		h.renderError(w, r, http.StatusNotFound, "Contributor has no profile")
		return
	}

	contributorProfile, err := h.profileService.GetProfile(r.Context(), *contributor.ProfileID)
	if err != nil {
		h.renderError(w, r, http.StatusNotFound, "Profile not found")
		return
	}

	userIDStr := middleware.GetUserID(r.Context())

	contributorProfile.Slug = normalizeSlug(r.FormValue("slug"))
	contributorProfile.Name = strings.TrimSpace(r.FormValue("name"))
	contributorProfile.Surname = strings.TrimSpace(r.FormValue("surname"))
	contributorProfile.Bio = strings.TrimSpace(r.FormValue("bio"))
	contributorProfile.SocialLinks = buildSocialLinksJSON(r)
	contributorProfile.UpdatedBy = userIDStr

	if err := h.profileService.UpdateProfile(r.Context(), contributorProfile); err != nil {
		h.render(w, r, "ssg/contributors/edit-profile", PageData{
			Title:              "Edit Profile: " + contributor.FullName(),
			Site:               site,
			Contributor:        contributor,
			ContributorProfile: contributorProfile,
			ProfileSocialLinks: parseSocialLinksToMap(contributorProfile.SocialLinks),
			Error:              "Cannot update profile",
		})
		return
	}

	h.siteRedirect(w, r, "/ssg/edit-contributor-profile?id="+id.String())
}

const profilesBasePath = "_workspace/profiles"

func (h *Handler) HandleUploadContributorPhoto(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseMultipartForm(5 << 20); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	contributorID, err := uuid.Parse(r.FormValue("contributor_id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid contributor ID")
		return
	}

	contributor, err := h.service.GetContributor(r.Context(), contributorID)
	if err != nil {
		h.renderError(w, r, http.StatusNotFound, "Contributor not found")
		return
	}

	if contributor.ProfileID == nil {
		http.Error(w, "Contributor has no profile. Save profile data first.", http.StatusBadRequest)
		return
	}

	contributorProfile, err := h.profileService.GetProfile(r.Context(), *contributor.ProfileID)
	if err != nil {
		http.Error(w, "Profile not found", http.StatusNotFound)
		return
	}

	file, header, err := r.FormFile("photo")
	if err != nil {
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	contributorsPhotoPath := filepath.Join(profilesBasePath, "contributors")
	if err := os.MkdirAll(contributorsPhotoPath, 0755); err != nil {
		h.log.Errorf("Cannot create profiles directory: %v", err)
		http.Error(w, "Cannot create directory", http.StatusInternalServerError)
		return
	}

	if contributorProfile.PhotoPath != "" {
		oldPath := filepath.Join(profilesBasePath, contributorProfile.PhotoPath)
		os.Remove(oldPath)
	}

	ext := filepath.Ext(header.Filename)
	fileName := filepath.Join("contributors", contributorProfile.ID.String()+ext)
	filePath := filepath.Join(profilesBasePath, fileName)

	dst, err := os.Create(filePath)
	if err != nil {
		h.log.Errorf("Cannot create file: %v", err)
		http.Error(w, "Cannot save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		h.log.Errorf("Cannot write file: %v", err)
		http.Error(w, "Cannot save file", http.StatusInternalServerError)
		return
	}

	userIDStr := middleware.GetUserID(r.Context())
	contributorProfile.PhotoPath = fileName
	contributorProfile.UpdatedBy = userIDStr

	if err := h.profileService.UpdateProfile(r.Context(), contributorProfile); err != nil {
		h.log.Errorf("Cannot update profile photo path: %v", err)
		http.Error(w, "Cannot update profile", http.StatusInternalServerError)
		return
	}

	h.log.Infof("Contributor profile photo uploaded: %s", fileName)
	h.siteRedirect(w, r, "/ssg/edit-contributor-profile?id="+contributorID.String())
}

func (h *Handler) HandleRemoveContributorPhoto(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	contributorID, err := uuid.Parse(r.FormValue("contributor_id"))
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid contributor ID")
		return
	}

	contributor, err := h.service.GetContributor(r.Context(), contributorID)
	if err != nil {
		h.renderError(w, r, http.StatusNotFound, "Contributor not found")
		return
	}

	if contributor.ProfileID == nil {
		h.siteRedirect(w, r, "/ssg/edit-contributor-profile?id="+contributorID.String())
		return
	}

	contributorProfile, err := h.profileService.GetProfile(r.Context(), *contributor.ProfileID)
	if err != nil {
		h.siteRedirect(w, r, "/ssg/edit-contributor-profile?id="+contributorID.String())
		return
	}

	if contributorProfile.PhotoPath != "" {
		filePath := filepath.Join(profilesBasePath, contributorProfile.PhotoPath)
		os.Remove(filePath)
	}

	userIDStr := middleware.GetUserID(r.Context())
	contributorProfile.PhotoPath = ""
	contributorProfile.UpdatedBy = userIDStr

	if err := h.profileService.UpdateProfile(r.Context(), contributorProfile); err != nil {
		h.log.Errorf("Cannot update profile: %v", err)
		http.Error(w, "Cannot update profile", http.StatusInternalServerError)
		return
	}

	h.log.Info("Contributor profile photo removed")
	h.siteRedirect(w, r, "/ssg/edit-contributor-profile?id="+contributorID.String())
}

// --- Generation Handlers ---

func (h *Handler) HandleBackupMarkdown(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	// Get all content with metadata
	contents, err := h.service.GetAllContentWithMeta(r.Context(), site.ID)
	if err != nil {
		h.log.Errorf("Cannot get content for markdown generation: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Cannot load content")
		return
	}

	// Generate markdown files
	result, err := h.generator.GenerateMarkdown(r.Context(), site.Slug, contents)
	if err != nil {
		h.log.Errorf("Markdown generation failed: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Markdown generation failed")
		return
	}

	h.log.Infof("Markdown generation complete: %d files generated", result.FilesGenerated)
	if len(result.Errors) > 0 {
		h.log.Infof("Markdown generation had %d errors", len(result.Errors))
	}

	// Check if backup repo is configured
	repoURL, _ := h.service.GetParamByRefKey(r.Context(), site.ID, "ssg.backup.repo.url")

	if repoURL != nil && repoURL.Value != "" {
		authToken, _ := h.service.GetParamByRefKey(r.Context(), site.ID, "ssg.backup.auth.token")
		branch, _ := h.service.GetParamByRefKey(r.Context(), site.ID, "ssg.backup.branch")
		commitName, _ := h.service.GetParamByRefKey(r.Context(), site.ID, "ssg.publish.commit.user.name")
		commitEmail, _ := h.service.GetParamByRefKey(r.Context(), site.ID, "ssg.publish.commit.user.email")

		branchValue := "main"
		if branch != nil && branch.Value != "" {
			branchValue = branch.Value
		}

		commitNameValue := "Clio Bot"
		if commitName != nil && commitName.Value != "" {
			commitNameValue = commitName.Value
		}

		commitEmailValue := "clio@localhost"
		if commitEmail != nil && commitEmail.Value != "" {
			commitEmailValue = commitEmail.Value
		}

		authTokenValue := ""
		useSSH := true
		if authToken != nil && authToken.Value != "" && strings.HasPrefix(repoURL.Value, "https://") {
			authTokenValue = authToken.Value
			useSSH = false
		}

		cfg := PublishConfig{
			RepoURL:     repoURL.Value,
			Branch:      branchValue,
			AuthToken:   authTokenValue,
			CommitName:  commitNameValue,
			CommitEmail: commitEmailValue,
			UseSSH:      useSSH,
		}

		backupResult, err := h.publisher.Backup(r.Context(), cfg, site.Slug)
		if err != nil {
			h.log.Errorf("Markdown backup to git failed: %v", err)
			http.Redirect(w, r, "/ssg/get-site?id="+site.ID.String()+"&error=backup_failed", http.StatusSeeOther)
			return
		}

		if backupResult.NoChanges {
			h.log.Info("Markdown backup: no changes to commit")
			http.Redirect(w, r, "/ssg/get-site?id="+site.ID.String()+"&success=backup_no_changes", http.StatusSeeOther)
			return
		}

		h.log.Infof("Markdown backup complete: %s", backupResult.CommitURL)
		http.Redirect(w, r, "/ssg/get-site?id="+site.ID.String()+"&success=backup", http.StatusSeeOther)
		return
	}

	// No backup repo configured, just redirect with markdown success
	http.Redirect(w, r, "/ssg/get-site?id="+site.ID.String()+"&success=markdown", http.StatusSeeOther)
}

func (h *Handler) HandleGenerateHTML(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	// Get all content with metadata
	contents, err := h.service.GetAllContentWithMeta(r.Context(), site.ID)
	if err != nil {
		h.log.Errorf("Cannot get content for HTML generation: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Cannot load content")
		return
	}

	// Get sections
	sections, err := h.service.GetSections(r.Context(), site.ID)
	if err != nil {
		h.log.Errorf("Cannot get sections for HTML generation: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Cannot load sections")
		return
	}

	params, err := h.service.GetParams(r.Context(), site.ID)
	if err != nil {
		h.log.Errorf("Cannot get params for HTML generation: %v", err)
		params = []*Param{}
	}

	contributors, err := h.service.GetContributors(r.Context(), site.ID)
	if err != nil {
		h.log.Errorf("Cannot get contributors for HTML generation: %v", err)
		contributors = []*Contributor{}
	}

	userAuthors := h.service.BuildUserAuthorsMap(r.Context(), contents, contributors)

	result, err := h.htmlGen.GenerateHTML(r.Context(), site, contents, sections, params, contributors, userAuthors)
	if err != nil {
		h.log.Errorf("HTML generation failed: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "HTML generation failed")
		return
	}

	h.log.Infof("HTML generation complete: %d pages, %d index pages, %d author pages", result.PagesGenerated, result.IndexPages, result.AuthorPages)
	if len(result.Errors) > 0 {
		h.log.Infof("HTML generation had %d errors", len(result.Errors))
	}

	// Redirect back to site with success message
	http.Redirect(w, r, "/ssg/get-site?id="+site.ID.String()+"&success=html", http.StatusSeeOther)
}

func (h *Handler) HandlePublish(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, r, http.StatusBadRequest, "Site context required")
		return
	}

	contents, err := h.service.GetAllContentWithMeta(r.Context(), site.ID)
	if err != nil {
		h.log.Errorf("Cannot get content for publish: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Cannot load content")
		return
	}

	sections, err := h.service.GetSections(r.Context(), site.ID)
	if err != nil {
		h.log.Errorf("Cannot get sections for publish: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Cannot load sections")
		return
	}

	params, err := h.service.GetParams(r.Context(), site.ID)
	if err != nil {
		h.log.Errorf("Cannot get params for publish: %v", err)
		params = []*Param{}
	}

	contributors, err := h.service.GetContributors(r.Context(), site.ID)
	if err != nil {
		h.log.Errorf("Cannot get contributors for publish: %v", err)
		contributors = []*Contributor{}
	}

	userAuthors := h.service.BuildUserAuthorsMap(r.Context(), contents, contributors)

	htmlResult, err := h.htmlGen.GenerateHTML(r.Context(), site, contents, sections, params, contributors, userAuthors)
	if err != nil {
		h.log.Errorf("HTML generation failed: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "HTML generation failed")
		return
	}
	h.log.Infof("HTML generation complete: %d pages", htmlResult.PagesGenerated)

	repoURL, _ := h.service.GetParamByRefKey(r.Context(), site.ID, "ssg.publish.repo.url")

	if repoURL == nil || repoURL.Value == "" {
		h.log.Error("Publish repo not configured")
		http.Redirect(w, r, "/ssg/get-site?id="+site.ID.String()+"&error=publish_not_configured", http.StatusSeeOther)
		return
	}

	authToken, _ := h.service.GetParamByRefKey(r.Context(), site.ID, "ssg.publish.auth.token")
	branch, _ := h.service.GetParamByRefKey(r.Context(), site.ID, "ssg.publish.branch")
	commitName, _ := h.service.GetParamByRefKey(r.Context(), site.ID, "ssg.publish.commit.user.name")
	commitEmail, _ := h.service.GetParamByRefKey(r.Context(), site.ID, "ssg.publish.commit.user.email")

	branchValue := "gh-pages"
	if branch != nil && branch.Value != "" {
		branchValue = branch.Value
	}

	commitNameValue := "Clio Bot"
	if commitName != nil && commitName.Value != "" {
		commitNameValue = commitName.Value
	}

	commitEmailValue := "clio@localhost"
	if commitEmail != nil && commitEmail.Value != "" {
		commitEmailValue = commitEmail.Value
	}

	authTokenValue := ""
	useSSH := true
	if authToken != nil && authToken.Value != "" && strings.HasPrefix(repoURL.Value, "https://") {
		authTokenValue = authToken.Value
		useSSH = false
	}

	cfg := PublishConfig{
		RepoURL:     repoURL.Value,
		Branch:      branchValue,
		AuthToken:   authTokenValue,
		CommitName:  commitNameValue,
		CommitEmail: commitEmailValue,
		UseSSH:      useSSH,
	}

	publishResult, err := h.publisher.Publish(r.Context(), cfg, site.Slug)
	if err != nil {
		h.log.Errorf("Publish to git failed: %v", err)
		http.Redirect(w, r, "/ssg/get-site?id="+site.ID.String()+"&error=publish_failed", http.StatusSeeOther)
		return
	}

	if publishResult.NoChanges {
		h.log.Info("Publish: no changes to commit")
		http.Redirect(w, r, "/ssg/get-site?id="+site.ID.String()+"&success=publish_no_changes", http.StatusSeeOther)
		return
	}

	h.log.Infof("Publish complete: %s", publishResult.CommitURL)
	http.Redirect(w, r, "/ssg/get-site?id="+site.ID.String()+"&success=publish", http.StatusSeeOther)
}

type profileSocialLink struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
}

func parseSocialLinksToMap(jsonStr string) map[string]string {
	result := make(map[string]string)
	if jsonStr == "" || jsonStr == "[]" {
		return result
	}
	var links []profileSocialLink
	if err := json.Unmarshal([]byte(jsonStr), &links); err != nil {
		return result
	}
	for _, link := range links {
		result[link.Platform] = link.URL
	}
	return result
}

var socialPlatforms = []string{
	"facebook", "youtube", "instagram", "x", "tiktok", "linkedin", "github",
	"whatsapp", "telegram", "reddit", "messenger", "snapchat", "pinterest",
	"tumblr", "discord", "twitch", "signal", "viber", "line", "kakaotalk",
	"wechat", "qq", "douyin", "kuaishou", "weibo", "xiaohongshu", "bilibili",
	"zhihu", "vk", "odnoklassniki", "mastodon", "bluesky", "threads", "flickr",
	"vimeo", "dailymotion", "quora",
}

func buildSocialLinksJSON(r *http.Request) string {
	var links []profileSocialLink
	for _, platform := range socialPlatforms {
		url := strings.TrimSpace(r.FormValue("profile_social_" + platform))
		if url != "" {
			links = append(links, profileSocialLink{Platform: platform, URL: url})
		}
	}
	if len(links) == 0 {
		return "[]"
	}
	data, err := json.Marshal(links)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func normalizeSlug(s string) string {
	s = strings.ToLower(s)
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	return result.String()
}
