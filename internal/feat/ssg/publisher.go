package ssg

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// PublishConfig holds configuration for publishing.
type PublishConfig struct {
	RepoURL     string // Git repository URL
	Branch      string // Target branch (e.g., "gh-pages")
	AuthToken   string // Auth token (GitHub PAT)
	CommitName  string // Commit author name
	CommitEmail string // Commit author email
}

// PublishResult contains the result of a publish operation.
type PublishResult struct {
	CommitHash string
	CommitURL  string
	Added      int
	Modified   int
	Deleted    int
}

// PlanResult contains the result of a dry-run plan.
type PlanResult struct {
	Added    []string
	Modified []string
	Deleted  []string
	Summary  string
}

// Publisher handles publishing generated HTML to a Git repository.
type Publisher struct {
	workspace *Workspace
}

// NewPublisher creates a new publisher.
func NewPublisher(workspace *Workspace) *Publisher {
	return &Publisher{
		workspace: workspace,
	}
}

// Validate checks if the configuration is valid.
func (p *Publisher) Validate(cfg PublishConfig) error {
	if cfg.RepoURL == "" {
		return fmt.Errorf("repository URL is required")
	}
	if cfg.Branch == "" {
		return fmt.Errorf("branch name is required")
	}
	if cfg.AuthToken == "" {
		return fmt.Errorf("auth token is required")
	}
	if cfg.CommitEmail == "" {
		return fmt.Errorf("commit email is required")
	}
	return nil
}

// Publish publishes the generated HTML to the configured repository.
func (p *Publisher) Publish(ctx context.Context, cfg PublishConfig, siteSlug string) (*PublishResult, error) {
	if err := p.Validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Get the HTML source directory
	sourceDir := p.workspace.GetHTMLPath(siteSlug)
	if _, err := os.Stat(sourceDir); err != nil {
		return nil, fmt.Errorf("source directory not found: %w", err)
	}

	// Create temp directory for clone
	tempDir, err := os.MkdirTemp("", "clio-publish-*")
	if err != nil {
		return nil, fmt.Errorf("cannot create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone repository
	auth := &http.BasicAuth{
		Username: "git", // Can be anything for token auth
		Password: cfg.AuthToken,
	}

	repo, err := git.PlainClone(tempDir, false, &git.CloneOptions{
		URL:      cfg.RepoURL,
		Auth:     auth,
		Progress: nil,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot clone repository: %w", err)
	}

	// Checkout or create branch
	w, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("cannot get worktree: %w", err)
	}

	branchRef := plumbing.NewBranchReferenceName(cfg.Branch)
	err = w.Checkout(&git.CheckoutOptions{
		Branch: branchRef,
		Create: false,
	})
	if err != nil {
		// Try to create the branch
		err = w.Checkout(&git.CheckoutOptions{
			Branch: branchRef,
			Create: true,
		})
		if err != nil {
			return nil, fmt.Errorf("cannot checkout branch %s: %w", cfg.Branch, err)
		}
	}

	// Clean directory (except .git)
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

	// Copy source to temp directory
	if err := copyDirRecursive(sourceDir, tempDir); err != nil {
		return nil, fmt.Errorf("cannot copy source: %w", err)
	}

	// Add all files
	if err := w.AddGlob("."); err != nil {
		return nil, fmt.Errorf("cannot stage files: %w", err)
	}

	// Commit
	commitMsg := fmt.Sprintf("Deploy site - %s", time.Now().Format("2006-01-02 15:04:05"))
	commitName := cfg.CommitName
	if commitName == "" {
		commitName = "Clio Publisher"
	}

	commit, err := w.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  commitName,
			Email: cfg.CommitEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot commit: %w", err)
	}

	// Push
	err = repo.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", cfg.Branch, cfg.Branch)),
		},
		Auth: auth,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return nil, fmt.Errorf("cannot push: %w", err)
	}

	// Build commit URL
	repoURL := strings.TrimSuffix(cfg.RepoURL, ".git")
	commitURL := fmt.Sprintf("%s/commit/%s", repoURL, commit.String())

	return &PublishResult{
		CommitHash: commit.String(),
		CommitURL:  commitURL,
	}, nil
}

