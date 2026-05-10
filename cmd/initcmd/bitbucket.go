package initcmd

import "github.com/spf13/cobra"

var bitbucketCmd = &cobra.Command{
	Use:   "bitbucket",
	Short: "Initialize repo for Bitbucket",
	Run: func(cmd *cobra.Command, args []string) {
		runProvider(cmd, "bitbucket")
	},
}
