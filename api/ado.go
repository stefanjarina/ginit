package api

import (
	"context"
	"runtime"
	"strings"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/identity"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/location"
	gerrors "github.com/stefanjarina/ginit/errors"
)

type AdoClient struct {
	url            string
	token          string
	coreClient     core.Client
	locationClient location.Client
	repoClient     git.Client
	user           *identity.Identity
	ctx            context.Context
}

func NewAdoClient(token string, orgUrl string) *AdoClient {
	if !strings.HasPrefix(orgUrl, "https://") {
		orgUrl = "https://dev.azure.com/" + orgUrl
	}
	return &AdoClient{url: orgUrl, token: token}
}

func (ac *AdoClient) Connect() error {
	ctx := context.Background()
	conn := azuredevops.NewPatConnection(ac.url, ac.token)
	coreClient, err := core.NewClient(ctx, conn)
	if err != nil {
		return err
	}
	gitClient, err := git.NewClient(ctx, conn)
	if err != nil {
		return err
	}
	locClient := location.NewClient(ctx, conn)
	conData, err := locClient.GetConnectionData(ctx, location.GetConnectionDataArgs{})
	if err != nil {
		return err
	}
	ac.repoClient = gitClient
	ac.locationClient = locClient
	ac.coreClient = coreClient
	ac.user = conData.AuthenticatedUser
	ac.ctx = ctx
	return nil
}

// GetProjects returns the names of every project the authenticated user can see.
func (ac *AdoClient) GetProjects() ([]string, error) {
	resp, err := ac.coreClient.GetProjects(ac.ctx, core.GetProjectsArgs{})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(resp.Value))
	for _, p := range resp.Value {
		if p.Name != nil {
			out = append(out, *p.Name)
		}
	}
	return out, nil
}

// CreateRepository creates a git repo under the given project. Returns SSH URL
// on Unix or RemoteUrl (HTTPS) on Windows. Mirrors AzureService.CreateRepository.
func (ac *AdoClient) CreateRepository(projectName, repoName string) (string, error) {
	projectRef, err := ac.coreClient.GetProject(ac.ctx, core.GetProjectArgs{ProjectId: &projectName})
	if err != nil {
		return "", gerrors.NewProvider("azure", "fetch project '"+projectName+"'", err)
	}

	tpRef := &core.TeamProjectReference{Id: projectRef.Id, Name: projectRef.Name}
	created, err := ac.repoClient.CreateRepository(ac.ctx, git.CreateRepositoryArgs{
		GitRepositoryToCreate: &git.GitRepositoryCreateOptions{
			Name:    &repoName,
			Project: tpRef,
		},
	})
	if err != nil {
		// Friendly error for the well-known duplicate-name case.
		if strings.Contains(err.Error(), "TF400948") {
			return "", gerrors.NewProvider("azure", "repository '"+repoName+"' already exists", err)
		}
		return "", gerrors.NewProvider("azure", "create repository", err)
	}
	if runtime.GOOS == "windows" && created.RemoteUrl != nil {
		return *created.RemoteUrl, nil
	}
	if created.SshUrl != nil {
		return *created.SshUrl, nil
	}
	if created.RemoteUrl != nil {
		return *created.RemoteUrl, nil
	}
	return "", gerrors.NewProvider("azure", "created repo has no URL", nil)
}
