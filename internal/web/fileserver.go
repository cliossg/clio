package web

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/cliossg/clio/pkg/cl/logger"
	"github.com/go-chi/chi/v5"
)

const (
	staticAssetsPath = "assets/static"
	staticURLPrefix  = "/static"
)

type FileServer struct {
	assetsFS embed.FS
	log      logger.Logger
}

func NewFileServer(assetsFS embed.FS, log logger.Logger) *FileServer {
	return &FileServer{
		assetsFS: assetsFS,
		log:      log,
	}
}

func (s *FileServer) RegisterRoutes(r chi.Router) {
	s.log.Infof("Registering file server: %s -> %s", staticURLPrefix, staticAssetsPath)

	staticFS, err := fs.Sub(s.assetsFS, staticAssetsPath)
	if err != nil {
		s.log.Errorf("Error creating static files sub-filesystem: %v", err)
		return
	}

	handler := http.StripPrefix(staticURLPrefix+"/", http.FileServer(http.FS(staticFS)))
	r.Handle(staticURLPrefix+"/*", handler)
	s.log.Info("File server registered successfully")
}
