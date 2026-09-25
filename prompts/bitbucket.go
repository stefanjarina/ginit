package prompts

import (
	"github.com/charmbracelet/huh"
	"github.com/stefanjarina/ginit/api"
	gerrors "github.com/stefanjarina/ginit/errors"
)

// AskForBitbucketWorkspace lets the user pick the workspace the new repo lives
// in. An empty list is an error.
func AskForBitbucketWorkspace(workspaces []api.BitbucketWorkspace, accessibility bool) (string, error) {
	if len(workspaces) == 0 {
		return "", gerrors.NewProvider("bitbucket", "no Bitbucket workspace to create the repo in", nil)
	}
	opts := make([]huh.Option[string], len(workspaces))
	for i, workspace := range workspaces {
		label := workspace.Name
		if label == "" {
			label = workspace.Slug
		}
		opts[i] = huh.NewOption(label, workspace.Slug)
	}
	var picked string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Title("Pick a Bitbucket workspace").Options(opts...).Value(&picked),
		),
	)
	if err := form.WithAccessible(accessibility).WithLayout(huh.LayoutStack).Run(); err != nil {
		return "", err
	}
	return picked, nil
}

// AskForBitbucketProject lets the user pick the project the new repo lives in.
// An empty list is an error.
func AskForBitbucketProject(projects []api.BitbucketProject, accessibility bool) (string, error) {
	if len(projects) == 0 {
		return "", gerrors.NewProvider("bitbucket", "no Bitbucket project to create the repo in", nil)
	}
	opts := make([]huh.Option[string], len(projects))
	for i, project := range projects {
		label := project.Name
		if label == "" {
			label = project.Key
		}
		opts[i] = huh.NewOption(label, project.Key)
	}
	var picked string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Title("Pick a Bitbucket project").Options(opts...).Value(&picked),
		),
	)
	if err := form.WithAccessible(accessibility).WithLayout(huh.LayoutStack).Run(); err != nil {
		return "", err
	}
	return picked, nil
}
