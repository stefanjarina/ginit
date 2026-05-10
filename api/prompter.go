package api

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/spf13/viper"
	"github.com/stefanjarina/ginit/prompts"
)

func GetAnswers(repo string, token string) (error, string, string, string, string, []string, []string) {
	accessibility := viper.GetBool("accessibility")

	var (
		name        string
		localFiles  []string
		description string
		visibility  string
		gitignore   []string
		orgUrl      string
		err         error
	)

	wd, _ := os.Getwd()
	name = filepath.Base(wd)

	if token == "" {
		tokenForm := huh.NewForm(
			prompts.GetTokenGroup(&token),
		)
		err = tokenForm.WithAccessible(accessibility).WithLayout(huh.LayoutStack).Run()
		if err != nil || token == "" {
			return err, "", "", "", "", nil, nil
		}
	}

	if repo == "azure" {
		orgUrl, err = GetGithubSpecificAnswers(orgUrl, accessibility)
		if err != nil {
			return err, "", "", "", "", nil, nil
		}
	}

	var providerService RepoService
	if token != "" {
		switch repo {
		case "azure":
			providerService = NewAdoClient(token, orgUrl)
		case "github":
			providerService = NewGithubClient(token)
		default:
			return fmt.Errorf("unsupported repository type: %s", repo), "", "", "", "", nil, nil
		}
		err = providerService.Connect()
		if err != nil {
			return err, "", "", "", "", nil, nil
		}
	}

	form1 := huh.NewForm(
		prompts.GetRepoDetailGroup(repo, &name, &description, &visibility),
	)

	form2 := huh.NewForm(
		prompts.GetGitIgnoreGroup(&localFiles, &gitignore),
	)

	err = form1.WithAccessible(accessibility).WithLayout(huh.LayoutStack).Run()
	if err != nil {
		fmt.Println(err.Error())
		return err, "", "", "", "", nil, nil
	}
	err = form2.WithAccessible(accessibility).WithLayout(huh.LayoutStack).Run()
	if err != nil {
		fmt.Println(err.Error())
		return err, "", "", "", "", nil, nil
	}

	return nil, token, name, description, visibility, localFiles, gitignore
}

func GetGithubSpecificAnswers(orgUrl string, accessibility bool) (string, error) {
	adoSpecificForm := huh.NewForm(
		prompts.AskForAzureOrganizationUrlGroup(&orgUrl),
	)
	err := adoSpecificForm.WithAccessible(accessibility).WithLayout(huh.LayoutStack).Run()
	if err != nil {
		return "", err
	}
	return orgUrl, nil
}
