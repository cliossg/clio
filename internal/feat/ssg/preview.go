package ssg

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cliossg/clio/pkg/cl/config"
	"github.com/cliossg/clio/pkg/cl/logger"
)

type PreviewServer struct {
	service   Service
	workspace *Workspace
	server    *http.Server
	cfg       *config.Config
	log       logger.Logger
}

func NewPreviewServer(service Service, cfg *config.Config, log logger.Logger) *PreviewServer {
	return &PreviewServer{
		service:   service,
		workspace: NewWorkspace(cfg.SSG.SitesBasePath),
		cfg:       cfg,
		log:       log,
	}
}

func (s *PreviewServer) Start(ctx context.Context) error {
	s.server = &http.Server{
		Addr:    s.cfg.SSG.PreviewAddr,
		Handler: s,
	}

	go func() {
		s.log.Infof("Preview server listening on %s (subdomain mode: <site>.localhost%s)", s.cfg.SSG.PreviewAddr, s.cfg.SSG.PreviewAddr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.Errorf("Preview server error: %v", err)
		}
	}()

	return nil
}

func (s *PreviewServer) Stop(ctx context.Context) error {
	if s.server != nil {
		s.log.Info("Stopping preview server")
		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *PreviewServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	siteSlug := s.extractSiteSlug(r.Host)
	if siteSlug == "" {
		http.Error(w, "Invalid host. Use <site>.localhost:3000", http.StatusBadRequest)
		return
	}

	if err := s.service.GenerateHTMLForSite(r.Context(), siteSlug); err != nil {
		s.log.Errorf("Failed to generate HTML for site %s: %v", siteSlug, err)
	}

	requestPath := r.URL.Path

	basePath := s.getBasePath(r.Context(), siteSlug)
	if basePath != "/" && strings.HasPrefix(requestPath, basePath) {
		requestPath = strings.TrimPrefix(requestPath, strings.TrimSuffix(basePath, "/"))
	}

	if requestPath == "/" || requestPath == "" {
		requestPath = "/index.html"
	}

	if strings.HasPrefix(requestPath, "/static/") {
		s.serveStatic(w, r, siteSlug, requestPath)
		return
	}

	if strings.HasPrefix(requestPath, "/images/") {
		s.serveImage(w, r, siteSlug, strings.TrimPrefix(requestPath, "/images/"))
		return
	}

	s.serveHTML(w, r, siteSlug, requestPath)
}

func (s *PreviewServer) extractSiteSlug(host string) string {
	host = strings.Split(host, ":")[0]

	if !strings.HasSuffix(host, ".localhost") {
		return ""
	}

	slug := strings.TrimSuffix(host, ".localhost")
	if slug == "" || slug == "localhost" {
		return ""
	}

	return slug
}

func (s *PreviewServer) serveHTML(w http.ResponseWriter, r *http.Request, siteSlug, requestPath string) {
	htmlPath := s.workspace.GetHTMLPath(siteSlug)
	fullPath := filepath.Join(htmlPath, requestPath)

	cleanPath := filepath.Clean(fullPath)
	if !strings.HasPrefix(cleanPath, filepath.Clean(htmlPath)) {
		http.Error(w, "Invalid path", http.StatusForbidden)
		return
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
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
		cleanPath = filepath.Join(cleanPath, "index.html")
		if _, err := os.Stat(cleanPath); err != nil {
			http.NotFound(w, r)
			return
		}
	}

	http.ServeFile(w, r, cleanPath)
}

func (s *PreviewServer) serveStatic(w http.ResponseWriter, r *http.Request, siteSlug, requestPath string) {
	htmlPath := s.workspace.GetHTMLPath(siteSlug)
	fullPath := filepath.Join(htmlPath, requestPath)

	cleanPath := filepath.Clean(fullPath)
	if !strings.HasPrefix(cleanPath, filepath.Clean(htmlPath)) {
		http.Error(w, "Invalid path", http.StatusForbidden)
		return
	}

	if _, err := os.Stat(cleanPath); err != nil {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, cleanPath)
}

func (s *PreviewServer) serveImage(w http.ResponseWriter, r *http.Request, siteSlug, imagePath string) {
	imagesPath := s.workspace.GetImagesPath(siteSlug)
	fullPath := filepath.Join(imagesPath, imagePath)

	cleanPath := filepath.Clean(fullPath)
	if !strings.HasPrefix(cleanPath, filepath.Clean(imagesPath)) {
		http.Error(w, "Invalid path", http.StatusForbidden)
		return
	}

	if _, err := os.Stat(cleanPath); err != nil {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, cleanPath)
}

func (s *PreviewServer) getBasePath(ctx context.Context, siteSlug string) string {
	site, err := s.service.GetSiteBySlug(ctx, siteSlug)
	if err != nil || site == nil {
		return "/"
	}

	param, err := s.service.GetSettingByRefKey(ctx, site.ID, "ssg.site.base_path")
	if err != nil || param == nil || param.Value == "" {
		return "/"
	}

	return param.Value
}
