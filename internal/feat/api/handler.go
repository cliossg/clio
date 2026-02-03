package api

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/cliossg/clio/internal/feat/ssg"
	"github.com/cliossg/clio/pkg/cl/config"
	"github.com/cliossg/clio/pkg/cl/logger"
	"github.com/cliossg/clio/pkg/cl/middleware"
	"github.com/cliossg/clio/pkg/cl/render"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler implements the REST API and token management UI.
type Handler struct {
	apiService  Service
	ssgService  ssg.Service
	workspace   *ssg.Workspace
	htmlGen     *ssg.HTMLGenerator
	publisher   *ssg.Publisher
	tokenAuthMw func(http.Handler) http.Handler
	sessionMw   func(http.Handler) http.Handler
	templatesFS embed.FS
	cfg         *config.Config
	log         logger.Logger
}

// NewHandler creates a new API handler.
func NewHandler(
	apiService Service,
	ssgService ssg.Service,
	workspace *ssg.Workspace,
	htmlGen *ssg.HTMLGenerator,
	publisher *ssg.Publisher,
	tokenAuthMw func(http.Handler) http.Handler,
	sessionMw func(http.Handler) http.Handler,
	templatesFS embed.FS,
	cfg *config.Config,
	log logger.Logger,
) *Handler {
	return &Handler{
		apiService:  apiService,
		ssgService:  ssgService,
		workspace:   workspace,
		htmlGen:     htmlGen,
		publisher:   publisher,
		tokenAuthMw: tokenAuthMw,
		sessionMw:   sessionMw,
		templatesFS: templatesFS,
		cfg:         cfg,
		log:         log,
	}
}

// Start initializes the handler.
func (h *Handler) Start(ctx context.Context) error {
	h.log.Info("API handler started")
	return nil
}

// RegisterRoutes registers API and token management routes.
func (h *Handler) RegisterRoutes(r chi.Router) {
	h.log.Info("Registering API routes")

	// REST API routes (token auth)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(h.tokenAuthMw)

		// Tokens
		r.Post("/tokens", h.APICreateToken)
		r.Get("/tokens", h.APIListTokens)
		r.Delete("/tokens/{id}", h.APIDeleteToken)

		// Sites
		r.Get("/sites", h.APIListSites)
		r.Get("/sites/{id}", h.APIGetSite)

		// Posts
		r.Get("/sites/{id}/posts", h.APIListPosts)
		r.Get("/sites/{id}/posts/{post_id}", h.APIGetPost)
		r.Post("/sites/{id}/posts", h.APICreatePost)
		r.Put("/sites/{id}/posts/{post_id}", h.APIUpdatePost)
		r.Delete("/sites/{id}/posts/{post_id}", h.APIDeletePost)

		// Generation & Publishing
		r.Post("/sites/{id}/generate", h.APIGenerate)
		r.Post("/sites/{id}/publish", h.APIPublish)
		r.Post("/sites/{id}/backup", h.APIBackup)
	})

	// Token management UI (session auth)
	r.Group(func(r chi.Router) {
		r.Use(h.sessionMw)
		r.Get("/api/tokens", h.HandleListTokens)
		r.Get("/api/new-token", h.HandleNewToken)
		r.Post("/api/create-token", h.HandleCreateToken)
		r.Post("/api/delete-token", h.HandleDeleteToken)
	})
}

// --- JSON helpers ---

func jsonOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

func jsonCreated(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(data)
}

// --- REST API: Tokens ---

func (h *Handler) APICreateToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	userID, err := uuid.Parse(GetUserIDFromContext(r.Context()))
	if err != nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized", "Invalid user")
		return
	}

	rawToken, token, err := h.apiService.CreateToken(r.Context(), userID, req.Name)
	if err != nil {
		h.log.Errorf("Cannot create API token: %v", err)
		jsonError(w, http.StatusInternalServerError, "internal_error", "Cannot create token")
		return
	}

	jsonCreated(w, map[string]any{
		"token": rawToken,
		"id":    token.ID,
		"name":  token.Name,
	})
}

