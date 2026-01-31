package main

import (
	"context"
	"embed"
	"os"
	"os/signal"
	"syscall"

	"github.com/cliossg/clio/internal/feat/auth"
	"github.com/cliossg/clio/internal/feat/profile"
	"github.com/cliossg/clio/internal/feat/ssg"
	"github.com/cliossg/clio/internal/web"
	"github.com/cliossg/clio/pkg/cl/app"
	"github.com/cliossg/clio/pkg/cl/config"
	"github.com/cliossg/clio/pkg/cl/database"
	"github.com/cliossg/clio/pkg/cl/logger"
	"github.com/cliossg/clio/pkg/cl/middleware"
	"github.com/go-chi/chi/v5"
)

//go:embed assets/migrations/sqlite/*.sql
var migrationsFS embed.FS

//go:embed assets/templates/*.html assets/templates/*/*.html assets/templates/*/*/*.html
var templatesFS embed.FS

//go:embed assets/static
var staticFS embed.FS

//go:embed assets/ssg
var ssgAssetsFS embed.FS

func main() {
	ctx := context.Background()

	cfg := config.Load()
	log := logger.New(cfg.Log.Level)

	log.Infof("Starting Clio [%s mode]", cfg.Env)
	log.Infof("Database: %s", cfg.Database.Path)
	log.Infof("Sites: %s", cfg.SSG.SitesBasePath)

	db := database.New(migrationsFS, cfg, log)
	db.SetMigrationPath("assets/migrations/sqlite")

	authService := auth.NewService(db, cfg, log)
	profileService := profile.NewService(db, cfg, log)
	ssgWorkspace := ssg.NewWorkspace(cfg.SSG.SitesBasePath)
	ssgHTMLGen := ssg.NewHTMLGenerator(ssgWorkspace, ssgAssetsFS)
	ssgService := ssg.NewService(db, ssgHTMLGen, cfg, log)

	optionalSessionMw := middleware.OptionalSession(authService)
	requiredSessionMw := middleware.Session(authService)
	siteCtxMw := ssg.SiteContextMiddleware(ssgService, log)

	authHandler := auth.NewHandler(authService, profileService, optionalSessionMw, templatesFS, cfg, log)
	profileHandler := profile.NewHandler(profileService, authService, requiredSessionMw, templatesFS, cfg, log)
	ssgHandler := ssg.NewHandler(ssgService, profileService, siteCtxMw, requiredSessionMw, templatesFS, ssgAssetsFS, cfg, log)
	previewServer := ssg.NewPreviewServer(ssgService, cfg, log)

	authSeeder := auth.NewSeeder(authService, profileService, templatesFS, log)
	if cfg.Credentials.Path != "" {
		authSeeder.SetCredentialsPath(cfg.Credentials.Path)
	}

	ssgSeeder := ssg.NewSeeder(ssgService, profileService, log)

	router := chi.NewRouter()
	middleware.DefaultStack(router)

	fileServer := web.NewFileServer(staticFS, log)

	deps := []any{db, authService, profileService, ssgService, authSeeder, ssgSeeder, authHandler, profileHandler, ssgHandler, previewServer, fileServer}

	starts, stops, registrars := app.Setup(ctx, router, deps...)
	if err := app.Start(ctx, log, starts, stops, registrars, router); err != nil {
		log.Errorf("Startup failed: %v", err)
		os.Exit(1)
	}

	go app.Serve(router, cfg.Server.Addr)
	log.Infof("Server listening on %s", cfg.Server.Addr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	app.Stop(ctx, log, stops)
	log.Info("Server stopped")
}
