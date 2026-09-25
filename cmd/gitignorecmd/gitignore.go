package gitignorecmd

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/stefanjarina/ginit/api/gitignoreio"
	"github.com/stefanjarina/ginit/console"
	"github.com/stefanjarina/ginit/gitops"
	"github.com/stefanjarina/ginit/prompts"
)

var GitignoreCmd = &cobra.Command{
	Use:   "gitignore",
	Short: "Generate .gitignore from gitignore.io templates and custom files",
	Run: func(cmd *cobra.Command, args []string) {
		accessible, _ := cmd.Root().PersistentFlags().GetBool("accessibility")
		cwd, err := os.Getwd()
		if err != nil {
			console.Error("get working directory", err)
			os.Exit(1)
		}
		if _, statErr := os.Stat(filepath.Join(cwd, ".gitignore")); statErr == nil {
			keep, err := prompts.AskToKeepExistingGitignore(accessible)
			if err != nil {
				console.Error("prompt", err)
				os.Exit(1)
			}
			if keep {
				return
			}
		}

		client := gitignoreio.NewClient()
		var availableTypes []string
		if err := console.Run("Fetching gitignore.io templates", func() error {
			list, err := client.List()
			availableTypes = list
			return err
		}); err != nil {
			os.Exit(1)
		}

		var customFiles, ignoreTypes []string
		group, err := prompts.GetGitIgnoreGroup(availableTypes, &customFiles, &ignoreTypes)
		if err != nil {
			console.Error("list files", err)
			os.Exit(1)
		}
		form := huh.NewForm(group)
		if err := form.WithAccessible(accessible).WithLayout(huh.LayoutStack).Run(); err != nil {
			console.Error("prompt", err)
			os.Exit(1)
		}

		if err := console.Run("Writing .gitignore", func() error {
			var ioContent string
			if len(ignoreTypes) > 0 {
				c, err := client.FetchConfig(ignoreTypes)
				if err != nil {
					return err
				}
				ioContent = c
			}
			return gitops.WriteGitignore(cwd, customFiles, ioContent)
		}); err != nil {
			os.Exit(1)
		}
	},
}