func (h *Handler) APIListTokens(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(GetUserIDFromContext(r.Context()))
	if err != nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized", "Invalid user")
		return
	}

	tokens, err := h.apiService.ListTokens(r.Context(), userID)
	if err != nil {
		h.log.Errorf("Cannot list API tokens: %v", err)
		jsonError(w, http.StatusInternalServerError, "internal_error", "Cannot list tokens")
		return
	}

	jsonOK(w, map[string]any{"tokens": tokens})
}

func (h *Handler) APIDeleteToken(w http.ResponseWriter, r *http.Request) {
	tokenID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid_id", "Invalid token ID")
		return
	}

	if err := h.apiService.DeleteToken(r.Context(), tokenID); err != nil {
		h.log.Errorf("Cannot delete API token: %v", err)
		jsonError(w, http.StatusInternalServerError, "internal_error", "Cannot delete token")
		return
	}

	jsonOK(w, map[string]string{"status": "deleted"})
}

// --- REST API: Sites ---

func (h *Handler) APIListSites(w http.ResponseWriter, r *http.Request) {
	sites, err := h.ssgService.ListSites(r.Context())
	if err != nil {
		h.log.Errorf("Cannot list sites: %v", err)
		jsonError(w, http.StatusInternalServerError, "internal_error", "Cannot list sites")
		return
	}

	jsonOK(w, map[string]any{"sites": sites})
}

func (h *Handler) APIGetSite(w http.ResponseWriter, r *http.Request) {
	siteID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid_id", "Invalid site ID")
		return
	}

	site, err := h.ssgService.GetSite(r.Context(), siteID)
	if err != nil {
		jsonError(w, http.StatusNotFound, "not_found", "Site not found")
		return
	}

	jsonOK(w, site)
}

// --- REST API: Posts ---

func (h *Handler) APIListPosts(w http.ResponseWriter, r *http.Request) {
	siteID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid_id", "Invalid site ID")
		return
	}

	contents, err := h.ssgService.GetAllContentWithMeta(r.Context(), siteID)
	if err != nil {
		h.log.Errorf("Cannot list posts: %v", err)
		jsonError(w, http.StatusInternalServerError, "internal_error", "Cannot list posts")
		return
	}

	jsonOK(w, map[string]any{"posts": contents})
}

func (h *Handler) APIGetPost(w http.ResponseWriter, r *http.Request) {
	postID, err := uuid.Parse(chi.URLParam(r, "post_id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid_id", "Invalid post ID")
		return
	}

	content, err := h.ssgService.GetContentWithMeta(r.Context(), postID)
	if err != nil {
		jsonError(w, http.StatusNotFound, "not_found", "Post not found")
		return
	}

	jsonOK(w, content)
}

