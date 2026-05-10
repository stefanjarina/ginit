package prompts

import (
	"github.com/charmbracelet/huh"
)

func AskGithubAuthenticationMethodGroup(authentication *string) *huh.Group {
	// TODO: get list of available groups from API

	authenticationChoices := []huh.Option[string]{
		huh.NewOption("Personal Access Token", "token"),
		huh.NewOption("Username & Password", "username_password"),
	}

	return huh.NewGroup(
		huh.NewSelect[string]().Title("Authentication method").Options(authenticationChoices...).Value(authentication),
	)
}
