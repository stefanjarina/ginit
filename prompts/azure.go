package prompts

import "github.com/charmbracelet/huh"

// AskForAzureOrgName asks for the Azure DevOps organization (just the name,
// not the full URL — the API client builds https://dev.azure.com/<org>).
func AskForAzureOrgName(accessibility bool) (string, error) {
	var org string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Azure DevOps organization name (https://dev.azure.com/<this>)").
				Value(&org),
		),
	)
	if err := form.WithAccessible(accessibility).WithLayout(huh.LayoutStack).Run(); err != nil {
		return "", err
	}
	return org, nil
}

// AskForAzureProject lets the user pick which Azure DevOps project the new repo lives in.
func AskForAzureProject(projects []string, accessibility bool) (string, error) {
	if len(projects) == 0 {
		return "", nil
	}
	opts := make([]huh.Option[string], len(projects))
	for i, p := range projects {
		opts[i] = huh.NewOption(p, p)
	}
	var picked string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Title("Pick an Azure DevOps project").Options(opts...).Value(&picked),
		),
	)
	if err := form.WithAccessible(accessibility).WithLayout(huh.LayoutStack).Run(); err != nil {
		return "", err
	}
	return picked, nil
}
