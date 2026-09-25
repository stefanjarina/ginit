package prompts

import (
	"github.com/charmbracelet/huh"
	"github.com/stefanjarina/ginit/api"
)

// AskForGithubOwner asks which account the repository is created under. The
// prompt is skipped when there is only one choice.
func AskForGithubOwner(owners []api.GithubOwner, accessibility bool) (api.GithubOwner, error) {
	if len(owners) == 0 {
		return api.GithubOwner{}, nil
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
	for _, owner := range owners {
		if owner.Login == picked {
			return owner, nil
		}
	}
	return api.GithubOwner{}, nil
}
