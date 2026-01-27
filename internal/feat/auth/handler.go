package auth

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"html/template"
	"net/http"
	"strings"
	"time"

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

	// Parse only auth templates to avoid collision with SSG templates
	tmpl, err := template.New("").Funcs(funcMap).ParseFS(h.assetsFS,
		"assets/templates/auth/*.html",
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

	r.Get("/signin", h.HandleSignIn)
	r.Post("/signin", h.HandleSignIn)
	r.Get("/login", h.HandleSignIn)
	r.Post("/login", h.HandleSignIn)
	r.Post("/signout", h.HandleSignOut)
	r.Post("/logout", h.HandleSignOut)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(h.sessionMw)
		r.Get("/", h.handleHome)
		r.Get("/change-password", h.HandleChangePassword)
		r.Post("/change-password", h.HandleChangePassword)

		// Admin-only routes (Users CRUD)
		r.Group(func(r chi.Router) {
			r.Use(h.requireAdmin)
			r.Get("/admin/list-users", h.HandleListUsers)
			r.Get("/admin/new-user", h.HandleNewUser)
			r.Post("/admin/create-user", h.HandleCreateUser)
			r.Get("/admin/get-user", h.HandleShowUser)
			r.Get("/admin/edit-user", h.HandleEditUser)
			r.Post("/admin/update-user", h.HandleUpdateUser)
			r.Post("/admin/delete-user", h.HandleDeleteUser)
		})
	})
}

func (h *Handler) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := h.GetCurrentUser(r.Context())
		if err != nil || !user.HasRole(RoleAdmin) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SessionMiddleware returns the session middleware for use by other handlers.
func (h *Handler) SessionMiddleware() func(http.Handler) http.Handler {
	return h.sessionMw
}

