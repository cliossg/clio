package auth

import (
	"context"
	"embed"
	"html/template"
	"net/http"

	"github.com/cliossg/clio/pkg/cl/config"
	"github.com/cliossg/clio/pkg/cl/logger"
	"github.com/cliossg/clio/pkg/cl/middleware"
	"github.com/cliossg/clio/pkg/cl/render"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler handles authentication routes.
type Handler struct {
	service   Service
	sessionMw func(http.Handler) http.Handler
	assetsFS  embed.FS
	tmpl      *template.Template
	cfg       *config.Config
	log       logger.Logger
}

// NewHandler creates a new auth handler.
func NewHandler(service Service, sessionMw func(http.Handler) http.Handler, assetsFS embed.FS, cfg *config.Config, log logger.Logger) *Handler {
	return &Handler{
		service:   service,
		sessionMw: sessionMw,
		assetsFS:  assetsFS,
		cfg:       cfg,
		log:       log,
	}
}

// Start initializes templates and other resources.
func (h *Handler) Start(ctx context.Context) error {
	funcMap := render.MergeFuncMaps(render.FuncMap(), template.FuncMap{
		"add": func(a, b int) int { return a + b },
	})

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(h.assetsFS,
		"assets/templates/*.html",
		"assets/templates/*/*.html",
	)
	if err != nil {
		return err
	}
	h.tmpl = tmpl

	h.log.Info("Auth handler started")
	return nil
}

// RegisterRoutes registers authentication routes.
func (h *Handler) RegisterRoutes(r chi.Router) {
	h.log.Info("Registering auth routes")

	r.Get("/login", h.HandleLogin)
	r.Post("/login", h.HandleLogin)
	r.Post("/logout", h.HandleLogout)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(h.sessionMw)
		r.Get("/", h.handleHome)
	})
}

// SessionMiddleware returns the session middleware for use by other handlers.
func (h *Handler) SessionMiddleware() func(http.Handler) http.Handler {
	return h.sessionMw
}

func (h *Handler) handleHome(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/ssg/sites", http.StatusSeeOther)
}

// PageData holds common page data for templates.
type PageData struct {
	Title    string
	Template string
	HideNav  bool
	AuthPage bool
	Error    string
	Success  string
	Email    string
	Site     interface{}
}

// HandleLogin handles both GET and POST for the login page.
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.renderLoginForm(w, "", "")
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		h.renderLoginForm(w, "Email and password are required", email)
		return
	}

	user, err := h.service.Authenticate(r.Context(), email, password)
	if err != nil {
		h.log.Errorf("Authentication failed: %v", err)
		h.renderLoginForm(w, "Invalid email or password", email)
		return
	}

	session, err := h.service.CreateSession(r.Context(), user.ID)
	if err != nil {
		h.log.Errorf("Cannot create session: %v", err)
		h.renderLoginForm(w, "Authentication error", email)
		return
	}

	maxAge := int(h.service.GetSessionTTL().Seconds())
	middleware.SetSessionCookie(w, session.ID, maxAge)

	h.log.Infof("User authenticated: %s", user.ID)
	http.Redirect(w, r, "/ssg/sites", http.StatusSeeOther)
}

func (h *Handler) renderLoginForm(w http.ResponseWriter, errorMsg, email string) {
	data := PageData{
		Title:    "Login",
		Template: "login",
		HideNav:  true,
		AuthPage: true,
		Error:    errorMsg,
		Email:    email,
	}

	if h.tmpl == nil {
		// Fallback: render a simple HTML form
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `<!DOCTYPE html>
<html>
<head><title>Login</title>
<style>
* { box-sizing: border-box; }
body { font-family: system-ui, -apple-system, sans-serif; margin: 0; padding: 0; background: #f5f5f5; }
.auth-container { max-width: 360px; margin: 80px auto; padding: 0 20px; }
.card { background: white; border-radius: 8px; padding: 30px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
h1 { margin: 0 0 24px 0; font-size: 1.5rem; text-align: center; }
.error { color: #dc3545; margin-bottom: 15px; }
.form-group { margin-bottom: 15px; }
.form-group label { display: block; margin-bottom: 5px; font-weight: 500; }
.form-group input { width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px; }
.btn { display: block; width: 100%; padding: 8px 16px; background: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer; margin-top: 8px; }
.btn:hover { background: #0056b3; }
</style>
</head>
<body>
<div class="auth-container">
<div class="card">
<h1>Login</h1>`
		if errorMsg != "" {
			html += `<div class="error">` + errorMsg + `</div>`
		}
		html += `<form method="POST" action="/login">
<div class="form-group"><label for="email">Email</label><input type="email" id="email" name="email" value="` + email + `" required></div>
<div class="form-group"><label for="password">Password</label><input type="password" id="password" name="password" required></div>
<button type="submit" class="btn">Login</button>
</form>
</div>
</div>
</body>
</html>`
		w.Write([]byte(html))
		return
	}

	if err := h.tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		h.log.Errorf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandleLogout handles user logout.
func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := middleware.GetSessionID(r.Context())
	if sessionID != "" {
		if err := h.service.DeleteSession(r.Context(), sessionID); err != nil {
			h.log.Errorf("Error deleting session: %v", err)
		}
	}

	middleware.ClearSessionCookie(w)
	h.log.Info("User signed out")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// GetCurrentUser returns the current user from the context.
func (h *Handler) GetCurrentUser(ctx context.Context) (*User, error) {
	userIDStr := middleware.GetUserID(ctx)
	if userIDStr == "" {
		return nil, ErrUserNotFound
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return h.service.GetUser(ctx, userID)
}
