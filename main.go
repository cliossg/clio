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
	"github.com/cliossg/clio/pkg/cl/git"
	"github.com/cliossg/clio/pkg/cl/llm"
	"github.com/cliossg/clio/pkg/cl/logger"
	"github.com/cliossg/clio/pkg/cl/middleware"
	"github.com/go-chi/chi/v5"
)

//go:embed assets
var assetsFS embed.FS

func main() {
	ctx := context.Background()

	cfg := config.Load()
	log := logger.New(cfg.Log.Level)

	log.Infof("Starting Clio [%s mode]", cfg.Env)
	log.Infof("Database: %s", cfg.Database.Path)
	log.Infof("Sites: %s", cfg.SSG.SitesBasePath)

	db := database.New(assetsFS, cfg, log)
	db.SetMigrationPath("assets/migrations/sqlite")

	authService := auth.NewService(db, cfg, log)
	profileService := profile.NewService(db, cfg, log)
	ssgWorkspace := ssg.NewWorkspace(cfg.SSG.SitesBasePath)
	ssgHTMLGen := ssg.NewHTMLGenerator(ssgWorkspace, assetsFS)
	ssgService := ssg.NewService(db, ssgHTMLGen, cfg, log)
	gitClient := git.NewClient(log)
	ssgPublisher := ssg.NewPublisher(ssgWorkspace, gitClient)
	llmClient := llm.NewClient(cfg.LLM.APIKey, cfg.LLM.Model, cfg.LLM.Temperature)

	optionalSessionMw := middleware.OptionalSession(authService)
	requiredSessionMw := middleware.Session(authService)
	siteCtxMw := ssg.SiteContextMiddleware(ssgService, log)

	authHandler := auth.NewHandler(authService, profileService, optionalSessionMw, assetsFS, cfg, log)
	profileHandler := profile.NewHandler(profileService, authService, requiredSessionMw, assetsFS, cfg, log)
	ssgHandler := ssg.NewHandler(ssgService, profileService, ssgWorkspace, ssgHTMLGen, ssgPublisher, llmClient, siteCtxMw, requiredSessionMw, assetsFS, cfg, log)
	previewServer := ssg.NewPreviewServer(ssgService, cfg, log)

	authSeeder := auth.NewSeeder(authService, profileService, assetsFS, log)
	if cfg.Credentials.Path != "" {
		authSeeder.SetCredentialsPath(cfg.Credentials.Path)
	}

	ssgSeeder := ssg.NewSeeder(ssgService, profileService, log)
	ssgScheduler := ssg.NewScheduler(ssgService, ssgHTMLGen, ssgPublisher, log)

	router := chi.NewRouter()
	middleware.DefaultStack(router)

	fileServer := web.NewFileServer(assetsFS, log)

	deps := []any{db, authService, profileService, ssgService, authSeeder, ssgSeeder, ssgScheduler, authHandler, profileHandler, ssgHandler, previewServer, fileServer}

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
