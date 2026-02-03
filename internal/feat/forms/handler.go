package forms

import (
	"context"
	"embed"
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
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

// Handler implements form submission and dashboard UI.
type Handler struct {
	formsService   Service
	ssgService     ssg.Service
	sessionMw      func(http.Handler) http.Handler
	templatesFS    embed.FS
	cfg            *config.Config
	log            logger.Logger
	limiter        *rateLimiter
	allowedOrigins []string
}

// NewHandler creates a new forms handler.
func NewHandler(
	formsService Service,
	ssgService ssg.Service,
	sessionMw func(http.Handler) http.Handler,
	templatesFS embed.FS,
	cfg *config.Config,
	log logger.Logger,
) *Handler {
	return &Handler{
		formsService: formsService,
		ssgService:   ssgService,
		sessionMw:    sessionMw,
		templatesFS:  templatesFS,
		cfg:          cfg,
		log:          log,
		limiter:      newRateLimiter(5), // default, updated on Start
	}
}

// Start initializes the forms handler.
func (h *Handler) Start(ctx context.Context) error {
	h.log.Info("Forms handler started")

	// Load settings from all sites to configure rate limit and origins
	h.loadGlobalSettings(ctx)
	return nil
}

// SubmitHandler returns an http.Handler for the public form submission API.
// This is mounted on the preview server so it's accessible via tunnel.
func (h *Handler) SubmitHandler() http.Handler {
	mux := chi.NewRouter()
	mux.Use(h.corsMiddleware)
	mux.Use(h.rateLimitMiddleware)
	mux.Post("/submit", h.HandleSubmit)
	return mux
}

func (h *Handler) loadGlobalSettings(ctx context.Context) {
	sites, err := h.ssgService.ListSites(ctx)
	if err != nil {
		return
	}
	for _, site := range sites {
		if rl, _ := h.ssgService.GetSettingByRefKey(ctx, site.ID, "ssg.forms.rate_limit"); rl != nil && rl.Value != "" {
			if n, err := strconv.Atoi(rl.Value); err == nil && n > 0 {
				h.limiter = newRateLimiter(n)
			}
		}
		if origins, _ := h.ssgService.GetSettingByRefKey(ctx, site.ID, "ssg.forms.allowed_origins"); origins != nil && origins.Value != "" {
			for _, o := range strings.Split(origins.Value, ",") {
				o = strings.TrimSpace(o)
				if o != "" {
					h.allowedOrigins = append(h.allowedOrigins, o)
				}
			}
		}
	}
}

// RegisterRoutes registers dashboard routes on the main server router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	h.log.Info("Registering forms dashboard routes")

	r.Group(func(r chi.Router) {
		r.Use(h.sessionMw)
		r.Get("/ssg/list-messages", h.HandleListMessages)
		r.Get("/ssg/get-message", h.HandleGetMessage)
		r.Post("/ssg/mark-message-read", h.HandleMarkRead)
		r.Post("/ssg/delete-message", h.HandleDeleteMessage)
	})
}

// --- Public submission endpoint ---

