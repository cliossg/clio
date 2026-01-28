package git

import (
	"context"
)

type Client interface {
	Clone(ctx context.Context, repoURL, localPath string, auth Auth, env []string) error
	Checkout(ctx context.Context, localRepoPath, branch string, create bool, env []string) error
	Add(ctx context.Context, localRepoPath, pathspec string, env []string) error
	Commit(ctx context.Context, localRepoPath string, commit Commit, env []string) (string, error)
	Push(ctx context.Context, localRepoPath string, auth Auth, remote, branch string, env []string) error
	Status(ctx context.Context, localRepoPath string, env []string) (string, error)
	Log(ctx context.Context, localRepoPath string, args []string, env []string) (string, error)
}

type Auth struct {
	Method AuthMethod
	Token  string
}

type AuthMethod string

const (
	AuthToken AuthMethod = "token"
	AuthSSH   AuthMethod = "ssh"
)

type Commit struct {
	UserName  string
	UserEmail string
	Message   string
}
