package ssg

import (
	"context"
	"embed"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cliossg/clio/pkg/cl/config"
	"github.com/cliossg/clio/pkg/cl/logger"
	"github.com/cliossg/clio/pkg/cl/middleware"
	"github.com/cliossg/clio/pkg/cl/render"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler handles SSG web routes.
type Handler struct {
	service   Service
	workspace *Workspace
	generator *Generator
	htmlGen   *HTMLGenerator
	siteCtxMw func(http.Handler) http.Handler
	sessionMw func(http.Handler) http.Handler
	assetsFS  embed.FS
	cfg       *config.Config
	log       logger.Logger
}

// NewHandler creates a new SSG handler.
func NewHandler(service Service, siteCtxMw, sessionMw func(http.Handler) http.Handler, assetsFS embed.FS, cfg *config.Config, log logger.Logger) *Handler {
	workspace := NewWorkspace(cfg.SSG.SitesBasePath)
	return &Handler{
		service:   service,
		workspace: workspace,
		generator: NewGenerator(workspace),
		htmlGen:   NewHTMLGenerator(workspace, assetsFS),
		siteCtxMw: siteCtxMw,
		sessionMw: sessionMw,
		assetsFS:  assetsFS,
		cfg:       cfg,
		log:       log,
	}
}

// Start initializes templates and other resources.
func (h *Handler) Start(ctx context.Context) error {
	h.log.Info("SSG handler started")
	return nil
}

// RegisterRoutes registers SSG routes.
func (h *Handler) RegisterRoutes(r chi.Router) {
	h.log.Info("Registering SSG routes")

	// All SSG routes require authentication
	r.Group(func(r chi.Router) {
		r.Use(h.sessionMw)

		// Sites list and create
		r.Get("/ssg/sites", h.HandleListSites)
		r.Get("/ssg/sites/new", h.HandleNewSite)
		r.Post("/ssg/sites", h.HandleCreateSite)

		// All site-specific routes under one Route block
		r.Route("/ssg/sites/{siteID}", func(r chi.Router) {
			// Site CRUD (no siteCtxMw needed - handlers load site directly)
			r.Get("/", h.HandleShowSite)
			r.Get("/edit", h.HandleEditSite)
			r.Post("/", h.HandleUpdateSite)
			r.Post("/delete", h.HandleDeleteSite)

			// Routes that need site context middleware
			r.Group(func(r chi.Router) {
				r.Use(h.siteCtxMw)

				// Sections
				r.Get("/sections", h.HandleListSections)
				r.Get("/sections/new", h.HandleNewSection)
				r.Post("/sections", h.HandleCreateSection)
				r.Get("/sections/{sectionID}", h.HandleShowSection)
				r.Get("/sections/{sectionID}/edit", h.HandleEditSection)
				r.Post("/sections/{sectionID}", h.HandleUpdateSection)
				r.Post("/sections/{sectionID}/delete", h.HandleDeleteSection)

				// Contents
				r.Get("/contents", h.HandleListContents)
				r.Get("/contents/new", h.HandleNewContent)
				r.Post("/contents", h.HandleCreateContent)
				r.Get("/contents/{contentID}", h.HandleShowContent)
				r.Get("/contents/{contentID}/edit", h.HandleEditContent)
				r.Post("/contents/{contentID}", h.HandleUpdateContent)
				r.Post("/contents/{contentID}/delete", h.HandleDeleteContent)

				// Layouts
				r.Get("/layouts", h.HandleListLayouts)
				r.Get("/layouts/new", h.HandleNewLayout)
				r.Post("/layouts", h.HandleCreateLayout)
				r.Get("/layouts/{layoutID}", h.HandleShowLayout)
				r.Get("/layouts/{layoutID}/edit", h.HandleEditLayout)
				r.Post("/layouts/{layoutID}", h.HandleUpdateLayout)
				r.Post("/layouts/{layoutID}/delete", h.HandleDeleteLayout)

				// Tags
				r.Get("/tags", h.HandleListTags)
				r.Get("/tags/new", h.HandleNewTag)
				r.Post("/tags", h.HandleCreateTag)
				r.Get("/tags/{tagID}", h.HandleShowTag)
				r.Get("/tags/{tagID}/edit", h.HandleEditTag)
				r.Post("/tags/{tagID}", h.HandleUpdateTag)
				r.Post("/tags/{tagID}/delete", h.HandleDeleteTag)

				// Params
				r.Get("/params", h.HandleListParams)
				r.Get("/params/new", h.HandleNewParam)
				r.Post("/params", h.HandleCreateParam)
				r.Get("/params/{paramID}", h.HandleShowParam)
				r.Get("/params/{paramID}/edit", h.HandleEditParam)
				r.Post("/params/{paramID}", h.HandleUpdateParam)
				r.Post("/params/{paramID}/delete", h.HandleDeleteParam)

				// Images
				r.Get("/images", h.HandleListImages)
				r.Get("/images/new", h.HandleNewImage)
				r.Post("/images", h.HandleCreateImage)
				r.Get("/images/{imageID}", h.HandleShowImage)
				r.Get("/images/{imageID}/edit", h.HandleEditImage)
				r.Post("/images/{imageID}", h.HandleUpdateImage)
				r.Post("/images/{imageID}/delete", h.HandleDeleteImage)

				// Generation
				r.Post("/generate-markdown", h.HandleGenerateMarkdown)
				r.Post("/generate-html", h.HandleGenerateHTML)
				r.Post("/publish", h.HandlePublish)
			})
		})
	})
}

// PageData holds common page data for templates.
type PageData struct {
	Title       string
	Template    string
	HideNav     bool
	AuthPage    bool
	Site        *Site
	Sites       []*Site
	Section     *Section
	Sections    []*Section
	Content     *Content
	Contents    []*Content
	Layout      *Layout
	Layouts     []*Layout
	Tag         *Tag
	Tags        []*Tag
	Param       *Param
	Params      []*Param
	Image       *Image
	Images      []*Image
	Error       string
	Success     string
	CSRFToken   string
	CurrentPage int
	TotalPages  int
	HasPrev     bool
	HasNext     bool
}

func (h *Handler) render(w http.ResponseWriter, templateName string, data PageData) {
	funcMap := render.MergeFuncMaps(render.FuncMap(), template.FuncMap{
		"add":      func(a, b int) int { return a + b },
		"subtract": func(a, b int) int { return a - b },
		"multiply": func(a, b int) int { return a * b },
	})

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(h.assetsFS,
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

func (h *Handler) renderError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	h.render(w, "error", PageData{
		Title: "Error",
		Error: message,
	})
}

// --- Site Handlers ---

func (h *Handler) HandleListSites(w http.ResponseWriter, r *http.Request) {
	sites, err := h.service.ListSites(r.Context())
	if err != nil {
		h.log.Errorf("Cannot list sites: %v", err)
		h.renderError(w, http.StatusInternalServerError, "Cannot load sites")
		return
	}

	if len(sites) == 0 {
		http.Redirect(w, r, "/ssg/sites/new", http.StatusSeeOther)
		return
	}

	h.render(w, "ssg/sites/list", PageData{
		Title: "Sites",
		Sites: sites,
	})
}

func (h *Handler) HandleNewSite(w http.ResponseWriter, r *http.Request) {
	h.render(w, "ssg/sites/new", PageData{
		Title: "New Site",
	})
}

func (h *Handler) HandleCreateSite(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid form data")
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
		h.render(w, "ssg/sites/new", PageData{
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
		h.render(w, "ssg/sites/new", PageData{
			Title: "New Site",
			Site:  site,
			Error: "Cannot create site directories",
		})
		return
	}

	h.log.Infof("Created site %s with directories", site.Slug)
	http.Redirect(w, r, "/ssg/sites", http.StatusSeeOther)
}

func (h *Handler) HandleShowSite(w http.ResponseWriter, r *http.Request) {
	siteID, err := uuid.Parse(chi.URLParam(r, "siteID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid site ID")
		return
	}

	site, err := h.service.GetSite(r.Context(), siteID)
	if err != nil {
		h.log.Errorf("Cannot get site: %v", err)
		h.renderError(w, http.StatusNotFound, "Site not found")
		return
	}

	h.render(w, "ssg/sites/show", PageData{
		Title: site.Name,
		Site:  site,
	})
}

func (h *Handler) HandleEditSite(w http.ResponseWriter, r *http.Request) {
	siteID, err := uuid.Parse(chi.URLParam(r, "siteID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid site ID")
		return
	}

	site, err := h.service.GetSite(r.Context(), siteID)
	if err != nil {
		h.log.Errorf("Cannot get site: %v", err)
		h.renderError(w, http.StatusNotFound, "Site not found")
		return
	}

	h.render(w, "ssg/sites/edit", PageData{
		Title: "Edit " + site.Name,
		Site:  site,
	})
}

func (h *Handler) HandleUpdateSite(w http.ResponseWriter, r *http.Request) {
	siteID, err := uuid.Parse(chi.URLParam(r, "siteID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid site ID")
		return
	}

	site, err := h.service.GetSite(r.Context(), siteID)
	if err != nil {
		h.log.Errorf("Cannot get site: %v", err)
		h.renderError(w, http.StatusNotFound, "Site not found")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid form data")
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
		h.render(w, "ssg/sites/edit", PageData{
			Title: "Edit " + site.Name,
			Site:  site,
			Error: "Cannot update site",
		})
		return
	}

	http.Redirect(w, r, "/ssg/sites/"+site.ID.String(), http.StatusSeeOther)
}

func (h *Handler) HandleDeleteSite(w http.ResponseWriter, r *http.Request) {
	siteID, err := uuid.Parse(chi.URLParam(r, "siteID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid site ID")
		return
	}

	// Get site first to get the slug for directory deletion
	site, err := h.service.GetSite(r.Context(), siteID)
	if err != nil {
		h.log.Errorf("Cannot get site for deletion: %v", err)
		h.renderError(w, http.StatusNotFound, "Site not found")
		return
	}

	if err := h.service.DeleteSite(r.Context(), siteID); err != nil {
		h.log.Errorf("Cannot delete site: %v", err)
		h.renderError(w, http.StatusInternalServerError, "Cannot delete site")
		return
	}

	// Delete site directories (errors logged but not fatal)
	if err := h.workspace.DeleteSiteDirectories(site.Slug); err != nil {
		h.log.Errorf("Cannot delete site directories: %v", err)
	}

	h.log.Infof("Deleted site %s", site.Slug)
	http.Redirect(w, r, "/ssg/sites", http.StatusSeeOther)
}

// --- Section Handlers ---

func (h *Handler) HandleListSections(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	sections, err := h.service.GetSections(r.Context(), site.ID)
	if err != nil {
		h.log.Errorf("Cannot list sections: %v", err)
		h.renderError(w, http.StatusInternalServerError, "Cannot load sections")
		return
	}

	h.render(w, "ssg/sections/list", PageData{
		Title:    "Sections",
		Site:     site,
		Sections: sections,
	})
}

func (h *Handler) HandleNewSection(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	layouts, _ := h.service.GetLayouts(r.Context(), site.ID)

	h.render(w, "ssg/sections/new", PageData{
		Title:   "New Section",
		Site:    site,
		Layouts: layouts,
	})
}

func (h *Handler) HandleCreateSection(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid form data")
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
		h.render(w, "ssg/sections/new", PageData{
			Title:   "New Section",
			Site:    site,
			Section: section,
			Layouts: layouts,
			Error:   "Cannot create section",
		})
		return
	}

	http.Redirect(w, r, "/ssg/sites/"+site.ID.String()+"/sections/"+section.ID.String(), http.StatusSeeOther)
}

func (h *Handler) HandleShowSection(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	sectionID, err := uuid.Parse(chi.URLParam(r, "sectionID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid section ID")
		return
	}

	section, err := h.service.GetSection(r.Context(), sectionID)
	if err != nil {
		h.log.Errorf("Cannot get section: %v", err)
		h.renderError(w, http.StatusNotFound, "Section not found")
		return
	}

	h.render(w, "ssg/sections/show", PageData{
		Title:   section.Name,
		Site:    site,
		Section: section,
	})
}

func (h *Handler) HandleEditSection(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	sectionID, err := uuid.Parse(chi.URLParam(r, "sectionID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid section ID")
		return
	}

	section, err := h.service.GetSection(r.Context(), sectionID)
	if err != nil {
		h.log.Errorf("Cannot get section: %v", err)
		h.renderError(w, http.StatusNotFound, "Section not found")
		return
	}

	layouts, _ := h.service.GetLayouts(r.Context(), site.ID)

	h.render(w, "ssg/sections/edit", PageData{
		Title:   "Edit " + section.Name,
		Site:    site,
		Section: section,
		Layouts: layouts,
	})
}

func (h *Handler) HandleUpdateSection(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	sectionID, err := uuid.Parse(chi.URLParam(r, "sectionID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid section ID")
		return
	}

	section, err := h.service.GetSection(r.Context(), sectionID)
	if err != nil {
		h.log.Errorf("Cannot get section: %v", err)
		h.renderError(w, http.StatusNotFound, "Section not found")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	section.Name = r.FormValue("name")
	section.Description = r.FormValue("description")
	section.Path = r.FormValue("path")

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
		h.render(w, "ssg/sections/edit", PageData{
			Title:   "Edit " + section.Name,
			Site:    site,
			Section: section,
			Layouts: layouts,
			Error:   "Cannot update section",
		})
		return
	}

	http.Redirect(w, r, "/ssg/sites/"+site.ID.String()+"/sections/"+section.ID.String(), http.StatusSeeOther)
}

func (h *Handler) HandleDeleteSection(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	sectionID, err := uuid.Parse(chi.URLParam(r, "sectionID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid section ID")
		return
	}

	if err := h.service.DeleteSection(r.Context(), sectionID); err != nil {
		h.log.Errorf("Cannot delete section: %v", err)
		h.renderError(w, http.StatusInternalServerError, "Cannot delete section")
		return
	}

	http.Redirect(w, r, "/ssg/sites/"+site.ID.String()+"/sections", http.StatusSeeOther)
}

// --- Content Handlers ---

func (h *Handler) HandleListContents(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	limit := 20
	offset := (page - 1) * limit
	search := r.URL.Query().Get("q")

	contents, total, err := h.service.GetContentWithPagination(r.Context(), site.ID, offset, limit, search)
	if err != nil {
		h.log.Errorf("Cannot list contents: %v", err)
		h.renderError(w, http.StatusInternalServerError, "Cannot load contents")
		return
	}

	totalPages := (total + limit - 1) / limit

	h.render(w, "ssg/contents/list", PageData{
		Title:       "Contents",
		Site:        site,
		Contents:    contents,
		CurrentPage: page,
		TotalPages:  totalPages,
		HasPrev:     page > 1,
		HasNext:     page < totalPages,
	})
}

func (h *Handler) HandleNewContent(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	sections, _ := h.service.GetSections(r.Context(), site.ID)
	tags, _ := h.service.GetTags(r.Context(), site.ID)

	h.render(w, "ssg/contents/new", PageData{
		Title:    "New Content",
		Site:     site,
		Sections: sections,
		Tags:     tags,
	})
}

func (h *Handler) HandleCreateContent(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid form data")
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

	if err := h.service.CreateContent(r.Context(), content); err != nil {
		h.log.Errorf("Cannot create content: %v", err)
		sections, _ := h.service.GetSections(r.Context(), site.ID)
		tags, _ := h.service.GetTags(r.Context(), site.ID)
		h.render(w, "ssg/contents/new", PageData{
			Title:    "New Content",
			Site:     site,
			Content:  content,
			Sections: sections,
			Tags:     tags,
			Error:    "Cannot create content",
		})
		return
	}

	// Handle tags
	tagIDs := r.Form["tag_ids"]
	for _, tagIDStr := range tagIDs {
		if tagID, err := uuid.Parse(tagIDStr); err == nil {
			_ = h.service.AddTagToContentByID(r.Context(), content.ID, tagID)
		}
	}

	http.Redirect(w, r, "/ssg/sites/"+site.ID.String()+"/contents/"+content.ID.String(), http.StatusSeeOther)
}

func (h *Handler) HandleShowContent(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	contentID, err := uuid.Parse(chi.URLParam(r, "contentID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid content ID")
		return
	}

	content, err := h.service.GetContent(r.Context(), contentID)
	if err != nil {
		h.log.Errorf("Cannot get content: %v", err)
		h.renderError(w, http.StatusNotFound, "Content not found")
		return
	}

	// Load tags
	content.Tags, _ = h.service.GetTagsForContent(r.Context(), contentID)

	h.render(w, "ssg/contents/show", PageData{
		Title:   content.Heading,
		Site:    site,
		Content: content,
	})
}

func (h *Handler) HandleEditContent(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	contentID, err := uuid.Parse(chi.URLParam(r, "contentID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid content ID")
		return
	}

	content, err := h.service.GetContent(r.Context(), contentID)
	if err != nil {
		h.log.Errorf("Cannot get content: %v", err)
		h.renderError(w, http.StatusNotFound, "Content not found")
		return
	}

	content.Tags, _ = h.service.GetTagsForContent(r.Context(), contentID)
	sections, _ := h.service.GetSections(r.Context(), site.ID)
	tags, _ := h.service.GetTags(r.Context(), site.ID)

	h.render(w, "ssg/contents/edit", PageData{
		Title:    "Edit " + content.Heading,
		Site:     site,
		Content:  content,
		Sections: sections,
		Tags:     tags,
	})
}

func (h *Handler) HandleUpdateContent(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	contentID, err := uuid.Parse(chi.URLParam(r, "contentID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid content ID")
		return
	}

	content, err := h.service.GetContent(r.Context(), contentID)
	if err != nil {
		h.log.Errorf("Cannot get content: %v", err)
		h.renderError(w, http.StatusNotFound, "Content not found")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid form data")
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
		h.render(w, "ssg/contents/edit", PageData{
			Title:    "Edit " + content.Heading,
			Site:     site,
			Content:  content,
			Sections: sections,
			Tags:     tags,
			Error:    "Cannot update content",
		})
		return
	}

	// Update tags
	_ = h.service.RemoveAllTagsFromContent(r.Context(), content.ID)
	tagIDs := r.Form["tag_ids"]
	for _, tagIDStr := range tagIDs {
		if tagID, err := uuid.Parse(tagIDStr); err == nil {
			_ = h.service.AddTagToContentByID(r.Context(), content.ID, tagID)
		}
	}

	http.Redirect(w, r, "/ssg/sites/"+site.ID.String()+"/contents/"+content.ID.String(), http.StatusSeeOther)
}

func (h *Handler) HandleDeleteContent(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	contentID, err := uuid.Parse(chi.URLParam(r, "contentID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid content ID")
		return
	}

	if err := h.service.DeleteContent(r.Context(), contentID); err != nil {
		h.log.Errorf("Cannot delete content: %v", err)
		h.renderError(w, http.StatusInternalServerError, "Cannot delete content")
		return
	}

	http.Redirect(w, r, "/ssg/sites/"+site.ID.String()+"/contents", http.StatusSeeOther)
}

// --- Layout Handlers ---

func (h *Handler) HandleListLayouts(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	layouts, err := h.service.GetLayouts(r.Context(), site.ID)
	if err != nil {
		h.log.Errorf("Cannot list layouts: %v", err)
		h.renderError(w, http.StatusInternalServerError, "Cannot load layouts")
		return
	}

	h.render(w, "ssg/layouts/list", PageData{
		Title:   "Layouts",
		Site:    site,
		Layouts: layouts,
	})
}

func (h *Handler) HandleNewLayout(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	h.render(w, "ssg/layouts/new", PageData{
		Title: "New Layout",
		Site:  site,
	})
}

func (h *Handler) HandleCreateLayout(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid form data")
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
		h.render(w, "ssg/layouts/new", PageData{
			Title:  "New Layout",
			Site:   site,
			Layout: layout,
			Error:  "Cannot create layout",
		})
		return
	}

	http.Redirect(w, r, "/ssg/sites/"+site.ID.String()+"/layouts/"+layout.ID.String(), http.StatusSeeOther)
}

func (h *Handler) HandleShowLayout(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	layoutID, err := uuid.Parse(chi.URLParam(r, "layoutID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid layout ID")
		return
	}

	layout, err := h.service.GetLayout(r.Context(), layoutID)
	if err != nil {
		h.log.Errorf("Cannot get layout: %v", err)
		h.renderError(w, http.StatusNotFound, "Layout not found")
		return
	}

	h.render(w, "ssg/layouts/show", PageData{
		Title:  layout.Name,
		Site:   site,
		Layout: layout,
	})
}

func (h *Handler) HandleEditLayout(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	layoutID, err := uuid.Parse(chi.URLParam(r, "layoutID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid layout ID")
		return
	}

	layout, err := h.service.GetLayout(r.Context(), layoutID)
	if err != nil {
		h.log.Errorf("Cannot get layout: %v", err)
		h.renderError(w, http.StatusNotFound, "Layout not found")
		return
	}

	h.render(w, "ssg/layouts/edit", PageData{
		Title:  "Edit " + layout.Name,
		Site:   site,
		Layout: layout,
	})
}

func (h *Handler) HandleUpdateLayout(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	layoutID, err := uuid.Parse(chi.URLParam(r, "layoutID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid layout ID")
		return
	}

	layout, err := h.service.GetLayout(r.Context(), layoutID)
	if err != nil {
		h.log.Errorf("Cannot get layout: %v", err)
		h.renderError(w, http.StatusNotFound, "Layout not found")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid form data")
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
		h.render(w, "ssg/layouts/edit", PageData{
			Title:  "Edit " + layout.Name,
			Site:   site,
			Layout: layout,
			Error:  "Cannot update layout",
		})
		return
	}

	http.Redirect(w, r, "/ssg/sites/"+site.ID.String()+"/layouts/"+layout.ID.String(), http.StatusSeeOther)
}

func (h *Handler) HandleDeleteLayout(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	layoutID, err := uuid.Parse(chi.URLParam(r, "layoutID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid layout ID")
		return
	}

	if err := h.service.DeleteLayout(r.Context(), layoutID); err != nil {
		h.log.Errorf("Cannot delete layout: %v", err)
		h.renderError(w, http.StatusInternalServerError, "Cannot delete layout")
		return
	}

	http.Redirect(w, r, "/ssg/sites/"+site.ID.String()+"/layouts", http.StatusSeeOther)
}

// --- Tag Handlers ---

func (h *Handler) HandleListTags(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	tags, err := h.service.GetTags(r.Context(), site.ID)
	if err != nil {
		h.log.Errorf("Cannot list tags: %v", err)
		h.renderError(w, http.StatusInternalServerError, "Cannot load tags")
		return
	}

	h.render(w, "ssg/tags/list", PageData{
		Title: "Tags",
		Site:  site,
		Tags:  tags,
	})
}

func (h *Handler) HandleNewTag(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	h.render(w, "ssg/tags/new", PageData{
		Title: "New Tag",
		Site:  site,
	})
}

func (h *Handler) HandleCreateTag(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid form data")
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
		h.render(w, "ssg/tags/new", PageData{
			Title: "New Tag",
			Site:  site,
			Tag:   tag,
			Error: "Cannot create tag",
		})
		return
	}

	http.Redirect(w, r, "/ssg/sites/"+site.ID.String()+"/tags", http.StatusSeeOther)
}

