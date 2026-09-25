package prompts

import (
	"github.com/charmbracelet/huh"
	"github.com/stefanjarina/ginit/api"
	gerrors "github.com/stefanjarina/ginit/errors"
)

// AskForGiteaOwner lets the user pick the user or organization that owns the
// new repo. An empty list is an error.
func AskForGiteaOwner(provider string, owners []api.GiteaOwner, accessibility bool) (api.GiteaOwner, error) {
	if len(owners) == 0 {
		return api.GiteaOwner{}, gerrors.NewProvider(provider, "no "+provider+" owner to create the repo under", nil)
	}
	opts := make([]huh.Option[string], len(owners))
	for i, owner := range owners {
		label := owner.Name
		if label == "" {
			label = owner.Username
		}
		opts[i] = huh.NewOption(label, owner.Username)
	}
	var picked string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Title("Pick a " + provider + " owner").Options(opts...).Value(&picked),
		),
	)
	if err := form.WithAccessible(accessibility).WithLayout(huh.LayoutStack).Run(); err != nil {
		return api.GiteaOwner{}, err
	}
	for _, owner := range owners {
		if owner.Username == picked {
			return owner, nil
		}
	}
	return api.GiteaOwner{}, nil
}
