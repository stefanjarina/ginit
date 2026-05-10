package prompts

import (
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/stefanjarina/ginit/api"
)

// AskForGitlabGroup lets the user pick a GitLab namespace (personal or group)
// and returns its numeric namespace_id, ready for project creation.
func AskForGitlabGroup(groups []api.GitlabGroup, accessibility bool) (int, error) {
	if len(groups) == 0 {
		return 0, nil
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
