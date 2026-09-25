package prompts

import (
	"github.com/charmbracelet/huh"
	"github.com/stefanjarina/ginit/api"
	gerrors "github.com/stefanjarina/ginit/errors"
)

// AskForGithubOwner asks which account the repository is created under. The
// prompt is skipped when there is only one choice. An empty list is an error,
// so a create is never sent without an owner.
func AskForGithubOwner(owners []api.GithubOwner, accessibility bool) (api.GithubOwner, error) {
	if len(owners) == 0 {
		return api.GithubOwner{}, gerrors.NewProvider("github", "no GitHub account to create the repo under", nil)
	}
	if len(owners) == 1 {
		return owners[0], nil
	}
	opts := make([]huh.Option[string], len(owners))
	for i, owner := range owners {
		label := owner.Login
		if owner.Name != "" && owner.Name != owner.Login {
			label = owner.Name + " (" + owner.Login + ")"
		}
		if !owner.IsOrg {
			label += " - personal account"
		}
		opts[i] = huh.NewOption(label, owner.Login)
	}
	var picked string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Title("Pick a GitHub owner").Options(opts...).Value(&picked),
		),
	)
	if err := form.WithAccessible(accessibility).WithLayout(huh.LayoutStack).Run(); err != nil {
		return api.GithubOwner{}, err
	}
	return findGithubOwner(owners, picked)
}

// findGithubOwner returns the owner whose login was picked. A login that is
// not in the list is an error rather than a zero owner.
func findGithubOwner(owners []api.GithubOwner, login string) (api.GithubOwner, error) {
	for _, owner := range owners {
		if owner.Login == login {
			return owner, nil
		}
	}
	return api.GithubOwner{}, gerrors.NewProvider("github", "picked GitHub owner "+login+" is not in the list of owners", nil)
}
