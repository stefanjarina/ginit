package initcmd

import "github.com/spf13/cobra"

var githubCmd = &cobra.Command{
	Use:   "github",
	Short: "Initialize repo for GitHub",
	Run: func(cmd *cobra.Command, args []string) {
		runProvider(cmd, "github")
	},
}
