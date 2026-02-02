package ssg

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cliossg/clio/pkg/cl/git"
)

type PublishConfig struct {
	RepoURL     string
	Branch      string
	AuthToken   string
	CommitName  string
	CommitEmail string
	UseSSH      bool
}

type PublishResult struct {
	CommitHash string
	CommitURL  string
	Added      int
	Modified   int
	Deleted    int
	NoChanges  bool
}

type PlanResult struct {
	Added    []string
	Modified []string
	Deleted  []string
	Summary  string
}

type Publisher struct {
	workspace *Workspace
	gitClient git.Client
}

func NewPublisher(workspace *Workspace, gitClient git.Client) *Publisher {
	return &Publisher{
		workspace: workspace,
		gitClient: gitClient,
	}
}

func (p *Publisher) Validate(cfg PublishConfig) error {
	if cfg.RepoURL == "" {
		return fmt.Errorf("repository URL is required")
	}
	if cfg.Branch == "" {
		return fmt.Errorf("branch name is required")
	}
	if !cfg.UseSSH && cfg.AuthToken == "" {
		return fmt.Errorf("auth token is required when not using SSH")
	}
	if cfg.CommitEmail == "" {
		return fmt.Errorf("commit email is required")
	}
	return nil
}

func (p *Publisher) Publish(ctx context.Context, cfg PublishConfig, siteSlug string) (*PublishResult, error) {
	if err := p.Validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	sourceDir := p.workspace.GetHTMLPath(siteSlug)
	if _, err := os.Stat(sourceDir); err != nil {
		return nil, fmt.Errorf("source directory not found: %w", err)
	}

	parentTempDir, err := os.MkdirTemp("", "clio-publish-*")
	if err != nil {
		return nil, fmt.Errorf("cannot create temp dir: %w", err)
	}
	defer os.RemoveAll(parentTempDir)

	tempDir := filepath.Join(parentTempDir, "repo")
	env := os.Environ()

	auth := git.Auth{
		Method: git.AuthToken,
		Token:  cfg.AuthToken,
	}
	if cfg.UseSSH {
		auth = git.Auth{Method: git.AuthSSH}
	}

	if err := p.gitClient.Clone(ctx, cfg.RepoURL, tempDir, auth, env); err != nil {
		return nil, fmt.Errorf("cannot clone repo: %w", err)
	}

	if err := p.gitClient.Checkout(ctx, tempDir, cfg.Branch, false, env); err != nil {
		if err := p.gitClient.Checkout(ctx, tempDir, cfg.Branch, true, env); err != nil {
			return nil, fmt.Errorf("cannot checkout branch %s: %w", cfg.Branch, err)
		}
	}

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read temp dir: %w", err)
	}
	for _, entry := range entries {
		if entry.Name() == ".git" {
			continue
		}
		entryPath := filepath.Join(tempDir, entry.Name())
		if err := os.RemoveAll(entryPath); err != nil {
			return nil, fmt.Errorf("cannot clean %s: %w", entry.Name(), err)
		}
	}

	if err := copyDirRecursive(sourceDir, tempDir); err != nil {
		return nil, fmt.Errorf("cannot copy source: %w", err)
	}

	if err := p.gitClient.Add(ctx, tempDir, ".", env); err != nil {
		return nil, fmt.Errorf("cannot stage files: %w", err)
	}

	commitName := cfg.CommitName
	if commitName == "" {
		commitName = "Clio Publisher"
	}

	commit := git.Commit{
		UserName:  commitName,
		UserEmail: cfg.CommitEmail,
		Message:   fmt.Sprintf("Deploy site - %s", time.Now().Format("2006-01-02 15:04:05")),
	}

	commitHash, err := p.gitClient.Commit(ctx, tempDir, commit, env)
	if err != nil {
		return nil, fmt.Errorf("cannot commit: %w", err)
	}

	if commitHash == "" {
		return &PublishResult{NoChanges: true}, nil
	}

	if err := p.gitClient.Push(ctx, tempDir, auth, "origin", cfg.Branch, env); err != nil {
		return nil, fmt.Errorf("cannot push: %w", err)
	}

	repoURL := strings.TrimSuffix(cfg.RepoURL, ".git")
	commitURL := fmt.Sprintf("%s/commit/%s", repoURL, commitHash)

	return &PublishResult{
		CommitHash: commitHash,
		CommitURL:  commitURL,
	}, nil
}

