package prompts

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/stefanjarina/ginit/model"
)

// AskForProjectInfo runs the repo-detail + gitignore selection forms and
// returns a populated ProjectInfo. availableTypes is the list fetched from
// gitignore.io.
//
// Mirrors novugit's Prompts.AskForProjectInfo: name → description → visibility,
// then if a .gitignore already exists prompts to keep it; otherwise asks for
// gitignore.io templates and (when there are files) custom files to ignore.
func AskForProjectInfo(provider string, availableTypes []string, accessibility bool) (*model.ProjectInfo, error) {
	pi := &model.ProjectInfo{}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	pi.Name = filepath.Base(cwd)

	detailForm := huh.NewForm(
		GetRepoDetailGroup(provider, &pi.Name, &pi.Description, &pi.Visibility),
	)
	if err := detailForm.WithAccessible(accessibility).WithLayout(huh.LayoutStack).Run(); err != nil {
		return nil, err
	}

	// If .gitignore already exists, offer to keep it as-is — matches novugit.
	if _, statErr := os.Stat(filepath.Join(cwd, ".gitignore")); statErr == nil {
		keep, err := AskToKeepExistingGitignore(accessibility)
		if err != nil {
			return nil, err
		}
		if keep {
			return pi, nil
		}
	}

	ignoreForm := huh.NewForm(
		GetGitIgnoreGroup(availableTypes, &pi.ExcludedLocalFiles, &pi.GitIgnoreConfigs),
	)
	if err := ignoreForm.WithAccessible(accessibility).WithLayout(huh.LayoutStack).Run(); err != nil {
		return nil, err
	}

	return pi, nil
}
