package prompts

import (
	"github.com/charmbracelet/huh"
)

func AskAzureAuthenticationMethodGroup(authentication *string) *huh.Group {
	// TODO: get list of available groups from API

	authenticationChoices := []huh.Option[string]{
		huh.NewOption("Personal Access Token", "token"),
		huh.NewOption("Username & Password", "username_password"),
	}

	return huh.NewGroup(
		huh.NewSelect[string]().Title("Authentication method").Options(authenticationChoices...).Value(authentication),
	)
}

func AskForAzureOrganizationUrlGroup(orgName *string) *huh.Group {
	return huh.NewGroup(
		huh.NewInput().Title("Enter your organization name (https://dev.azure.com/{yourorgname})").Value(orgName),
	)
}
