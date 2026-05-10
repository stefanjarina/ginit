package initcmd

import "github.com/spf13/cobra"

var giteaCmd = &cobra.Command{
	Use:   "gitea",
	Short: "Initialize repo for Gitea",
	Run: func(cmd *cobra.Command, args []string) {
		runProvider(cmd, "gitea")
	},
}