func (h *Handler) HandleShowTag(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	tagID, err := uuid.Parse(chi.URLParam(r, "tagID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid tag ID")
		return
	}

	tag, err := h.service.GetTag(r.Context(), tagID)
	if err != nil {
		h.log.Errorf("Cannot get tag: %v", err)
		h.renderError(w, http.StatusNotFound, "Tag not found")
		return
	}

	h.render(w, "ssg/tags/show", PageData{
		Title: tag.Name,
		Site:  site,
		Tag:   tag,
	})
}

func (h *Handler) HandleEditTag(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	tagID, err := uuid.Parse(chi.URLParam(r, "tagID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid tag ID")
		return
	}

	tag, err := h.service.GetTag(r.Context(), tagID)
	if err != nil {
		h.log.Errorf("Cannot get tag: %v", err)
		h.renderError(w, http.StatusNotFound, "Tag not found")
		return
	}

	h.render(w, "ssg/tags/edit", PageData{
		Title: "Edit " + tag.Name,
		Site:  site,
		Tag:   tag,
	})
}

func (h *Handler) HandleUpdateTag(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	tagID, err := uuid.Parse(chi.URLParam(r, "tagID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid tag ID")
		return
	}

	tag, err := h.service.GetTag(r.Context(), tagID)
	if err != nil {
		h.log.Errorf("Cannot get tag: %v", err)
		h.renderError(w, http.StatusNotFound, "Tag not found")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid form data")
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
		h.render(w, "ssg/tags/edit", PageData{
			Title: "Edit " + tag.Name,
			Site:  site,
			Tag:   tag,
			Error: "Cannot update tag",
		})
		return
	}

	http.Redirect(w, r, "/ssg/sites/"+site.ID.String()+"/tags", http.StatusSeeOther)
}