func (h *Handler) APICreatePost(w http.ResponseWriter, r *http.Request) {
	siteID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid_id", "Invalid site ID")
		return
	}

	var req struct {
		SectionID   string `json:"section_id"`
		Heading     string `json:"heading"`
		Body        string `json:"body"`
		Summary     string `json:"summary"`
		Kind        string `json:"kind"`
		Draft       *bool  `json:"draft"`
		Featured    bool   `json:"featured"`
		Series      string `json:"series"`
		SeriesOrder int    `json:"series_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.Heading == "" {
		jsonError(w, http.StatusBadRequest, "validation_error", "Heading is required")
		return
	}

	var sectionID uuid.UUID
	if req.SectionID != "" {
		sectionID, err = uuid.Parse(req.SectionID)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid_id", "Invalid section ID")
			return
		}
	}

	content := ssg.NewContent(siteID, sectionID, req.Heading, req.Body)
	content.Summary = req.Summary
	content.Kind = req.Kind
	if content.Kind == "" {
		content.Kind = "post"
	}
	if req.Draft != nil {
		content.Draft = *req.Draft
	}
	content.Featured = req.Featured
	content.Series = req.Series
	content.SeriesOrder = req.SeriesOrder

	userIDStr := GetUserIDFromContext(r.Context())
	if userID, err := uuid.Parse(userIDStr); err == nil {
		content.UserID = userID
		content.CreatedBy = userID
		content.UpdatedBy = userID
	}

	if err := h.ssgService.CreateContent(r.Context(), content); err != nil {
		h.log.Errorf("Cannot create post: %v", err)
		jsonError(w, http.StatusInternalServerError, "internal_error", "Cannot create post")
		return
	}

	jsonCreated(w, content)
}

func (h *Handler) APIUpdatePost(w http.ResponseWriter, r *http.Request) {
	postID, err := uuid.Parse(chi.URLParam(r, "post_id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid_id", "Invalid post ID")
		return
	}

	existing, err := h.ssgService.GetContent(r.Context(), postID)
	if err != nil {
		jsonError(w, http.StatusNotFound, "not_found", "Post not found")
		return
	}

	var req struct {
		SectionID   *string `json:"section_id"`
		Heading     *string `json:"heading"`
		Body        *string `json:"body"`
		Summary     *string `json:"summary"`
		Kind        *string `json:"kind"`
		Draft       *bool   `json:"draft"`
		Featured    *bool   `json:"featured"`
		Series      *string `json:"series"`
		SeriesOrder *int    `json:"series_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.Heading != nil {
		existing.Heading = *req.Heading
	}
	if req.Body != nil {
		existing.Body = *req.Body
	}
	if req.Summary != nil {
		existing.Summary = *req.Summary
	}
	if req.Kind != nil {
		existing.Kind = *req.Kind
	}
	if req.Draft != nil {
		existing.Draft = *req.Draft
	}
	if req.Featured != nil {
		existing.Featured = *req.Featured
	}
	if req.Series != nil {
		existing.Series = *req.Series
	}
	if req.SeriesOrder != nil {
		existing.SeriesOrder = *req.SeriesOrder
	}
	if req.SectionID != nil {
		if sid, err := uuid.Parse(*req.SectionID); err == nil {
			existing.SectionID = sid
		}
	}

	existing.UpdatedAt = time.Now()
	userIDStr := GetUserIDFromContext(r.Context())
	if userID, err := uuid.Parse(userIDStr); err == nil {
		existing.UpdatedBy = userID
	}

	if err := h.ssgService.UpdateContent(r.Context(), existing); err != nil {
		h.log.Errorf("Cannot update post: %v", err)
		jsonError(w, http.StatusInternalServerError, "internal_error", "Cannot update post")
		return
	}

	jsonOK(w, existing)
}

func (h *Handler) APIDeletePost(w http.ResponseWriter, r *http.Request) {
	postID, err := uuid.Parse(chi.URLParam(r, "post_id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid_id", "Invalid post ID")
		return
	}

	if err := h.ssgService.DeleteContent(r.Context(), postID); err != nil {
		h.log.Errorf("Cannot delete post: %v", err)
		jsonError(w, http.StatusInternalServerError, "internal_error", "Cannot delete post")
		return
	}

	jsonOK(w, map[string]string{"status": "deleted"})
}

// --- REST API: Generate, Publish, Backup ---

func (h *Handler) APIGenerate(w http.ResponseWriter, r *http.Request) {
	site, err := h.getSiteFromParam(w, r)
	if err != nil {
		return
	}

	result, err := h.generateHTML(r.Context(), site)
	if err != nil {
		h.log.Errorf("HTML generation failed: %v", err)
		jsonError(w, http.StatusInternalServerError, "generation_error", err.Error())
		return
	}

	jsonOK(w, map[string]any{
		"status":          "generated",
		"pages_generated": result.PagesGenerated,
		"index_pages":     result.IndexPages,
		"author_pages":    result.AuthorPages,
		"errors":          len(result.Errors),
	})
}

func (h *Handler) APIPublish(w http.ResponseWriter, r *http.Request) {
	site, err := h.getSiteFromParam(w, r)
	if err != nil {
		return
	}

	// Generate HTML first
	_, err = h.generateHTML(r.Context(), site)
	if err != nil {
		h.log.Errorf("HTML generation failed during publish: %v", err)
		jsonError(w, http.StatusInternalServerError, "generation_error", err.Error())
		return
	}

	// Get publish settings
	publishCfg, err := h.getPublishConfig(r.Context(), site.ID, "ssg.publish.repo.url", "ssg.publish.auth.token", "ssg.publish.branch", "gh-pages")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "config_error", err.Error())
		return
	}

	publishResult, err := h.publisher.Publish(r.Context(), publishCfg, site.Slug)
	if err != nil {
		h.log.Errorf("Publish failed: %v", err)
		jsonError(w, http.StatusInternalServerError, "publish_error", "Publish to git failed")
		return
	}

	if publishResult.NoChanges {
		jsonOK(w, map[string]any{"status": "no_changes"})
		return
	}

	site.LastPublishedAt = timePtr(time.Now())
	_ = h.ssgService.UpdateSite(r.Context(), site)

	jsonOK(w, map[string]any{
		"status":      "published",
		"commit_hash": publishResult.CommitHash,
		"commit_url":  publishResult.CommitURL,
		"added":       publishResult.Added,
		"modified":    publishResult.Modified,
		"deleted":     publishResult.Deleted,
	})
}

