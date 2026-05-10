package initcmd

import "github.com/spf13/cobra"

var gitlabCmd = &cobra.Command{
	Use:   "gitlab",
	Short: "Initialize repo for GitLab",
	Run: func(cmd *cobra.Command, args []string) {
		runProvider(cmd, "gitlab")
	},
}