func (h *Handler) HandleDeleteTag(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	tagID, err := uuid.Parse(chi.URLParam(r, "tagID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid tag ID")
		return
	}

	if err := h.service.DeleteTag(r.Context(), tagID); err != nil {
		h.log.Errorf("Cannot delete tag: %v", err)
		h.renderError(w, http.StatusInternalServerError, "Cannot delete tag")
		return
	}

	http.Redirect(w, r, "/ssg/sites/"+site.ID.String()+"/tags", http.StatusSeeOther)
}

// --- Param Handlers ---

func (h *Handler) HandleListParams(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	params, err := h.service.GetParams(r.Context(), site.ID)
	if err != nil {
		h.log.Errorf("Cannot list params: %v", err)
		h.renderError(w, http.StatusInternalServerError, "Cannot load params")
		return
	}

	h.render(w, "ssg/params/list", PageData{
		Title:  "Parameters",
		Site:   site,
		Params: params,
	})
}

func (h *Handler) HandleNewParam(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	h.render(w, "ssg/params/new", PageData{
		Title: "New Parameter",
		Site:  site,
	})
}

func (h *Handler) HandleCreateParam(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid form data")
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
		h.render(w, "ssg/params/new", PageData{
			Title: "New Parameter",
			Site:  site,
			Param: param,
			Error: "Cannot create parameter",
		})
		return
	}

	http.Redirect(w, r, "/ssg/sites/"+site.ID.String()+"/params", http.StatusSeeOther)
}

