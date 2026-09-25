package prompts

import (
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/stefanjarina/ginit/api"
	gerrors "github.com/stefanjarina/ginit/errors"
)

// AskForGitlabGroup lets the user pick a GitLab namespace (personal or group)
// and returns its numeric namespace_id, ready for project creation. An empty
// list is an error, so a create is never sent with namespace_id 0.
func AskForGitlabGroup(groups []api.GitlabGroup, accessibility bool) (int, error) {
	if len(groups) == 0 {
		return 0, gerrors.NewProvider("gitlab", "no GitLab group or namespace to create the project in", nil)
	}
	opts := make([]huh.Option[string], len(groups))
	for i, g := range groups {
		opts[i] = huh.NewOption(g.Name, strconv.Itoa(g.ID))
	}
	var picked string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Title("Pick a GitLab namespace").Options(opts...).Value(&picked),
		),
	)
	if err := form.WithAccessible(accessibility).WithLayout(huh.LayoutStack).Run(); err != nil {
		return 0, err
	}
	return strconv.Atoi(picked)
}
