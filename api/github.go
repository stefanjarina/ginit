package api

import (
	"context"
	"runtime"

	"github.com/google/go-github/v80/github"
	gerrors "github.com/stefanjarina/ginit/errors"
)

type GithubClient struct {
	token  string
	client *github.Client
	user   *github.User
}

func NewGithubClient(token string) *GithubClient {
	return &GithubClient{token: token}
}

func (gc *GithubClient) Connect() error {
	ctx := context.Background()
	gc.client = github.NewClient(nil).WithAuthToken(gc.token)
	user, _, err := gc.client.Users.Get(ctx, "")
	if err != nil {
		return err
	}
	gc.user = user
	return nil
}

// CreateRepository creates the repo on github.com under the authenticated user
// and returns the SSH URL on Unix (HTTPS clone URL on Windows).
func (gc *GithubClient) CreateRepository(name, description, visibility string) (string, error) {
	private := visibility == "private"
	repo := &github.Repository{
		Name:        github.Ptr(name),
		Description: github.Ptr(description),
		Private:     github.Ptr(private),
	}
	created, _, err := gc.client.Repositories.Create(context.Background(), "", repo)
	if err != nil {
		return "", gerrors.NewProvider("github", "create repository", err)
	}
	if runtime.GOOS == "windows" {
		return created.GetCloneURL(), nil
	}
	return created.GetSSHURL(), nil
}