func (h *Handler) handleHome(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/ssg/list-sites", http.StatusSeeOther)
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

// HandleSignIn handles both GET and POST for the sign in page.
func (h *Handler) HandleSignIn(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.renderSignInForm(w, "", "")
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		h.renderSignInForm(w, "Email and password are required", email)
		return
	}

	user, err := h.service.Authenticate(r.Context(), email, password)
	if err != nil {
		h.log.Errorf("Authentication failed: %v", err)
		h.renderSignInForm(w, "Invalid email or password", email)
		return
	}

	session, err := h.service.CreateSession(r.Context(), user.ID)
	if err != nil {
		h.log.Errorf("Cannot create session: %v", err)
		h.renderSignInForm(w, "Authentication error", email)
		return
	}

	maxAge := int(h.service.GetSessionTTL().Seconds())
	middleware.SetSessionCookie(w, session.ID, maxAge)

	h.log.Infof("User authenticated: %s", user.ID)

	if user.MustChangePassword {
		http.Redirect(w, r, "/change-password", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/ssg/list-sites", http.StatusSeeOther)
}

func (h *Handler) renderSignInForm(w http.ResponseWriter, errorMsg, email string) {
	data := PageData{
		Title:    "Sign In",
		Template: "signin",
		HideNav:  true,
		AuthPage: true,
		Error:    errorMsg,
		Email:    email,
	}

	if h.tmpl == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `<!DOCTYPE html>
<html>
<head><title>Sign In</title>
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
<h1>Sign In</h1>`
		if errorMsg != "" {
			html += `<div class="error">` + errorMsg + `</div>`
		}
		html += `<form method="POST" action="/signin">
<div class="form-group"><label for="email">Email</label><input type="email" id="email" name="email" value="` + email + `" required></div>
<div class="form-group"><label for="password">Password</label><input type="password" id="password" name="password" required></div>
<button type="submit" class="btn">Sign In</button>
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

// HandleSignOut handles user sign out.
func (h *Handler) HandleSignOut(w http.ResponseWriter, r *http.Request) {
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
	http.Redirect(w, r, "/signin", http.StatusSeeOther)
}

// HandleChangePassword handles both GET and POST for the change password page.
func (h *Handler) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr == "" {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	user, err := h.service.GetUser(r.Context(), userID)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodGet {
		h.renderChangePasswordForm(w, "", user.MustChangePassword)
		return
	}

	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	if currentPassword == "" || newPassword == "" || confirmPassword == "" {
		h.renderChangePasswordForm(w, "All fields are required", user.MustChangePassword)
		return
	}

	if newPassword != confirmPassword {
		h.renderChangePasswordForm(w, "Passwords do not match", user.MustChangePassword)
		return
	}

	if len(newPassword) < 8 {
		h.renderChangePasswordForm(w, "Password must be at least 8 characters", user.MustChangePassword)
		return
	}

	if !user.CheckPassword(currentPassword) {
		h.renderChangePasswordForm(w, "Current password is incorrect", user.MustChangePassword)
		return
	}

	if err := user.UpdatePassword(newPassword); err != nil {
		h.log.Errorf("Cannot update password: %v", err)
		h.renderChangePasswordForm(w, "Cannot update password", user.MustChangePassword)
		return
	}

	user.MustChangePassword = false
	if err := h.service.UpdateUser(r.Context(), user); err != nil {
		h.log.Errorf("Cannot save user: %v", err)
		h.renderChangePasswordForm(w, "Cannot save changes", user.MustChangePassword)
		return
	}

	h.log.Infof("Password changed for user: %s", user.ID)
	http.Redirect(w, r, "/ssg/list-sites", http.StatusSeeOther)
}

func (h *Handler) renderChangePasswordForm(w http.ResponseWriter, errorMsg string, mustChange bool) {
	data := PageData{
		Title:    "Change Password",
		Template: "change-password",
		HideNav:  mustChange,
		AuthPage: true,
		Error:    errorMsg,
	}

	if h.tmpl == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		title := "Change Password"
		if mustChange {
			title = "Password Change Required"
		}
		html := `<!DOCTYPE html>
<html>
<head><title>` + title + `</title>
<style>
* { box-sizing: border-box; }
body { font-family: system-ui, -apple-system, sans-serif; margin: 0; padding: 0; background: #f5f5f5; }
.auth-container { max-width: 400px; margin: 80px auto; padding: 0 20px; }
.card { background: white; border-radius: 8px; padding: 30px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
h1 { margin: 0 0 24px 0; font-size: 1.5rem; text-align: center; }
.error { color: #dc3545; margin-bottom: 15px; }
.info { color: #0d6efd; margin-bottom: 15px; padding: 10px; background: #e7f1ff; border-radius: 4px; }
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
<h1>` + title + `</h1>`
		if mustChange {
			html += `<div class="info">Please change your password to continue.</div>`
		}
		if errorMsg != "" {
			html += `<div class="error">` + errorMsg + `</div>`
		}
		html += `<form method="POST" action="/change-password">
<div class="form-group"><label for="current_password">Current Password</label><input type="password" id="current_password" name="current_password" required></div>
<div class="form-group"><label for="new_password">New Password</label><input type="password" id="new_password" name="new_password" required minlength="8"></div>
<div class="form-group"><label for="confirm_password">Confirm New Password</label><input type="password" id="confirm_password" name="confirm_password" required></div>
<button type="submit" class="btn">Change Password</button>
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

// GetUserName returns the current user's display name from context.
func (h *Handler) GetUserName(ctx context.Context) string {
	user, err := h.GetCurrentUser(ctx)
	if err != nil {
		return ""
	}
	if user.Name != "" {
		return user.Name
	}
	return user.Email
}

// GetUserRoles returns the current user's roles from context.
func (h *Handler) GetUserRoles(ctx context.Context) string {
	user, err := h.GetCurrentUser(ctx)
	if err != nil {
		return ""
	}
	return user.Roles
}

// --- Users CRUD ---

// AdminPageData holds data for admin pages.
type AdminPageData struct {
	Title            string
	HideNav          bool
	AuthPage         bool
	Site             interface{}
	Error            string
	Success          string
	User             *User
	Users            []*User
	GeneratedPass    string
	CurrentUserName  string
	CurrentUserRoles string
}

func (h *Handler) renderAdmin(w http.ResponseWriter, r *http.Request, templateName string, data AdminPageData) {
	funcMap := render.MergeFuncMaps(render.FuncMap(), template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"hasRole": func(roles, role string) bool {
			for _, r := range strings.Split(roles, ",") {
				if strings.TrimSpace(r) == role {
					return true
				}
			}
			return false
		},
	})

	if data.CurrentUserName == "" || data.CurrentUserRoles == "" {
		if user, err := h.GetCurrentUser(r.Context()); err == nil {
			if data.CurrentUserName == "" {
				data.CurrentUserName = user.Name
				if data.CurrentUserName == "" {
					data.CurrentUserName = user.Email
				}
			}
			if data.CurrentUserRoles == "" {
				data.CurrentUserRoles = user.Roles
			}
		}
	}

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

func generatePassword(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:length]
}

func (h *Handler) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.ListUsers(r.Context())
	if err != nil {
		h.log.Errorf("Cannot list users: %v", err)
		h.renderAdmin(w, r, "admin/users/list", AdminPageData{
			Title: "Users",
			Error: "Cannot load users",
		})
		return
	}

	h.renderAdmin(w, r, "admin/users/list", AdminPageData{
		Title: "Users",
		Users: users,
	})
}

func (h *Handler) HandleNewUser(w http.ResponseWriter, r *http.Request) {
	h.renderAdmin(w, r, "admin/users/new", AdminPageData{
		Title:         "New User",
		GeneratedPass: generatePassword(12),
	})
}

func (h *Handler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	email := r.FormValue("email")
	name := r.FormValue("name")
	password := r.FormValue("password")
	roles := strings.Join(r.Form["roles"], ",")
	mustChangePassword := r.FormValue("must_change_password") == "on"

	if email == "" || password == "" {
		h.renderAdmin(w, r, "admin/users/new", AdminPageData{
			Title:         "New User",
			Error:         "Email and password are required",
			GeneratedPass: password,
		})
		return
	}

	_, err := h.service.CreateUser(r.Context(), email, password, name, roles, mustChangePassword)
	if err != nil {
		h.log.Errorf("Cannot create user: %v", err)
		h.renderAdmin(w, r, "admin/users/new", AdminPageData{
			Title:         "New User",
			Error:         "Cannot create user: " + err.Error(),
			GeneratedPass: password,
		})
		return
	}

	http.Redirect(w, r, "/admin/list-users", http.StatusSeeOther)
}

func (h *Handler) HandleShowUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := h.service.GetUser(r.Context(), id)
	if err != nil {
		h.log.Errorf("Cannot get user: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	h.renderAdmin(w, r, "admin/users/show", AdminPageData{
		Title: user.Name,
		User:  user,
	})
}

func (h *Handler) HandleEditUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := h.service.GetUser(r.Context(), id)
	if err != nil {
		h.log.Errorf("Cannot get user: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	h.renderAdmin(w, r, "admin/users/edit", AdminPageData{
		Title: "Edit " + user.Name,
		User:  user,
	})
}

func (h *Handler) HandleUpdateUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.FormValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := h.service.GetUser(r.Context(), id)
	if err != nil {
		h.log.Errorf("Cannot get user: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	user.Email = r.FormValue("email")
	user.Name = r.FormValue("name")
	user.Status = r.FormValue("status")
	// Only update roles if user is not admin (admin role is immutable)
	if !user.HasRole(RoleAdmin) {
		roles := r.Form["roles"]
		user.Roles = strings.Join(roles, ",")
	}
	user.MustChangePassword = r.FormValue("must_change_password") == "on"
	user.UpdatedAt = time.Now()

	newPassword := r.FormValue("new_password")
	if newPassword != "" {
		if err := user.UpdatePassword(newPassword); err != nil {
			h.log.Errorf("Cannot update password: %v", err)
			h.renderAdmin(w, r, "admin/users/edit", AdminPageData{
				Title: "Edit " + user.Name,
				User:  user,
				Error: "Cannot update password",
			})
			return
		}
	}

	if err := h.service.UpdateUser(r.Context(), user); err != nil {
		h.log.Errorf("Cannot update user: %v", err)
		errMsg := "Cannot save user"
		if err == ErrCannotChangeAdmin {
			errMsg = "Cannot change admin role"
		}
		h.renderAdmin(w, r, "admin/users/edit", AdminPageData{
			Title: "Edit " + user.Name,
			User:  user,
			Error: errMsg,
		})
		return
	}

	http.Redirect(w, r, "/admin/get-user?id="+id.String(), http.StatusSeeOther)
}

func (h *Handler) HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.FormValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Prevent deleting yourself
	currentUserIDStr := middleware.GetUserID(r.Context())
	if currentUserIDStr == id.String() {
		http.Error(w, "Cannot delete yourself", http.StatusBadRequest)
		return
	}

	// Prevent deleting the admin user
	user, err := h.service.GetUser(r.Context(), id)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if user.HasRole(RoleAdmin) {
		http.Error(w, "Cannot delete the admin user", http.StatusForbidden)
		return
	}

	if err := h.service.DeleteUser(r.Context(), id); err != nil {
		h.log.Errorf("Cannot delete user: %v", err)
		http.Error(w, "Cannot delete user", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/list-users", http.StatusSeeOther)
}