func (p *Publisher) Backup(ctx context.Context, cfg PublishConfig, siteSlug string) (*PublishResult, error) {
	if err := p.Validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	markdownDir := p.workspace.GetMarkdownPath(siteSlug)
	if _, err := os.Stat(markdownDir); err != nil {
		return nil, fmt.Errorf("markdown directory not found: %w", err)
	}

	imagesDir := p.workspace.GetImagesPath(siteSlug)
	metaDir := p.workspace.GetMetaPath(siteSlug)
	profilesDir := p.workspace.GetProfilesPath()

	parentTempDir, err := os.MkdirTemp("", "clio-backup-*")
	if err != nil {
		return nil, fmt.Errorf("cannot create temp dir: %w", err)
	}
	defer os.RemoveAll(parentTempDir)

	tempDir := filepath.Join(parentTempDir, "repo")
	env := os.Environ()

	auth := git.Auth{
		Method: git.AuthToken,
		Token:  cfg.AuthToken,
	}
	if cfg.UseSSH {
		auth = git.Auth{Method: git.AuthSSH}
	}

	if err := p.gitClient.Clone(ctx, cfg.RepoURL, tempDir, auth, env); err != nil {
		return nil, fmt.Errorf("cannot clone repo: %w", err)
	}

	if err := p.gitClient.Checkout(ctx, tempDir, cfg.Branch, false, env); err != nil {
		if err := p.gitClient.Checkout(ctx, tempDir, cfg.Branch, true, env); err != nil {
			return nil, fmt.Errorf("cannot checkout branch %s: %w", cfg.Branch, err)
		}
	}

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read temp dir: %w", err)
	}
	for _, entry := range entries {
		if entry.Name() == ".git" {
			continue
		}
		entryPath := filepath.Join(tempDir, entry.Name())
		if err := os.RemoveAll(entryPath); err != nil {
			return nil, fmt.Errorf("cannot clean %s: %w", entry.Name(), err)
		}
	}

	contentDst := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDst, 0755); err != nil {
		return nil, fmt.Errorf("cannot create content dir: %w", err)
	}
	if err := copyDirRecursive(markdownDir, contentDst); err != nil {
		return nil, fmt.Errorf("cannot copy markdown: %w", err)
	}

	if _, err := os.Stat(imagesDir); err == nil {
		imagesDst := filepath.Join(tempDir, "images")
		if err := os.MkdirAll(imagesDst, 0755); err != nil {
			return nil, fmt.Errorf("cannot create images dir: %w", err)
		}
		if err := copyDirRecursive(imagesDir, imagesDst); err != nil {
			return nil, fmt.Errorf("cannot copy images: %w", err)
		}
	}

	if _, err := os.Stat(profilesDir); err == nil {
		profilesDst := filepath.Join(tempDir, "profiles")
		if err := os.MkdirAll(profilesDst, 0755); err != nil {
			return nil, fmt.Errorf("cannot create profiles dir: %w", err)
		}
		if err := copyDirRecursive(profilesDir, profilesDst); err != nil {
			return nil, fmt.Errorf("cannot copy profiles: %w", err)
		}
	}

	if _, err := os.Stat(metaDir); err == nil {
		metaDst := filepath.Join(tempDir, "meta")
		if err := os.MkdirAll(metaDst, 0755); err != nil {
			return nil, fmt.Errorf("cannot create meta dir: %w", err)
		}
		if err := copyDirRecursive(metaDir, metaDst); err != nil {
			return nil, fmt.Errorf("cannot copy meta: %w", err)
		}
	}

	if err := p.gitClient.Add(ctx, tempDir, ".", env); err != nil {
		return nil, fmt.Errorf("cannot stage files: %w", err)
	}

	commitName := cfg.CommitName
	if commitName == "" {
		commitName = "Clio Publisher"
	}

	commit := git.Commit{
		UserName:  commitName,
		UserEmail: cfg.CommitEmail,
		Message:   fmt.Sprintf("Backup site - %s", time.Now().Format("2006-01-02 15:04:05")),
	}

	commitHash, err := p.gitClient.Commit(ctx, tempDir, commit, env)
	if err != nil {
		return nil, fmt.Errorf("cannot commit: %w", err)
	}

	if commitHash == "" {
		return &PublishResult{NoChanges: true}, nil
	}

	if err := p.gitClient.Push(ctx, tempDir, auth, "origin", cfg.Branch, env); err != nil {
		return nil, fmt.Errorf("cannot push: %w", err)
	}

	repoURL := strings.TrimSuffix(cfg.RepoURL, ".git")
	commitURL := fmt.Sprintf("%s/commit/%s", repoURL, commitHash)

	return &PublishResult{
		CommitHash: commitHash,
		CommitURL:  commitURL,
	}, nil
}

