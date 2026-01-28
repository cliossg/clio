package git

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	"github.com/cliossg/clio/pkg/cl/logger"
)

type client struct {
	log logger.Logger
}

func NewClient(log logger.Logger) Client {
	return &client{
		log: log,
	}
}

func (c *client) Clone(ctx context.Context, repoURL, localPath string, auth Auth, env []string) error {
	if auth.Method == AuthToken {
		u, err := url.Parse(repoURL)
		if err != nil {
			return fmt.Errorf("cannot parse repo URL: %w", err)
		}
		u.User = url.UserPassword("oauth2", auth.Token)
		repoURL = u.String()
	}

	cmd := exec.CommandContext(ctx, "git", "clone", repoURL, localPath)
	cmd.Env = env
	return c.runCommand(cmd)
}

func (c *client) Checkout(ctx context.Context, localRepoPath, branch string, create bool, env []string) error {
	args := []string{"checkout"}
	if create {
		args = append(args, "-b")
	}
	args = append(args, branch)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = localRepoPath
	cmd.Env = env
	return c.runCommand(cmd)
}

func (c *client) Add(ctx context.Context, localRepoPath, pathspec string, env []string) error {
	cmd := exec.CommandContext(ctx, "git", "add", pathspec)
	cmd.Dir = localRepoPath
	cmd.Env = env
	return c.runCommand(cmd)
}

func (c *client) Commit(ctx context.Context, localRepoPath string, commit Commit, env []string) (string, error) {
	configUserCmd := exec.CommandContext(ctx, "git", "config", "user.name", commit.UserName)
	configUserCmd.Dir = localRepoPath
	configUserCmd.Env = env
	if err := c.runCommand(configUserCmd); err != nil {
		return "", fmt.Errorf("cannot set git user name: %w", err)
	}

	configEmailCmd := exec.CommandContext(ctx, "git", "config", "user.email", commit.UserEmail)
	configEmailCmd.Dir = localRepoPath
	configEmailCmd.Env = env
	if err := c.runCommand(configEmailCmd); err != nil {
		return "", fmt.Errorf("cannot set git user email: %w", err)
	}

	status, err := c.Status(ctx, localRepoPath, env)
	if err != nil {
		return "", fmt.Errorf("cannot check git status before commit: %w", err)
	}
	if status == "" {
		c.log.Info("No changes to commit")
		return "", nil
	}

	commitCmd := exec.CommandContext(ctx, "git", "commit", "-m", commit.Message)
	commitCmd.Dir = localRepoPath
	commitCmd.Env = env
	if err := c.runCommand(commitCmd); err != nil {
		return "", fmt.Errorf("cannot commit changes: %w", err)
	}

	hashCmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	hashCmd.Dir = localRepoPath
	hashCmd.Env = env
	var out bytes.Buffer
	hashCmd.Stdout = &out
	if err := c.runCommand(hashCmd); err != nil {
		return "", fmt.Errorf("cannot get commit hash: %w", err)
	}

	return strings.TrimSpace(out.String()), nil
}

func (c *client) Push(ctx context.Context, localRepoPath string, auth Auth, remote, branch string, env []string) error {
	pushRepoURL := remote

	if auth.Method == AuthToken {
		getURLCmd := exec.CommandContext(ctx, "git", "remote", "get-url", remote)
		getURLCmd.Dir = localRepoPath
		getURLCmd.Env = env
		var stdout bytes.Buffer
		getURLCmd.Stdout = &stdout
		if err := c.runCommand(getURLCmd); err != nil {
			return fmt.Errorf("cannot get remote URL for %s: %w", remote, err)
		}
		baseRepoURL := strings.TrimSpace(stdout.String())

		u, err := url.Parse(baseRepoURL)
		if err != nil {
			return fmt.Errorf("cannot parse base repo URL: %w", err)
		}
		u.User = url.UserPassword("oauth2", auth.Token)
		pushRepoURL = u.String()
	}

	cmd := exec.CommandContext(ctx, "git", "push", "--force", pushRepoURL, branch)
	cmd.Dir = localRepoPath
	cmd.Env = env

	return c.runCommand(cmd)
}

func (c *client) Status(ctx context.Context, localRepoPath string, env []string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = localRepoPath
	cmd.Env = env
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := c.runCommand(cmd); err != nil {
		return "", fmt.Errorf("cannot get git status: %w", err)
	}
	return stdout.String(), nil
}

func (c *client) Log(ctx context.Context, localRepoPath string, args []string, env []string) (string, error) {
	cmdArgs := []string{"log"}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.CommandContext(ctx, "git", cmdArgs...)
	cmd.Dir = localRepoPath
	cmd.Env = env
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := c.runCommand(cmd); err != nil {
		return "", fmt.Errorf("cannot get git log: %w", err)
	}

	return stdout.String(), nil
}

func (c *client) runCommand(cmd *exec.Cmd) error {
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("command %q failed: %w: %s", cmd.String(), err, stderr.String())
	}

	return nil
}
