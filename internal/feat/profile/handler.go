package profile

import (
	"context"
	"embed"
	"encoding/json"
	"html/template"
	"net/http"
	"strings"

	"github.com/cliossg/clio/pkg/cl/config"
	"github.com/cliossg/clio/pkg/cl/logger"
	"github.com/cliossg/clio/pkg/cl/render"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type UserProvider interface {
	GetCurrentUserID(ctx context.Context) (uuid.UUID, error)
	GetCurrentUserProfileID(ctx context.Context) (*uuid.UUID, error)
	GetCurrentUserRoles(ctx context.Context) string
	GetUserName(ctx context.Context) string
	SetUserProfile(ctx context.Context, userID, profileID uuid.UUID) error
}

type Handler struct {
	service      Service
	userProvider UserProvider
	sessionMw    func(http.Handler) http.Handler
	assetsFS     embed.FS
	cfg          *config.Config
	log          logger.Logger
}

func NewHandler(service Service, userProvider UserProvider, sessionMw func(http.Handler) http.Handler, assetsFS embed.FS, cfg *config.Config, log logger.Logger) *Handler {
	return &Handler{
		service:      service,
		userProvider: userProvider,
		sessionMw:    sessionMw,
		assetsFS:     assetsFS,
		cfg:          cfg,
		log:          log,
	}
}

func (h *Handler) Start(ctx context.Context) error {
	h.log.Info("Profile handler started")
	return nil
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	h.log.Info("Registering profile routes")

	r.Group(func(r chi.Router) {
		r.Use(h.sessionMw)
		r.Get("/profile", h.HandleShowProfile)
		r.Get("/profile/edit", h.HandleEditProfile)
		r.Post("/profile/create", h.HandleCreateProfile)
		r.Post("/profile/update", h.HandleUpdateProfile)
	})
}

type PageData struct {
	Title            string
	Template         string
	HideNav          bool
	AuthPage         bool
	Site             interface{}
	Profile          *Profile
	SocialLinksMap   map[string]string
	CurrentUserRoles string
	CurrentUserName  string
	Error            string
	Success          string
}

func (h *Handler) HandleShowProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	profileID, err := h.userProvider.GetCurrentUserProfileID(ctx)
	if err != nil {
		h.log.Errorf("Cannot get user profile ID: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	roles := h.userProvider.GetCurrentUserRoles(ctx)
	userName := h.userProvider.GetUserName(ctx)

	if profileID == nil {
		h.renderTemplate(w, "profile/new.html", PageData{
			Title:            "Create Profile",
			Template:         "profile/new.html",
			CurrentUserRoles: roles,
			CurrentUserName:  userName,
		})
		return
	}

	profile, err := h.service.GetProfile(ctx, *profileID)
	if err != nil {
		h.log.Errorf("Cannot get profile: %v", err)
		http.Error(w, "Profile not found", http.StatusNotFound)
		return
	}

	h.renderTemplate(w, "profile/show.html", PageData{
		Title:            "My Profile",
		Template:         "profile/show.html",
		Profile:          profile,
		SocialLinksMap:   parseSocialLinksJSON(profile.SocialLinks),
		CurrentUserRoles: roles,
		CurrentUserName:  userName,
	})
}

func (h *Handler) HandleEditProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	profileID, err := h.userProvider.GetCurrentUserProfileID(ctx)
	if err != nil || profileID == nil {
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}

	profile, err := h.service.GetProfile(ctx, *profileID)
	if err != nil {
		h.log.Errorf("Cannot get profile: %v", err)
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}

	roles := h.userProvider.GetCurrentUserRoles(ctx)
	userName := h.userProvider.GetUserName(ctx)

	h.renderTemplate(w, "profile/edit.html", PageData{
		Title:            "Edit Profile",
		Template:         "profile/edit.html",
		Profile:          profile,
		SocialLinksMap:   parseSocialLinksJSON(profile.SocialLinks),
		CurrentUserRoles: roles,
		CurrentUserName:  userName,
	})
}

func (h *Handler) HandleCreateProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, err := h.userProvider.GetCurrentUserID(ctx)
	if err != nil {
		h.log.Errorf("Cannot get user ID: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	slug := normalizeSlug(r.FormValue("slug"))
	name := strings.TrimSpace(r.FormValue("name"))
	surname := strings.TrimSpace(r.FormValue("surname"))
	bio := strings.TrimSpace(r.FormValue("bio"))
	socialLinks := buildSocialLinksJSON(r)

	profile, err := h.service.CreateProfile(ctx, slug, name, surname, bio, socialLinks, "", userID.String())
	if err != nil {
		h.log.Errorf("Cannot create profile: %v", err)
		roles := h.userProvider.GetCurrentUserRoles(ctx)
		userName := h.userProvider.GetUserName(ctx)
		h.renderTemplate(w, "profile/new.html", PageData{
			Title:            "Create Profile",
			Template:         "profile/new.html",
			CurrentUserRoles: roles,
			CurrentUserName:  userName,
			Error:            "Cannot create profile: " + err.Error(),
		})
		return
	}

	err = h.userProvider.SetUserProfile(ctx, userID, profile.ID)
	if err != nil {
		h.log.Errorf("Cannot link profile to user: %v", err)
	}

	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

func (h *Handler) HandleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	profileID, err := h.userProvider.GetCurrentUserProfileID(ctx)
	if err != nil || profileID == nil {
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}

	profile, err := h.service.GetProfile(ctx, *profileID)
	if err != nil {
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}

	userID, _ := h.userProvider.GetCurrentUserID(ctx)

	profile.Slug = normalizeSlug(r.FormValue("slug"))
	profile.Name = strings.TrimSpace(r.FormValue("name"))
	profile.Surname = strings.TrimSpace(r.FormValue("surname"))
	profile.Bio = strings.TrimSpace(r.FormValue("bio"))
	profile.SocialLinks = buildSocialLinksJSON(r)
	profile.UpdatedBy = userID.String()

	err = h.service.UpdateProfile(ctx, profile)
	if err != nil {
		h.log.Errorf("Cannot update profile: %v", err)
		roles := h.userProvider.GetCurrentUserRoles(ctx)
		userName := h.userProvider.GetUserName(ctx)
		h.renderTemplate(w, "profile/edit.html", PageData{
			Title:            "Edit Profile",
			Template:         "profile/edit.html",
			Profile:          profile,
			SocialLinksMap:   parseSocialLinksJSON(profile.SocialLinks),
			CurrentUserRoles: roles,
			CurrentUserName:  userName,
			Error:            "Cannot update profile",
		})
		return
	}

	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

func (h *Handler) renderTemplate(w http.ResponseWriter, templateName string, data PageData) {
	funcMap := render.MergeFuncMaps(render.FuncMap(), template.FuncMap{
		"hasRole": func(roles, role string) bool {
			for _, r := range splitRoles(roles) {
				if r == role {
					return true
				}
			}
			return false
		},
	})

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(h.assetsFS,
		"assets/templates/base.html",
		"assets/templates/"+templateName,
	)
	if err != nil {
		h.log.Errorf("Template parse error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		h.log.Errorf("Template execute error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func splitRoles(roles string) []string {
	if roles == "" {
		return nil
	}
	var result []string
	for _, r := range split(roles, ",") {
		if trimmed := trim(r); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// socialPlatforms defines all supported social platforms
var socialPlatforms = []string{
	"facebook", "youtube", "instagram", "x", "tiktok", "linkedin", "github",
	"whatsapp", "telegram", "reddit", "messenger", "snapchat", "pinterest",
	"tumblr", "discord", "twitch", "signal", "viber", "line", "kakaotalk",
	"wechat", "qq", "douyin", "kuaishou", "weibo", "xiaohongshu", "bilibili",
	"zhihu", "vk", "odnoklassniki", "mastodon", "bluesky", "threads", "flickr",
	"vimeo", "dailymotion", "quora",
}

type socialLink struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
}

func parseSocialLinksJSON(jsonStr string) map[string]string {
	result := make(map[string]string)
	if jsonStr == "" || jsonStr == "[]" {
		return result
	}
	var links []socialLink
	if err := json.Unmarshal([]byte(jsonStr), &links); err != nil {
		return result
	}
	for _, link := range links {
		result[link.Platform] = link.URL
	}
	return result
}

func buildSocialLinksJSON(r *http.Request) string {
	var links []socialLink
	for _, platform := range socialPlatforms {
		url := strings.TrimSpace(r.FormValue("social_" + platform))
		if url != "" {
			links = append(links, socialLink{Platform: platform, URL: url})
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

func split(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep[0] {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trim(s string) string {
	start := 0
	end := len(s)
	for start < end && s[start] == ' ' {
		start++
	}
	for end > start && s[end-1] == ' ' {
		end--
	}
	return s[start:end]
}
