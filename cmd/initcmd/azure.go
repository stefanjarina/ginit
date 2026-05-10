package initcmd

import "github.com/spf13/cobra"

var azureCmd = &cobra.Command{
	Use:   "azure",
	Short: "Initialize repo for Azure DevOps",
	Run: func(cmd *cobra.Command, args []string) {
		runProvider(cmd, "azure")
	},
}