func (h *Handler) HandleShowParam(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	paramID, err := uuid.Parse(chi.URLParam(r, "paramID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid param ID")
		return
	}

	param, err := h.service.GetParam(r.Context(), paramID)
	if err != nil {
		h.log.Errorf("Cannot get param: %v", err)
		h.renderError(w, http.StatusNotFound, "Parameter not found")
		return
	}

	h.render(w, "ssg/params/show", PageData{
		Title: param.Name,
		Site:  site,
		Param: param,
	})
}

func (h *Handler) HandleEditParam(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	paramID, err := uuid.Parse(chi.URLParam(r, "paramID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid param ID")
		return
	}

	param, err := h.service.GetParam(r.Context(), paramID)
	if err != nil {
		h.log.Errorf("Cannot get param: %v", err)
		h.renderError(w, http.StatusNotFound, "Parameter not found")
		return
	}

	h.render(w, "ssg/params/edit", PageData{
		Title: "Edit " + param.Name,
		Site:  site,
		Param: param,
	})
}

func (h *Handler) HandleUpdateParam(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	paramID, err := uuid.Parse(chi.URLParam(r, "paramID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid param ID")
		return
	}

	param, err := h.service.GetParam(r.Context(), paramID)
	if err != nil {
		h.log.Errorf("Cannot get param: %v", err)
		h.renderError(w, http.StatusNotFound, "Parameter not found")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	param.Name = r.FormValue("name")
	param.Description = r.FormValue("description")
	param.Value = r.FormValue("value")
	param.RefKey = r.FormValue("ref_key")

	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			param.UpdatedBy = userID
		}
	}

	if err := h.service.UpdateParam(r.Context(), param); err != nil {
		h.log.Errorf("Cannot update param: %v", err)
		h.render(w, "ssg/params/edit", PageData{
			Title: "Edit " + param.Name,
			Site:  site,
			Param: param,
			Error: "Cannot update parameter",
		})
		return
	}

	http.Redirect(w, r, "/ssg/sites/"+site.ID.String()+"/params", http.StatusSeeOther)
}

