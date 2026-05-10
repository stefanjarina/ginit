package api

import (
	"context"
	"strings"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/identity"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/location"
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
	ac := new(AdoClient)
	if !strings.HasPrefix(orgUrl, "https://") {
		orgUrl = "https://dev.azure.com/" + orgUrl
	}
	ac.url = orgUrl
	ac.token = token
	return ac
}

func (ac *AdoClient) Connect() error {
	ctx := context.Background()
	connection := azuredevops.NewPatConnection(ac.url, ac.token)
	coreClient, err := core.NewClient(ctx, connection)
	if err != nil {
		return err
	}
	gitClient, err := git.NewClient(ctx, connection)
	if err != nil {
		return err
	}
	locationClient := location.NewClient(ctx, connection)
	conData, err := locationClient.GetConnectionData(ctx, location.GetConnectionDataArgs{})
	if err != nil {
		return err
	}
	ac.repoClient = gitClient
	ac.locationClient = locationClient
	ac.coreClient = coreClient
	ac.user = conData.AuthenticatedUser
	ac.ctx = ctx
	return nil
}