func (p *Publisher) Plan(ctx context.Context, cfg PublishConfig, siteSlug string) (*PlanResult, error) {
	if err := p.Validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	sourceDir := p.workspace.GetHTMLPath(siteSlug)
	if _, err := os.Stat(sourceDir); err != nil {
		return nil, fmt.Errorf("source directory not found: %w", err)
	}

	parentTempDir, err := os.MkdirTemp("", "clio-plan-*")
	if err != nil {
		return nil, fmt.Errorf("cannot create temp dir: %w", err)
	}
	defer os.RemoveAll(parentTempDir)

	tempDir := filepath.Join(parentTempDir, "repo")
	env := os.Environ()

	auth := git.Auth{
		Method: git.AuthToken,
		Token:  cfg.AuthToken,
	}
	if cfg.UseSSH {
		auth = git.Auth{Method: git.AuthSSH}
	}

	if err := p.gitClient.Clone(ctx, cfg.RepoURL, tempDir, auth, env); err != nil {
		return nil, fmt.Errorf("cannot clone repo: %w", err)
	}

	_ = p.gitClient.Checkout(ctx, tempDir, cfg.Branch, false, env)

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read temp dir: %w", err)
	}
	for _, entry := range entries {
		if entry.Name() == ".git" {
			continue
		}
		entryPath := filepath.Join(tempDir, entry.Name())
		if err := os.RemoveAll(entryPath); err != nil {
			return nil, fmt.Errorf("cannot clean %s: %w", entry.Name(), err)
		}
	}

	if err := copyDirRecursive(sourceDir, tempDir); err != nil {
		return nil, fmt.Errorf("cannot copy source: %w", err)
	}

	if err := p.gitClient.Add(ctx, tempDir, ".", env); err != nil {
		return nil, fmt.Errorf("cannot stage files: %w", err)
	}

	statusOutput, err := p.gitClient.Status(ctx, tempDir, env)
	if err != nil {
		return nil, fmt.Errorf("cannot get status: %w", err)
	}

	result := &PlanResult{}
	lines := strings.Split(statusOutput, "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}
		status := line[0:2]
		filename := strings.TrimSpace(line[3:])

		switch status {
		case "A ", "??":
			result.Added = append(result.Added, filename)
		case "M ":
			result.Modified = append(result.Modified, filename)
		case "D ":
			result.Deleted = append(result.Deleted, filename)
		}
	}

	result.Summary = fmt.Sprintf("Added: %d, Modified: %d, Deleted: %d",
		len(result.Added), len(result.Modified), len(result.Deleted))

	return result, nil
}

func copyDirRecursive(src, dst string) error {
	return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(dstPath, data, info.Mode())
	})
}