func (h *Handler) HandleDeleteParam(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	paramID, err := uuid.Parse(chi.URLParam(r, "paramID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid param ID")
		return
	}

	if err := h.service.DeleteParam(r.Context(), paramID); err != nil {
		h.log.Errorf("Cannot delete param: %v", err)
		h.renderError(w, http.StatusInternalServerError, "Cannot delete parameter")
		return
	}

	http.Redirect(w, r, "/ssg/sites/"+site.ID.String()+"/params", http.StatusSeeOther)
}

// --- Image Handlers ---

func (h *Handler) HandleListImages(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	images, err := h.service.GetImages(r.Context(), site.ID)
	if err != nil {
		h.log.Errorf("Cannot list images: %v", err)
		h.renderError(w, http.StatusInternalServerError, "Cannot load images")
		return
	}

	h.render(w, "ssg/images/list", PageData{
		Title:  "Images",
		Site:   site,
		Images: images,
	})
}

func (h *Handler) HandleNewImage(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	h.render(w, "ssg/images/new", PageData{
		Title: "New Image",
		Site:  site,
	})
}

func (h *Handler) HandleCreateImage(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		h.log.Errorf("Cannot parse multipart form: %v", err)
		h.renderError(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	// Get uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		h.log.Errorf("Cannot get uploaded file: %v", err)
		h.render(w, "ssg/images/new", PageData{
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
		h.renderError(w, http.StatusInternalServerError, "Cannot create images directory")
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
		h.renderError(w, http.StatusInternalServerError, "Cannot save file")
		return
	}
	defer dst.Close()

	// Copy uploaded file
	if _, err := io.Copy(dst, file); err != nil {
		h.log.Errorf("Cannot write file: %v", err)
		h.renderError(w, http.StatusInternalServerError, "Cannot save file")
		return
	}

	// Create image record
	image := NewImage(site.ID, header.Filename, "/images/"+fileName)
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
		h.render(w, "ssg/images/new", PageData{
			Title: "Upload Image",
			Site:  site,
			Error: "Cannot save image record",
		})
		return
	}

	h.log.Infof("Image uploaded: %s", fileName)
	http.Redirect(w, r, "/ssg/sites/"+site.ID.String()+"/images", http.StatusSeeOther)
}

func (h *Handler) HandleShowImage(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	imageID, err := uuid.Parse(chi.URLParam(r, "imageID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid image ID")
		return
	}

	image, err := h.service.GetImage(r.Context(), imageID)
	if err != nil {
		h.log.Errorf("Cannot get image: %v", err)
		h.renderError(w, http.StatusNotFound, "Image not found")
		return
	}

	h.render(w, "ssg/images/show", PageData{
		Title: image.FileName,
		Site:  site,
		Image: image,
	})
}

func (h *Handler) HandleEditImage(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	imageID, err := uuid.Parse(chi.URLParam(r, "imageID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid image ID")
		return
	}

	image, err := h.service.GetImage(r.Context(), imageID)
	if err != nil {
		h.log.Errorf("Cannot get image: %v", err)
		h.renderError(w, http.StatusNotFound, "Image not found")
		return
	}

	h.render(w, "ssg/images/edit", PageData{
		Title: "Edit " + image.FileName,
		Site:  site,
		Image: image,
	})
}

func (h *Handler) HandleUpdateImage(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	imageID, err := uuid.Parse(chi.URLParam(r, "imageID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid image ID")
		return
	}

	image, err := h.service.GetImage(r.Context(), imageID)
	if err != nil {
		h.log.Errorf("Cannot get image: %v", err)
		h.renderError(w, http.StatusNotFound, "Image not found")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid form data")
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
		h.render(w, "ssg/images/edit", PageData{
			Title: "Edit " + image.FileName,
			Site:  site,
			Image: image,
			Error: "Cannot update image",
		})
		return
	}

	http.Redirect(w, r, "/ssg/sites/"+site.ID.String()+"/images", http.StatusSeeOther)
}

func (h *Handler) HandleDeleteImage(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	imageID, err := uuid.Parse(chi.URLParam(r, "imageID"))
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid image ID")
		return
	}

	if err := h.service.DeleteImage(r.Context(), imageID); err != nil {
		h.log.Errorf("Cannot delete image: %v", err)
		h.renderError(w, http.StatusInternalServerError, "Cannot delete image")
		return
	}

	http.Redirect(w, r, "/ssg/sites/"+site.ID.String()+"/images", http.StatusSeeOther)
}

