package prompts

import (
	"log"
	"os"
	"slices"

	"github.com/charmbracelet/huh"
)

func GetTokenGroup(token *string) *huh.Group {
	return huh.NewGroup(
		huh.NewInput().Title("Enter your Personal Access Token").EchoMode(huh.EchoModePassword).Value(token),
	)
}

func GetRepoDetailGroup(repository string, repoName *string, description *string, visibility *string) *huh.Group {
	var visibilityChoices []huh.Option[string]

	switch repository {
	case "azure":
		visibilityChoices = []huh.Option[string]{
			huh.NewOption("Private", "private"),
			huh.NewOption("Public", "public"),
		}
	case "github":
		visibilityChoices = []huh.Option[string]{
			huh.NewOption("Private", "private"),
			huh.NewOption("Public", "public"),
		}
	case "gitlab":
		visibilityChoices = []huh.Option[string]{
			huh.NewOption("Private", "private"),
			huh.NewOption("Internal", "internal"),
			huh.NewOption("Public", "public"),
		}
	}

	return huh.NewGroup(
		huh.NewInput().Title("Repository Name").Value(repoName),
		huh.NewInput().Title("Description").Value(description),
		huh.NewSelect[string]().Title("Visibility").Options(visibilityChoices...).Value(visibility),
	)
}

// GetGitIgnoreGroup builds the multi-select prompt for the .gitignore command
// and the init flow. availableTypes is the list returned by gitignore.io.
//
// Order matches novugit's AskForGitignoreDetails: gitignore.io templates first,
// custom files second. The custom-files multiselect is only included when there
// are files to choose from in the current directory.
func GetGitIgnoreGroup(availableTypes []string, filesVal *[]string, typesVal *[]string) *huh.Group {
	defaultFiles := []string{"node_modules"}
	defaultTypes := []string{"windows", "linux", "macos", "node", "dotnetcore", "visualstudiocode", "webstorm+all"}

	var availableTypesOptions []huh.Option[string]
	for _, at := range availableTypes {
		if at == "" {
			continue
		}
		opt := huh.NewOption(at, at)
		if slices.Contains(defaultTypes, at) {
			opt = opt.Selected(true)
		}
		availableTypesOptions = append(availableTypesOptions, opt)
	}

	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	files := getListOfFiles(currentDir)

	var filesOptions []huh.Option[string]
	for _, f := range files {
		if f == "" {
			continue
		}
		opt := huh.NewOption(f, f)
		if slices.Contains(defaultFiles, f) {
			opt = opt.Selected(true)
		}
		filesOptions = append(filesOptions, opt)
	}

	fields := []huh.Field{
		huh.NewMultiSelect[string]().
			Title("Select config names you wish to fetch from https://gitignore.io").
			Options(availableTypesOptions...).
			Value(typesVal),
	}
	if len(filesOptions) > 0 {
		fields = append(fields,
			huh.NewMultiSelect[string]().
				Title("Select the files and/or folders you wish to ignore").
				Options(filesOptions...).
				Value(filesVal),
		)
	}
	return huh.NewGroup(fields...).WithHeight(10)
}

// AskForToken runs a single-input form for a PAT.
func AskForToken(provider string, accessibility bool) (string, error) {
	var token string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Enter your "+provider+" Personal Access Token").
				EchoMode(huh.EchoModePassword).
				Value(&token),
		),
	)
	if err := form.WithAccessible(accessibility).WithLayout(huh.LayoutStack).Run(); err != nil {
		return "", err
	}
	return token, nil
}

// AskForBaseUrl prompts for a provider's API base URL (used for self-hosted Gitea/Forgejo/GitLab).
func AskForBaseUrl(provider string, accessibility bool) (string, error) {
	var url string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Enter " + provider + " base URL").Value(&url),
		),
	)
	if err := form.WithAccessible(accessibility).WithLayout(huh.LayoutStack).Run(); err != nil {
		return "", err
	}
	return url, nil
}

// AskToPushToRemote returns whether the user wants to push the initial commit.
func AskToPushToRemote(accessibility bool) (bool, error) {
	confirm := true
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().Title("Push to remote now?").Affirmative("Yes").Negative("No").Value(&confirm),
		),
	)
	if err := form.WithAccessible(accessibility).WithLayout(huh.LayoutStack).Run(); err != nil {
		return false, err
	}
	return confirm, nil
}

// AskToDeleteCurrentLocalRepo asks whether to wipe an existing .git directory.
func AskToDeleteCurrentLocalRepo(accessibility bool) (bool, error) {
	confirm := false
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Existing .git directory found. Delete it and start over?").
				Affirmative("Yes").Negative("No").Value(&confirm),
		),
	)
	if err := form.WithAccessible(accessibility).WithLayout(huh.LayoutStack).Run(); err != nil {
		return false, err
	}
	return confirm, nil
}

func getListOfFiles(name string) []string {
	file, err := os.Open(name)
	if err != nil {
		log.Fatalf("failed opening directory: %s", err)
	}
	defer file.Close()

	list, _ := file.Readdirnames(0)
	return list
}
