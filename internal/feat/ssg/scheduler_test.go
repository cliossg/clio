package ssg

import (
	"testing"
	"time"
)

func TestIsPublishable(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	justNow := now.Add(-time.Millisecond)

	tests := []struct {
		name string
		c    *Content
		want bool
	}{
		{"draft", &Content{Draft: true}, false},
		{"draft with past date", &Content{Draft: true, PublishedAt: &past}, false},
		{"draft with future date", &Content{Draft: true, PublishedAt: &future}, false},
		{"draft with nil date", &Content{Draft: true, PublishedAt: nil}, false},
		{"not draft nil date", &Content{Draft: false, PublishedAt: nil}, true},
		{"not draft past date", &Content{Draft: false, PublishedAt: &past}, true},
		{"not draft future date", &Content{Draft: false, PublishedAt: &future}, false},
		{"not draft just now", &Content{Draft: false, PublishedAt: &justNow}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPublishable(tt.c); got != tt.want {
				t.Errorf("isPublishable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasPendingContent(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	farPast := now.Add(-24 * time.Hour)
	future := now.Add(time.Hour)
	recentPast := now.Add(-5 * time.Minute)

	tests := []struct {
		name     string
		contents []*Content
		since    *time.Time
		want     bool
	}{
		{"empty contents", nil, nil, false},
		{"only drafts", []*Content{{Draft: true, PublishedAt: &past}}, nil, false},
		{"future published date", []*Content{{Draft: false, PublishedAt: &future}}, nil, false},
		{"nil published date", []*Content{{Draft: false, PublishedAt: nil}}, nil, false},
		{"past date no since", []*Content{{Draft: false, PublishedAt: &past}}, nil, true},
		{"past date after since", []*Content{{Draft: false, PublishedAt: &past}}, &farPast, true},
		{"past date before since", []*Content{{Draft: false, PublishedAt: &farPast}}, &past, false},
		{"past date equal to since", []*Content{{Draft: false, PublishedAt: &past}}, &past, false},
		{
			name: "multiple drafts no pending",
			contents: []*Content{
				{Draft: true, PublishedAt: &past},
				{Draft: true, PublishedAt: &farPast},
				{Draft: true, PublishedAt: nil},
			},
			since: nil,
			want:  false,
		},
		{
			name: "mixed with one pending",
			contents: []*Content{
				{Draft: true, PublishedAt: &past},
				{Draft: false, PublishedAt: &future},
				{Draft: false, PublishedAt: nil},
				{Draft: false, PublishedAt: &past},
			},
			since: &farPast,
			want:  true,
		},
		{
			name: "all already published before since",
			contents: []*Content{
				{Draft: false, PublishedAt: &farPast},
				{Draft: false, PublishedAt: &farPast},
			},
			since: &past,
			want:  false,
		},
		{
			name: "recent post after last publish",
			contents: []*Content{
				{Draft: false, PublishedAt: &farPast},
				{Draft: false, PublishedAt: &recentPast},
			},
			since: &past,
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasPendingContent(tt.contents, tt.since); got != tt.want {
				t.Errorf("hasPendingContent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildPublishConfigFromSettings(t *testing.T) {
	tests := []struct {
		name     string
		settings []*Setting
		wantErr  bool
		wantCfg  PublishConfig
	}{
		{"nil settings", nil, true, PublishConfig{}},
		{"empty settings", []*Setting{}, true, PublishConfig{}},
		{"missing repo url", []*Setting{{RefKey: "ssg.publish.branch", Value: "main"}}, true, PublishConfig{}},
		{"empty repo url", []*Setting{{RefKey: "ssg.publish.repo.url", Value: ""}}, true, PublishConfig{}},
		{
			name:     "ssh defaults",
			settings: []*Setting{{RefKey: "ssg.publish.repo.url", Value: "git@github.com:u/r.git"}},
			wantCfg: PublishConfig{
				RepoURL: "git@github.com:u/r.git", Branch: "gh-pages",
				CommitName: "Clio Bot", CommitEmail: "clio@localhost", UseSSH: true,
			},
		},
		{
			name: "https with token",
			settings: []*Setting{
				{RefKey: "ssg.publish.repo.url", Value: "https://github.com/u/r.git"},
				{RefKey: "ssg.publish.auth.token", Value: "tok"},
				{RefKey: "ssg.publish.branch", Value: "main"},
				{RefKey: "ssg.git.commit.user.name", Value: "Bot"},
				{RefKey: "ssg.git.commit.user.email", Value: "b@b.com"},
			},
			wantCfg: PublishConfig{
				RepoURL: "https://github.com/u/r.git", Branch: "main",
				AuthToken: "tok", CommitName: "Bot", CommitEmail: "b@b.com", UseSSH: false,
			},
		},
		{
			name: "ssh with token stays ssh",
			settings: []*Setting{
				{RefKey: "ssg.publish.repo.url", Value: "git@github.com:u/r.git"},
				{RefKey: "ssg.publish.auth.token", Value: "tok"},
			},
			wantCfg: PublishConfig{
				RepoURL: "git@github.com:u/r.git", Branch: "gh-pages",
				AuthToken: "tok", CommitName: "Clio Bot", CommitEmail: "clio@localhost", UseSSH: true,
			},
		},
		{
			name: "https without token uses ssh",
			settings: []*Setting{
				{RefKey: "ssg.publish.repo.url", Value: "https://github.com/u/r.git"},
			},
			wantCfg: PublishConfig{
				RepoURL: "https://github.com/u/r.git", Branch: "gh-pages",
				CommitName: "Clio Bot", CommitEmail: "clio@localhost", UseSSH: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := buildPublishConfigFromSettings(tt.settings)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg != tt.wantCfg {
				t.Errorf("got %+v, want %+v", cfg, tt.wantCfg)
			}
		})
	}
}

func TestNewScheduler(t *testing.T) {
	s := NewScheduler(nil, nil, nil, nil)
	if s == nil {
		t.Fatal("NewScheduler returned nil")
	}
	if s.running {
		t.Error("should not be running after creation")
	}
	if s.stop != nil {
		t.Error("stop channel should be nil after creation")
	}
}

func TestSchedulerStopWhenNotRunning(t *testing.T) {
	s := NewScheduler(nil, nil, nil, nil)
	if err := s.Stop(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
