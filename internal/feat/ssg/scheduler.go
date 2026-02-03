package ssg

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/cliossg/clio/pkg/cl/logger"
)

type Scheduler struct {
	service   Service
	htmlGen   *HTMLGenerator
	publisher *Publisher
	log       logger.Logger
	stop      chan struct{}
	mu        sync.Mutex
	running   bool
}

func NewScheduler(service Service, htmlGen *HTMLGenerator, publisher *Publisher, log logger.Logger) *Scheduler {
	return &Scheduler{
		service:   service,
		htmlGen:   htmlGen,
		publisher: publisher,
		log:       log,
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	sites, err := s.service.ListSites(ctx)
	if err != nil {
		s.log.Errorf("Scheduler: cannot list sites: %v", err)
		return nil
	}

	for _, site := range sites {
		enabled, _ := s.service.GetSettingByRefKey(ctx, site.ID, "ssg.scheduled.publish.enabled")
		if enabled != nil && enabled.Value == "true" {
			intervalSetting, _ := s.service.GetSettingByRefKey(ctx, site.ID, "ssg.scheduled.publish.interval")
			interval := time.Hour
			if intervalSetting != nil && intervalSetting.Value != "" {
				if parsed, err := time.ParseDuration(intervalSetting.Value); err == nil && parsed >= time.Minute {
					interval = parsed
				}
			}

			s.mu.Lock()
			if !s.running {
				s.stop = make(chan struct{})
				s.running = true
				go s.run(ctx, interval)
				s.log.Infof("Scheduler: started with interval %s", interval)
			}
			s.mu.Unlock()
			return nil
		}
	}

	s.log.Info("Scheduler: no sites with scheduling enabled")
	return nil
}

func (s *Scheduler) Stop(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		close(s.stop)
		s.running = false
		s.log.Info("Scheduler: stopped")
	}
	return nil
}

func (s *Scheduler) run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkAllSites(ctx)
		case <-s.stop:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) checkAllSites(ctx context.Context) {
	sites, err := s.service.ListSites(ctx)
	if err != nil {
		s.log.Errorf("Scheduler: cannot list sites: %v", err)
		return
	}

	for _, site := range sites {
		s.checkAndPublish(ctx, site)
	}
}

func (s *Scheduler) checkAndPublish(ctx context.Context, site *Site) {
	enabled, _ := s.service.GetSettingByRefKey(ctx, site.ID, "ssg.scheduled.publish.enabled")
	if enabled == nil || enabled.Value != "true" {
		return
	}

	contents, err := s.service.GetAllContentWithMeta(ctx, site.ID)
	if err != nil {
		s.log.Errorf("Scheduler: cannot get content for site %s: %v", site.Slug, err)
		return
	}

	if !hasPendingContent(contents, site.LastPublishedAt) {
		return
	}

	s.log.Infof("Scheduler: pending content found for site %s, publishing", site.Slug)

	sections, err := s.service.GetSections(ctx, site.ID)
	if err != nil {
		s.log.Errorf("Scheduler: cannot get sections for site %s: %v", site.Slug, err)
		return
	}

	layouts, _ := s.service.GetLayouts(ctx, site.ID)
	if layouts == nil {
		layouts = []*Layout{}
	}

	settings, _ := s.service.GetSettings(ctx, site.ID)
	if settings == nil {
		settings = []*Setting{}
	}

	contributors, _ := s.service.GetContributors(ctx, site.ID)
	if contributors == nil {
		contributors = []*Contributor{}
	}

	userAuthors := s.service.BuildUserAuthorsMap(ctx, contents, contributors)

	_, err = s.htmlGen.GenerateHTML(ctx, site, contents, sections, layouts, settings, contributors, userAuthors)
	if err != nil {
		s.log.Errorf("Scheduler: HTML generation failed for site %s: %v", site.Slug, err)
		return
	}

	cfg, err := buildPublishConfigFromSettings(settings)
	if err != nil {
		s.log.Errorf("Scheduler: cannot build publish config for site %s: %v", site.Slug, err)
		return
	}

	result, err := s.publisher.Publish(ctx, cfg, site.Slug)
	if err != nil {
		s.log.Errorf("Scheduler: publish failed for site %s: %v", site.Slug, err)
		return
	}

	if result.NoChanges {
		s.log.Infof("Scheduler: no changes for site %s", site.Slug)
	} else {
		s.log.Infof("Scheduler: published site %s: %s", site.Slug, result.CommitURL)
	}

	now := time.Now()
	site.LastPublishedAt = &now
	_ = s.service.UpdateSite(ctx, site)
}

var errPublishNotConfigured = errors.New("publish not configured")

func buildPublishConfigFromSettings(settings []*Setting) (PublishConfig, error) {
	m := make(map[string]string)
	for _, p := range settings {
		m[p.RefKey] = p.Value
	}

	repoURL := m["ssg.publish.repo.url"]
	if repoURL == "" {
		return PublishConfig{}, errPublishNotConfigured
	}

	branch := m["ssg.publish.branch"]
	if branch == "" {
		branch = "gh-pages"
	}

	commitName := m["ssg.git.commit.user.name"]
	if commitName == "" {
		commitName = "Clio Bot"
	}

	commitEmail := m["ssg.git.commit.user.email"]
	if commitEmail == "" {
		commitEmail = "clio@localhost"
	}

	authToken := m["ssg.publish.auth.token"]
	useSSH := true
	if authToken != "" && strings.HasPrefix(repoURL, "https://") {
		useSSH = false
	}

	return PublishConfig{
		RepoURL:     repoURL,
		Branch:      branch,
		AuthToken:   authToken,
		CommitName:  commitName,
		CommitEmail: commitEmail,
		UseSSH:      useSSH,
	}, nil
}

func hasPendingContent(contents []*Content, since *time.Time) bool {
	now := time.Now()
	for _, c := range contents {
		if c.Draft {
			continue
		}
		if c.PublishedAt == nil {
			continue
		}
		if c.PublishedAt.After(now) {
			continue
		}
		if since == nil || c.PublishedAt.After(*since) {
			return true
		}
	}
	return false
}