// Backup backs up the generated markdown to the configured repository for versioning.
func (p *Publisher) Backup(ctx context.Context, cfg PublishConfig, siteSlug string) (*PublishResult, error) {
	if err := p.Validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	markdownDir := p.workspace.GetMarkdownPath(siteSlug)
	if _, err := os.Stat(markdownDir); err != nil {
		return nil, fmt.Errorf("markdown directory not found: %w", err)
	}

	imagesDir := p.workspace.GetImagesPath(siteSlug)

	tempDir, err := os.MkdirTemp("", "clio-backup-*")
	if err != nil {
		return nil, fmt.Errorf("cannot create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	auth := &http.BasicAuth{
		Username: "git",
		Password: cfg.AuthToken,
	}

	repo, err := git.PlainClone(tempDir, false, &git.CloneOptions{
		URL:      cfg.RepoURL,
		Auth:     auth,
		Progress: nil,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot clone repository: %w", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("cannot get worktree: %w", err)
	}

	branchRef := plumbing.NewBranchReferenceName(cfg.Branch)
	err = w.Checkout(&git.CheckoutOptions{
		Branch: branchRef,
		Create: false,
	})
	if err != nil {
		err = w.Checkout(&git.CheckoutOptions{
			Branch: branchRef,
			Create: true,
		})
		if err != nil {
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

	if err := w.AddGlob("."); err != nil {
		return nil, fmt.Errorf("cannot stage files: %w", err)
	}

	commitMsg := fmt.Sprintf("Backup site - %s", time.Now().Format("2006-01-02 15:04:05"))
	commitName := cfg.CommitName
	if commitName == "" {
		commitName = "Clio Publisher"
	}

	commit, err := w.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  commitName,
			Email: cfg.CommitEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot commit: %w", err)
	}

	err = repo.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", cfg.Branch, cfg.Branch)),
		},
		Auth: auth,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return nil, fmt.Errorf("cannot push: %w", err)
	}

	repoURL := strings.TrimSuffix(cfg.RepoURL, ".git")
	commitURL := fmt.Sprintf("%s/commit/%s", repoURL, commit.String())

	return &PublishResult{
		CommitHash: commit.String(),
		CommitURL:  commitURL,
	}, nil
}

// Plan performs a dry-run showing what would change.
func (p *Publisher) Plan(ctx context.Context, cfg PublishConfig, siteSlug string) (*PlanResult, error) {
	if err := p.Validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Get the HTML source directory
	sourceDir := p.workspace.GetHTMLPath(siteSlug)
	if _, err := os.Stat(sourceDir); err != nil {
		return nil, fmt.Errorf("source directory not found: %w", err)
	}

	// Create temp directory for clone
	tempDir, err := os.MkdirTemp("", "clio-plan-*")
	if err != nil {
		return nil, fmt.Errorf("cannot create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone repository
	auth := &http.BasicAuth{
		Username: "git",
		Password: cfg.AuthToken,
	}

	repo, err := git.PlainClone(tempDir, false, &git.CloneOptions{
		URL:      cfg.RepoURL,
		Auth:     auth,
		Progress: nil,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot clone repository: %w", err)
	}

	// Checkout branch
	w, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("cannot get worktree: %w", err)
	}

	branchRef := plumbing.NewBranchReferenceName(cfg.Branch)
	_ = w.Checkout(&git.CheckoutOptions{
		Branch: branchRef,
		Create: false,
	})

	// Clean directory (except .git)
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

	// Copy source to temp directory
	if err := copyDirRecursive(sourceDir, tempDir); err != nil {
		return nil, fmt.Errorf("cannot copy source: %w", err)
	}

	// Add all files
	if err := w.AddGlob("."); err != nil {
		return nil, fmt.Errorf("cannot stage files: %w", err)
	}

	// Get status
	status, err := w.Status()
	if err != nil {
		return nil, fmt.Errorf("cannot get status: %w", err)
	}

	result := &PlanResult{}
	for file, s := range status {
		switch s.Staging {
		case git.Added, git.Untracked:
			result.Added = append(result.Added, file)
		case git.Modified:
			result.Modified = append(result.Modified, file)
		case git.Deleted:
			result.Deleted = append(result.Deleted, file)
		}
	}

	result.Summary = fmt.Sprintf("Added: %d, Modified: %d, Deleted: %d",
		len(result.Added), len(result.Modified), len(result.Deleted))

	return result, nil
}

// copyDirRecursive copies the contents of src to dst recursively.
func copyDirRecursive(src, dst string) error {
	return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path
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