func (h *Handler) APIBackup(w http.ResponseWriter, r *http.Request) {
	site, err := h.getSiteFromParam(w, r)
	if err != nil {
		return
	}

	// Generate markdown
	contents, err := h.ssgService.GetAllContentWithMeta(r.Context(), site.ID)
	if err != nil {
		h.log.Errorf("Cannot get content for backup: %v", err)
		jsonError(w, http.StatusInternalServerError, "internal_error", "Cannot load content")
		return
	}

	generator := ssg.NewGenerator(h.workspace)
	mdResult, err := generator.GenerateMarkdown(r.Context(), site.Slug, contents)
	if err != nil {
		h.log.Errorf("Markdown generation failed: %v", err)
		jsonError(w, http.StatusInternalServerError, "generation_error", "Markdown generation failed")
		return
	}

	h.log.Infof("Markdown generation complete: %d files", mdResult.FilesGenerated)

	// Push to backup repo
	backupCfg, err := h.getPublishConfig(r.Context(), site.ID, "ssg.backup.repo.url", "ssg.backup.auth.token", "ssg.backup.branch", "main")
	if err != nil {
		// No backup repo configured, just return markdown result
		jsonOK(w, map[string]any{
			"status":          "markdown_generated",
			"files_generated": mdResult.FilesGenerated,
			"errors":          len(mdResult.Errors),
		})
		return
	}

	backupResult, err := h.publisher.Backup(r.Context(), backupCfg, site.Slug)
	if err != nil {
		h.log.Errorf("Backup to git failed: %v", err)
		jsonError(w, http.StatusInternalServerError, "backup_error", "Backup to git failed")
		return
	}

	if backupResult.NoChanges {
		jsonOK(w, map[string]any{"status": "no_changes"})
		return
	}

	jsonOK(w, map[string]any{
		"status":      "backed_up",
		"commit_hash": backupResult.CommitHash,
		"commit_url":  backupResult.CommitURL,
		"added":       backupResult.Added,
		"modified":    backupResult.Modified,
		"deleted":     backupResult.Deleted,
	})
}

// --- Internal helpers ---

func (h *Handler) getSiteFromParam(w http.ResponseWriter, r *http.Request) (*ssg.Site, error) {
	siteID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid_id", "Invalid site ID")
		return nil, err
	}

	site, err := h.ssgService.GetSite(r.Context(), siteID)
	if err != nil {
		jsonError(w, http.StatusNotFound, "not_found", "Site not found")
		return nil, err
	}

	return site, nil
}

func (h *Handler) generateHTML(ctx context.Context, site *ssg.Site) (*ssg.GenerateHTMLResult, error) {
	contents, err := h.ssgService.GetAllContentWithMeta(ctx, site.ID)
	if err != nil {
		return nil, fmt.Errorf("cannot load content: %w", err)
	}

	sections, err := h.ssgService.GetSections(ctx, site.ID)
	if err != nil {
		return nil, fmt.Errorf("cannot load sections: %w", err)
	}

	layouts, err := h.ssgService.GetLayouts(ctx, site.ID)
	if err != nil {
		layouts = []*ssg.Layout{}
	}

	params, err := h.ssgService.GetSettings(ctx, site.ID)
	if err != nil {
		params = []*ssg.Setting{}
	}

	contributors, err := h.ssgService.GetContributors(ctx, site.ID)
	if err != nil {
		contributors = []*ssg.Contributor{}
	}

	userAuthors := h.ssgService.BuildUserAuthorsMap(ctx, contents, contributors)

	return h.htmlGen.GenerateHTML(ctx, site, contents, sections, layouts, params, contributors, userAuthors)
}

