package prompts

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/charmbracelet/huh"
)

func GetTokenGroup(token *string) *huh.Group {
	return huh.NewGroup(
		huh.NewInput().Title("Enter your Personal Access Token").EchoMode(huh.EchoModePassword).Value(token),
	)
}

func GetRepoDetailGroup(repository string, repoName *string, description *string, visibility *string) *huh.Group {
	choices := VisibilityChoices(repository)
	visibilityChoices := make([]huh.Option[string], 0, len(choices))
	for _, choice := range choices {
		visibilityChoices = append(visibilityChoices, huh.NewOption(toTitle(choice), choice))
	}
	if *visibility == "" {
		*visibility = "private"
	}

	return huh.NewGroup(
		huh.NewInput().Title("Repository Name").Value(repoName).Validate(func(name string) error {
			return ValidateRepoName(repository, name)
		}),
		huh.NewInput().Title("Description").Value(description),
		huh.NewSelect[string]().Title("Visibility").Options(visibilityChoices...).Value(visibility),
	)
}

func VisibilityChoices(repository string) []string {
	switch repository {
	case "gitlab":
		return []string{"private", "internal", "public"}
	case "gitea", "forgejo":
		return []string{"private", "limited", "public"}
	default:
		return []string{"private", "public"}
	}
}

// GetGitIgnoreGroup builds the multi-select prompt for the .gitignore command
// and the init flow. availableTypes is the list returned by gitignore.io.
//
// Order: gitignore.io templates first,
// custom files second. Each multiselect is only included when it has options
// (the template list is empty when gitignore.io is unavailable). Returns a nil
// group when there is nothing to choose from, and an error when the working
// directory cannot be listed.
func GetGitIgnoreGroup(availableTypes []string, filesVal *[]string, typesVal *[]string) (*huh.Group, error) {
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
		return nil, fmt.Errorf("get working directory: %w", err)
	}
	files, err := getListOfFiles(currentDir)
	if err != nil {
		return nil, err
	}

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

	var fields []huh.Field
	if len(availableTypesOptions) > 0 {
		fields = append(fields,
			huh.NewMultiSelect[string]().
				Title("Select config names you wish to fetch from https://gitignore.io").
				Options(availableTypesOptions...).
				Value(typesVal),
		)
	}
	if len(filesOptions) > 0 {
		fields = append(fields,
			huh.NewMultiSelect[string]().
				Title("Select the files and/or folders you wish to ignore").
				Options(filesOptions...).
				Value(filesVal),
		)
	}
	if len(fields) == 0 {
		return nil, nil
	}
	return huh.NewGroup(fields...).WithHeight(10), nil
}

func tokenTitle(provider string) string {
	if provider == "bitbucket" {
		return "Enter your Bitbucket API token (Cloud) or HTTP access token (Data Center / Server)"
	}
	return "Enter your " + provider + " Personal Access Token"
}

// AskForToken runs a single-input form for a PAT.
func AskForToken(provider string, accessibility bool) (string, error) {
	var token string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(tokenTitle(provider)).
				EchoMode(huh.EchoModePassword).
				Value(&token).
				Validate(required(provider + " token")),
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
			huh.NewInput().Title("Enter " + provider + " base URL").Value(&url).Validate(required(provider + " base URL")),
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

func AskToKeepExistingGitignore(accessibility bool) (bool, error) {
	keep := true
	confirm := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title("Existing .gitignore found. Use it as-is?").
			Affirmative("Yes").Negative("No").Value(&keep),
	))
	if err := confirm.WithAccessible(accessibility).WithLayout(huh.LayoutStack).Run(); err != nil {
		return false, err
	}
	return keep, nil
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

// AskToUpdateRemote asks whether a remote that points at current should be
// changed to url.
func AskToUpdateRemote(name, current, url string, accessibility bool) (bool, error) {
	confirm := false
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Remote %q already points to %s. Change it to %s?", name, current, url)).
				Affirmative("Yes").Negative("No").Value(&confirm),
		),
	)
	if err := form.WithAccessible(accessibility).WithLayout(huh.LayoutStack).Run(); err != nil {
		return false, err
	}
	return confirm, nil
}

func required(label string) func(string) error {
	return func(value string) error {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", label)
		}
		return nil
	}
}

func toTitle(value string) string {
	if value == "" {
		return ""
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func getListOfFiles(name string) ([]string, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("open directory: %w", err)
	}
	defer file.Close()

	list, err := file.Readdirnames(0)
	if err != nil {
		return nil, fmt.Errorf("read directory %s: %w", name, err)
	}
	// .git in .gitignore has no effect on git's own metadata directory.
	return slices.DeleteFunc(list, func(n string) bool { return n == ".git" }), nil
}
