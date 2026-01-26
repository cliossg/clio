package ssg

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cliossg/clio/pkg/cl/config"
	"github.com/cliossg/clio/pkg/cl/logger"
	"github.com/go-chi/chi/v5"
)

// PreviewHandler serves static HTML files for site preview.
// URL format: /preview/{siteSlug}/*
type PreviewHandler struct {
	workspace *Workspace
	cfg       *config.Config
	log       logger.Logger
}

// NewPreviewHandler creates a new preview handler.
func NewPreviewHandler(cfg *config.Config, log logger.Logger) *PreviewHandler {
	return &PreviewHandler{
		workspace: NewWorkspace(cfg.SSG.SitesBasePath),
		cfg:       cfg,
		log:       log,
	}
}

// RegisterRoutes registers preview routes.
func (h *PreviewHandler) RegisterRoutes(r chi.Router) {
	h.log.Info("Registering preview routes")
	r.Get("/preview/{siteSlug}/*", h.ServePreview)
	r.Get("/preview/{siteSlug}", h.ServePreview)
}

// ServePreview serves static files from the generated HTML directory.
func (h *PreviewHandler) ServePreview(w http.ResponseWriter, r *http.Request) {
	siteSlug := chi.URLParam(r, "siteSlug")
	if siteSlug == "" {
		http.Error(w, "Site slug required", http.StatusBadRequest)
		return
	}

	// Get the path after /preview/{siteSlug}/
	requestPath := chi.URLParam(r, "*")
	if requestPath == "" {
		requestPath = "index.html"
	}

	// Get site HTML path
	htmlPath := h.workspace.GetHTMLPath(siteSlug)

	// Build full file path
	fullPath := filepath.Join(htmlPath, requestPath)

	// Security check: prevent directory traversal
	cleanPath := filepath.Clean(fullPath)
	if !strings.HasPrefix(cleanPath, filepath.Clean(htmlPath)) {
		http.Error(w, "Invalid path", http.StatusForbidden)
		return
	}

	// Check if file exists
	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Try with index.html for directory-style URLs
			indexPath := filepath.Join(cleanPath, "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				cleanPath = indexPath
			} else {
				http.NotFound(w, r)
				return
			}
		} else {
			http.Error(w, "Error reading file", http.StatusInternalServerError)
			return
		}
	} else if info.IsDir() {
		// Serve index.html for directories
		cleanPath = filepath.Join(cleanPath, "index.html")
		if _, err := os.Stat(cleanPath); err != nil {
			http.NotFound(w, r)
			return
		}
	}

	h.log.Debugf("Serving preview: %s -> %s", requestPath, cleanPath)
	http.ServeFile(w, r, cleanPath)
}

// ServeImages serves images from the workspace images directory.
// URL format: /preview/{siteSlug}/images/*
func (h *PreviewHandler) RegisterImageRoutes(r chi.Router) {
	r.Get("/siteimages/{siteSlug}/*", h.ServeImage)
}

// ServeImage serves image files from the site's images directory.
func (h *PreviewHandler) ServeImage(w http.ResponseWriter, r *http.Request) {
	siteSlug := chi.URLParam(r, "siteSlug")
	if siteSlug == "" {
		http.Error(w, "Site slug required", http.StatusBadRequest)
		return
	}

	// Get the path after /siteimages/{siteSlug}/
	requestPath := chi.URLParam(r, "*")
	if requestPath == "" {
		http.NotFound(w, r)
		return
	}

	// Get site images path
	imagesPath := h.workspace.GetImagesPath(siteSlug)

	// Build full file path
	fullPath := filepath.Join(imagesPath, requestPath)

	// Security check: prevent directory traversal
	cleanPath := filepath.Clean(fullPath)
	if !strings.HasPrefix(cleanPath, filepath.Clean(imagesPath)) {
		http.Error(w, "Invalid path", http.StatusForbidden)
		return
	}

	// Check if file exists
	if _, err := os.Stat(cleanPath); err != nil {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, cleanPath)
}