// --- Generation Handlers ---

func (h *Handler) HandleGenerateMarkdown(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	// Get all content with metadata
	contents, err := h.service.GetAllContentWithMeta(r.Context(), site.ID)
	if err != nil {
		h.log.Errorf("Cannot get content for markdown generation: %v", err)
		h.renderError(w, http.StatusInternalServerError, "Cannot load content")
		return
	}

	// Generate markdown files
	result, err := h.generator.GenerateMarkdown(r.Context(), site.Slug, contents)
	if err != nil {
		h.log.Errorf("Markdown generation failed: %v", err)
		h.renderError(w, http.StatusInternalServerError, "Markdown generation failed")
		return
	}

	h.log.Infof("Markdown generation complete: %d files generated", result.FilesGenerated)
	if len(result.Errors) > 0 {
		h.log.Infof("Markdown generation had %d errors", len(result.Errors))
	}

	// Redirect back to site with success message
	http.Redirect(w, r, "/ssg/sites/"+site.ID.String()+"?success=markdown", http.StatusSeeOther)
}

func (h *Handler) HandleGenerateHTML(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	// Get all content with metadata
	contents, err := h.service.GetAllContentWithMeta(r.Context(), site.ID)
	if err != nil {
		h.log.Errorf("Cannot get content for HTML generation: %v", err)
		h.renderError(w, http.StatusInternalServerError, "Cannot load content")
		return
	}

	// Get sections
	sections, err := h.service.GetSections(r.Context(), site.ID)
	if err != nil {
		h.log.Errorf("Cannot get sections for HTML generation: %v", err)
		h.renderError(w, http.StatusInternalServerError, "Cannot load sections")
		return
	}

	// Get params
	params, err := h.service.GetParams(r.Context(), site.ID)
	if err != nil {
		h.log.Errorf("Cannot get params for HTML generation: %v", err)
		// Continue without params
		params = []*Param{}
	}

	// Generate HTML site
	result, err := h.htmlGen.GenerateHTML(r.Context(), site, contents, sections, params)
	if err != nil {
		h.log.Errorf("HTML generation failed: %v", err)
		h.renderError(w, http.StatusInternalServerError, "HTML generation failed")
		return
	}

	h.log.Infof("HTML generation complete: %d pages, %d index pages", result.PagesGenerated, result.IndexPages)
	if len(result.Errors) > 0 {
		h.log.Infof("HTML generation had %d errors", len(result.Errors))
	}

	// Redirect back to site with success message
	http.Redirect(w, r, "/ssg/sites/"+site.ID.String()+"?success=html", http.StatusSeeOther)
}

func (h *Handler) HandlePublish(w http.ResponseWriter, r *http.Request) {
	site := getSiteFromContext(r.Context())
	if site == nil {
		h.renderError(w, http.StatusBadRequest, "Site context required")
		return
	}

	// TODO: Implement publishing with go-git
	// For now, just redirect back with a message
	h.log.Info("Publish requested but not yet implemented", "site", site.Slug)
	http.Redirect(w, r, "/ssg/sites/"+site.ID.String()+"?error=publish_not_implemented", http.StatusSeeOther)
}
