package prompts

import (
	"log"
	"os"
	"slices"

	"github.com/charmbracelet/huh"
	"github.com/stefanjarina/ginit/api/gitignoreio"
)

func GetTokenGroup(token *string) *huh.Group {
	return huh.NewGroup(
		huh.NewInput().Title("Enter your Personal Access Token").Value(token),
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

	group := huh.NewGroup(
		huh.NewInput().Title("Repository Name").Value(repoName),
		huh.NewInput().Title("Description").Value(description),
		huh.NewSelect[string]().Title("Visibility").Options(visibilityChoices...).Value(visibility),
	)

	return group
}

func GetGitIgnoreGroup(filesVal *[]string, typesVal *[]string) *huh.Group {
	defaultFiles := []string{"package.json"}
	defaultTypes := []string{"windows", "linux", "macos", "node", "dotnetcore", "visualstudiocode", "webstorm+all"}

	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	files := getListOfFiles(currentDir)

	giClient := gitignoreio.NewClient()

	// Get the list of available types.
	availableTypes, err := giClient.List()
	if err != nil {
		log.Fatal(err)
	}

	var filesOptions []huh.Option[string]
	for _, f := range files {
		if f == "" {
			continue
		}
		var newOption huh.Option[string]
		if slices.Contains(defaultFiles, f) {
			newOption = huh.NewOption(f, f).Selected(true)
		} else {
			newOption = huh.NewOption(f, f)
		}

		filesOptions = append(filesOptions, newOption)
	}

	availableTypesOptions := make([]huh.Option[string], len(availableTypes))
	for _, at := range availableTypes {
		if at == "" {
			continue
		}
		var newOption huh.Option[string]
		if slices.Contains(defaultTypes, at) {
			newOption = huh.NewOption(at, at).Selected(true)
		} else {
			newOption = huh.NewOption(at, at)
		}
		availableTypesOptions = append(availableTypesOptions, newOption)
	}

	group := huh.NewGroup(
		huh.NewMultiSelect[string]().Title("Files/Folders to add to .gitignore custom section").Options(filesOptions...).Value(filesVal),
		huh.NewMultiSelect[string]().Title("Select config names you wish to fetch from https://gitignore.io").Options(availableTypesOptions...).Value(typesVal),
	).WithHeight(10)

	return group
}

func getListOfFiles(name string) []string {
	file, err := os.Open(name)
	if err != nil {
		log.Fatalf("failed opening directory: %s", err)
	}
	defer file.Close()

	list, _ := file.Readdirnames(0) // 0 to read all files and folders

	return list
}