// HandleSubmit processes a form submission from the public internet.
func (h *Handler) HandleSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		// Fallback: also handles url-encoded forms
		if err2 := r.ParseForm(); err2 != nil {
			h.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid form data"})
			return
		}
	}

	// Honeypot check
	if r.FormValue("_honeypot") != "" {
		// Silently accept to fool bots
		h.jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	siteIDStr := r.FormValue("_site")
	siteID, err := uuid.Parse(siteIDStr)
	if err != nil {
		h.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid site"})
		return
	}

	// Verify site exists and forms are enabled
	site, err := h.ssgService.GetSite(r.Context(), siteID)
	if err != nil || site == nil {
		h.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid site"})
		return
	}

	enabled, _ := h.ssgService.GetSettingByRefKey(r.Context(), siteID, "ssg.forms.enabled")
	if enabled == nil || enabled.Value != "true" {
		h.jsonResponse(w, http.StatusForbidden, map[string]string{"error": "forms not enabled"})
		return
	}

	message := strings.TrimSpace(r.FormValue("message"))
	if message == "" {
		h.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "message is required"})
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	if email != "" && !strings.Contains(email, "@") {
		h.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid email"})
		return
	}

	sub := &FormSubmission{
		ID:        uuid.New(),
		SiteID:    siteID,
		FormType:  r.FormValue("_form"),
		Name:      strings.TrimSpace(r.FormValue("name")),
		Email:     email,
		Message:   message,
		IPAddress: extractIP(r),
		UserAgent: r.UserAgent(),
		CreatedAt: time.Now(),
	}
	if sub.FormType == "" {
		sub.FormType = "contact"
	}

	if err := h.formsService.CreateSubmission(r.Context(), sub); err != nil {
		h.log.Errorf("Cannot save form submission: %v", err)
		h.jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to save"})
		return
	}

	h.log.Infof("Form submission received from %s for site %s", sub.Email, siteIDStr)

	if accept := r.Header.Get("Accept"); strings.Contains(accept, "application/json") {
		h.jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	redirectURL := r.Header.Get("Referer")
	if redirectURL == "" {
		redirectURL = "/"
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// --- Dashboard handlers ---

type messagesPageData struct {
	Title            string
	Template         string
	CurrentUserName  string
	CurrentUserRoles string
	Site             *ssg.Site
	HideNav          bool
	AuthPage         bool
	Error            string
	Success          string
	Messages         []*FormSubmission
	Message          *FormSubmission
	UnreadCount      int64
}

func (h *Handler) HandleListMessages(w http.ResponseWriter, r *http.Request) {
	siteIDStr := r.URL.Query().Get("site_id")
	siteID, err := uuid.Parse(siteIDStr)
	if err != nil {
		http.Redirect(w, r, "/ssg/list-sites", http.StatusSeeOther)
		return
	}

	site, err := h.ssgService.GetSite(r.Context(), siteID)
	if err != nil {
		http.Redirect(w, r, "/ssg/list-sites", http.StatusSeeOther)
		return
	}

	messages, err := h.formsService.ListSubmissions(r.Context(), siteID)
	if err != nil {
		h.log.Errorf("Cannot list messages: %v", err)
		messages = []*FormSubmission{}
	}

	unread, _ := h.formsService.CountUnread(r.Context(), siteID)

	h.renderPage(w, r, "ssg/messages/list", messagesPageData{
		Title:       "Messages",
		Site:        site,
		Messages:    messages,
		UnreadCount: unread,
	})
}

func (h *Handler) HandleGetMessage(w http.ResponseWriter, r *http.Request) {
	siteIDStr := r.URL.Query().Get("site_id")
	siteID, err := uuid.Parse(siteIDStr)
	if err != nil {
		http.Redirect(w, r, "/ssg/list-sites", http.StatusSeeOther)
		return
	}

	site, err := h.ssgService.GetSite(r.Context(), siteID)
	if err != nil {
		http.Redirect(w, r, "/ssg/list-sites", http.StatusSeeOther)
		return
	}

	msgID, err := uuid.Parse(r.URL.Query().Get("id"))
	if err != nil {
		http.Redirect(w, r, "/ssg/list-messages?site_id="+siteIDStr, http.StatusSeeOther)
		return
	}

	msg, err := h.formsService.GetSubmission(r.Context(), msgID)
	if err != nil {
		http.Redirect(w, r, "/ssg/list-messages?site_id="+siteIDStr, http.StatusSeeOther)
		return
	}

	// Auto-mark as read when viewing
	if !msg.IsRead() {
		_ = h.formsService.MarkRead(r.Context(), msgID)
		now := time.Now()
		msg.ReadAt = &now
	}

	h.renderPage(w, r, "ssg/messages/show", messagesPageData{
		Title:   "Message from " + msg.Name,
		Site:    site,
		Message: msg,
	})
}

func (h *Handler) HandleMarkRead(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/ssg/list-sites", http.StatusSeeOther)
		return
	}

	siteID := r.FormValue("site_id")
	msgID, err := uuid.Parse(r.FormValue("id"))
	if err != nil {
		http.Redirect(w, r, "/ssg/list-messages?site_id="+siteID, http.StatusSeeOther)
		return
	}

	if err := h.formsService.MarkRead(r.Context(), msgID); err != nil {
		h.log.Errorf("Cannot mark message as read: %v", err)
	}

	http.Redirect(w, r, "/ssg/get-message?id="+msgID.String()+"&site_id="+siteID, http.StatusSeeOther)
}

func (h *Handler) HandleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/ssg/list-sites", http.StatusSeeOther)
		return
	}

	siteID := r.FormValue("site_id")
	msgID, err := uuid.Parse(r.FormValue("id"))
	if err != nil {
		http.Redirect(w, r, "/ssg/list-messages?site_id="+siteID, http.StatusSeeOther)
		return
	}

	if err := h.formsService.DeleteSubmission(r.Context(), msgID); err != nil {
		h.log.Errorf("Cannot delete message: %v", err)
	}

	http.Redirect(w, r, "/ssg/list-messages?site_id="+siteID, http.StatusSeeOther)
}

// --- Rendering helpers ---

func (h *Handler) renderPage(w http.ResponseWriter, r *http.Request, templateName string, data messagesPageData) {
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

func (h *Handler) jsonResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
