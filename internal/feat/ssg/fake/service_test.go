package fake

import (
	"context"
	"testing"
	"time"

	"github.com/cliossg/clio/internal/feat/ssg"
	"github.com/google/uuid"
)

func TestServiceImplementsInterface(t *testing.T) {
	var _ ssg.Service = (*Service)(nil)
}

func TestServiceListSites(t *testing.T) {
	s := NewService()
	site := &ssg.Site{ID: uuid.New(), Name: "test"}
	s.Sites = []*ssg.Site{site}

	sites, err := s.ListSites(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sites) != 1 || sites[0].Name != "test" {
		t.Errorf("got %v, want [test]", sites)
	}
}

func TestServiceListSitesError(t *testing.T) {
	s := NewService()
	s.ListSitesErr = errTest

	_, err := s.ListSites(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceGetSettingByRefKey(t *testing.T) {
	s := NewService()
	siteID := uuid.New()
	s.Settings[siteID] = []*ssg.Setting{
		{RefKey: "a", Value: "1"},
		{RefKey: "b", Value: "2"},
	}

	got, err := s.GetSettingByRefKey(context.Background(), siteID, "b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.Value != "2" {
		t.Errorf("got %v, want value=2", got)
	}

	got, err = s.GetSettingByRefKey(context.Background(), siteID, "missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestServiceGetSettingByRefKeyFunc(t *testing.T) {
	s := NewService()
	s.GetSettingByRefKeyFunc = func(_ uuid.UUID, refKey string) (*ssg.Setting, error) {
		if refKey == "custom" {
			return &ssg.Setting{Value: "custom-val"}, nil
		}
		return nil, nil
	}

	got, _ := s.GetSettingByRefKey(context.Background(), uuid.New(), "custom")
	if got == nil || got.Value != "custom-val" {
		t.Errorf("got %v, want custom-val", got)
	}
}

func TestServiceUpdateSiteRecordsCalls(t *testing.T) {
	s := NewService()
	site := &ssg.Site{ID: uuid.New(), Name: "updated"}

	err := s.UpdateSite(context.Background(), site)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.UpdateSiteCalls) != 1 || s.UpdateSiteCalls[0].Name != "updated" {
		t.Errorf("UpdateSiteCalls = %v, want [updated]", s.UpdateSiteCalls)
	}
}

func TestServiceGetAllContentWithMeta(t *testing.T) {
	s := NewService()
	siteID := uuid.New()
	now := time.Now()
	s.Contents[siteID] = []*ssg.Content{
		{Heading: "post1", PublishedAt: &now},
	}

	contents, err := s.GetAllContentWithMeta(context.Background(), siteID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(contents) != 1 || contents[0].Heading != "post1" {
		t.Errorf("got %v, want [post1]", contents)
	}
}

func TestServiceBuildUserAuthorsMap(t *testing.T) {
	s := NewService()
	m := s.BuildUserAuthorsMap(context.Background(), nil, nil)
	if m == nil {
		t.Fatal("expected non-nil map")
	}
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

var errTest = &testError{msg: "test error"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
