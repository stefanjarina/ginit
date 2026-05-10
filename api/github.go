package api

import (
	"context"

	"github.com/google/go-github/v80/github"
)

type GithubClient struct {
	token  string
	client *github.Client
	user   *github.User
}

func NewGithubClient(token string) *GithubClient {
	gc := new(GithubClient)
	gc.token = token
	return gc
}

func (gc *GithubClient) Connect() error {
	ctx := context.Background()

	gc.client = github.NewClient(nil).WithAuthToken(gc.token)

	currentUser, _, err := gc.client.Users.Get(ctx, "")
	if err != nil {
		return err
	}

	gc.user = currentUser
	return nil
}