func (h *Handler) getPublishConfig(ctx context.Context, siteID uuid.UUID, repoKey, tokenKey, branchKey, defaultBranch string) (ssg.PublishConfig, error) {
	repoURL, _ := h.ssgService.GetSettingByRefKey(ctx, siteID, repoKey)
	if repoURL == nil || repoURL.Value == "" {
		return ssg.PublishConfig{}, fmt.Errorf("repository URL not configured (%s)", repoKey)
	}

	authToken, _ := h.ssgService.GetSettingByRefKey(ctx, siteID, tokenKey)
	branch, _ := h.ssgService.GetSettingByRefKey(ctx, siteID, branchKey)
	commitName, _ := h.ssgService.GetSettingByRefKey(ctx, siteID, "ssg.git.commit.user.name")
	commitEmail, _ := h.ssgService.GetSettingByRefKey(ctx, siteID, "ssg.git.commit.user.email")

	branchValue := defaultBranch
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

	return ssg.PublishConfig{
		RepoURL:     repoURL.Value,
		Branch:      branchValue,
		AuthToken:   authTokenValue,
		CommitName:  commitNameValue,
		CommitEmail: commitEmailValue,
		UseSSH:      useSSH,
	}, nil
}

func timePtr(t time.Time) *time.Time {
	return &t
}

// --- Token Management UI ---

type tokenPageData struct {
	Title            string
	Template         string
	CurrentUserName  string
	CurrentUserRoles string
	Site             interface{}
	AuthPage         bool
	Tokens           []*APIToken
	Token            *APIToken
	RawToken         string
	Error            string
	Success          string
	HideNav          bool
}

func (h *Handler) renderToken(w http.ResponseWriter, r *http.Request, templateName string, data tokenPageData) {
	funcMap := render.MergeFuncMaps(render.FuncMap(), template.FuncMap{
		"hasRole": func(roles, role string) bool {
			for _, r := range strings.Split(roles, ",") {
				if strings.TrimSpace(r) == role {
					return true
				}
			}
			return false
		},
	})

	if data.CurrentUserName == "" {
		data.CurrentUserName = middleware.GetUserName(r.Context())
	}
	if data.CurrentUserRoles == "" {
		data.CurrentUserRoles = middleware.GetUserRoles(r.Context())
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

func (h *Handler) HandleListTokens(w http.ResponseWriter, r *http.Request) {
	userIDStr := middleware.GetUserID(r.Context())
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	tokens, err := h.apiService.ListTokens(r.Context(), userID)
	if err != nil {
		h.log.Errorf("Cannot list tokens: %v", err)
		tokens = []*APIToken{}
	}

	h.renderToken(w, r, "api/tokens/list", tokenPageData{
		Title:  "API Tokens",
		Tokens: tokens,
	})
}

func (h *Handler) HandleNewToken(w http.ResponseWriter, r *http.Request) {
	h.renderToken(w, r, "api/tokens/new", tokenPageData{
		Title: "New API Token",
	})
}

func (h *Handler) HandleCreateToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderToken(w, r, "api/tokens/new", tokenPageData{
			Title: "New API Token",
			Error: "Invalid form data",
		})
		return
	}

	userIDStr := middleware.GetUserID(r.Context())
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		name = "Unnamed token"
	}

	rawToken, token, err := h.apiService.CreateToken(r.Context(), userID, name)
	if err != nil {
		h.log.Errorf("Cannot create token: %v", err)
		h.renderToken(w, r, "api/tokens/new", tokenPageData{
			Title: "New API Token",
			Error: "Cannot create token",
		})
		return
	}

	h.renderToken(w, r, "api/tokens/show", tokenPageData{
		Title:    "Token Created",
		Token:    token,
		RawToken: rawToken,
	})
}

func (h *Handler) HandleDeleteToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/api/tokens", http.StatusSeeOther)
		return
	}

	tokenID, err := uuid.Parse(r.FormValue("id"))
	if err != nil {
		http.Redirect(w, r, "/api/tokens", http.StatusSeeOther)
		return
	}

	if err := h.apiService.DeleteToken(r.Context(), tokenID); err != nil {
		h.log.Errorf("Cannot delete token: %v", err)
	}

	http.Redirect(w, r, "/api/tokens", http.StatusSeeOther)
}
