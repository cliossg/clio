package ssg_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cliossg/clio/internal/feat/ssg"
	"github.com/cliossg/clio/internal/feat/ssg/fake"
	"github.com/cliossg/clio/pkg/cl/logger"
	"github.com/google/uuid"
)

func newTestLogger() logger.Logger {
	return logger.New("error")
}

func enabledSettings(siteID uuid.UUID, interval string) map[uuid.UUID][]*ssg.Setting {
	return map[uuid.UUID][]*ssg.Setting{
		siteID: {
			{RefKey: "ssg.scheduled.publish.enabled", Value: "true"},
			{RefKey: "ssg.scheduled.publish.interval", Value: interval},
		},
	}
}

func TestSchedulerStartNoSites(t *testing.T) {
	svc := fake.NewService()
	s := ssg.NewScheduler(svc, nil, nil, newTestLogger())

	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Stop(context.Background()); err != nil {
		t.Fatalf("unexpected stop error: %v", err)
	}
}

func TestSchedulerStartListSitesError(t *testing.T) {
	svc := fake.NewService()
	svc.ListSitesErr = errors.New("db down")
	s := ssg.NewScheduler(svc, nil, nil, newTestLogger())

	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start should not propagate error, got: %v", err)
	}
}

func TestSchedulerStartSchedulingDisabled(t *testing.T) {
	svc := fake.NewService()
	siteID := uuid.New()
	svc.Sites = []*ssg.Site{{ID: siteID, Slug: "test"}}
	svc.Settings = map[uuid.UUID][]*ssg.Setting{
		siteID: {
			{RefKey: "ssg.scheduled.publish.enabled", Value: "false"},
		},
	}

	s := ssg.NewScheduler(svc, nil, nil, newTestLogger())
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Stop(context.Background()); err != nil {
		t.Fatalf("unexpected stop error: %v", err)
	}
}

func TestSchedulerStartNoEnabledSetting(t *testing.T) {
	svc := fake.NewService()
	siteID := uuid.New()
	svc.Sites = []*ssg.Site{{ID: siteID, Slug: "test"}}

	s := ssg.NewScheduler(svc, nil, nil, newTestLogger())
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Stop(context.Background()); err != nil {
		t.Fatalf("unexpected stop error: %v", err)
	}
}

func TestSchedulerStartAndStop(t *testing.T) {
	svc := fake.NewService()
	siteID := uuid.New()
	svc.Sites = []*ssg.Site{{ID: siteID, Slug: "test"}}
	svc.Settings = enabledSettings(siteID, "1h")

	s := ssg.NewScheduler(svc, nil, nil, newTestLogger())
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}
	if err := s.Stop(context.Background()); err != nil {
		t.Fatalf("unexpected stop error: %v", err)
	}
}

func TestSchedulerStartDefaultInterval(t *testing.T) {
	svc := fake.NewService()
	siteID := uuid.New()
	svc.Sites = []*ssg.Site{{ID: siteID, Slug: "test"}}
	svc.Settings = map[uuid.UUID][]*ssg.Setting{
		siteID: {
			{RefKey: "ssg.scheduled.publish.enabled", Value: "true"},
		},
	}

	s := ssg.NewScheduler(svc, nil, nil, newTestLogger())
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Stop(context.Background()); err != nil {
		t.Fatalf("unexpected stop error: %v", err)
	}
}

func TestSchedulerStartInvalidInterval(t *testing.T) {
	svc := fake.NewService()
	siteID := uuid.New()
	svc.Sites = []*ssg.Site{{ID: siteID, Slug: "test"}}
	svc.Settings = enabledSettings(siteID, "not-a-duration")

	s := ssg.NewScheduler(svc, nil, nil, newTestLogger())
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Stop(context.Background()); err != nil {
		t.Fatalf("unexpected stop error: %v", err)
	}
}

func TestSchedulerStartIntervalTooShort(t *testing.T) {
	svc := fake.NewService()
	siteID := uuid.New()
	svc.Sites = []*ssg.Site{{ID: siteID, Slug: "test"}}
	svc.Settings = enabledSettings(siteID, "5s")

	s := ssg.NewScheduler(svc, nil, nil, newTestLogger())
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Stop(context.Background()); err != nil {
		t.Fatalf("unexpected stop error: %v", err)
	}
}

func TestSchedulerStartMultipleSites(t *testing.T) {
	svc := fake.NewService()
	siteA := uuid.New()
	siteB := uuid.New()
	svc.Sites = []*ssg.Site{
		{ID: siteA, Slug: "site-a"},
		{ID: siteB, Slug: "site-b"},
	}
	svc.Settings = map[uuid.UUID][]*ssg.Setting{
		siteA: {{RefKey: "ssg.scheduled.publish.enabled", Value: "false"}},
		siteB: {{RefKey: "ssg.scheduled.publish.enabled", Value: "true"},
			{RefKey: "ssg.scheduled.publish.interval", Value: "30m"}},
	}

	s := ssg.NewScheduler(svc, nil, nil, newTestLogger())
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Stop(context.Background()); err != nil {
		t.Fatalf("unexpected stop error: %v", err)
	}
}

func TestSchedulerStopsOnContextCancel(t *testing.T) {
	svc := fake.NewService()
	siteID := uuid.New()
	svc.Sites = []*ssg.Site{{ID: siteID, Slug: "test"}}
	svc.Settings = enabledSettings(siteID, "1h")

	ctx, cancel := context.WithCancel(context.Background())
	s := ssg.NewScheduler(svc, nil, nil, newTestLogger())
	if err := s.Start(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cancel()
	time.Sleep(50 * time.Millisecond)

	if err := s.Stop(context.Background()); err != nil {
		t.Fatalf("unexpected stop error: %v", err)
	}
}

func TestSchedulerDoubleStop(t *testing.T) {
	svc := fake.NewService()
	siteID := uuid.New()
	svc.Sites = []*ssg.Site{{ID: siteID, Slug: "test"}}
	svc.Settings = enabledSettings(siteID, "1h")

	s := ssg.NewScheduler(svc, nil, nil, newTestLogger())
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Stop(context.Background()); err != nil {
		t.Fatalf("first stop error: %v", err)
	}
	if err := s.Stop(context.Background()); err != nil {
		t.Fatalf("second stop error: %v", err)
	}
}

func TestSchedulerStartOnlyFirstEnabled(t *testing.T) {
	svc := fake.NewService()
	siteA := uuid.New()
	siteB := uuid.New()
	svc.Sites = []*ssg.Site{
		{ID: siteA, Slug: "site-a"},
		{ID: siteB, Slug: "site-b"},
	}
	svc.Settings = map[uuid.UUID][]*ssg.Setting{
		siteA: {{RefKey: "ssg.scheduled.publish.enabled", Value: "true"},
			{RefKey: "ssg.scheduled.publish.interval", Value: "2h"}},
		siteB: {{RefKey: "ssg.scheduled.publish.enabled", Value: "true"},
			{RefKey: "ssg.scheduled.publish.interval", Value: "30m"}},
	}

	s := ssg.NewScheduler(svc, nil, nil, newTestLogger())
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Stop(context.Background()); err != nil {
		t.Fatalf("unexpected stop error: %v", err)
	}
}
