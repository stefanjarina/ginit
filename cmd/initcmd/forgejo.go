package initcmd

import "github.com/spf13/cobra"

var forgejoCmd = &cobra.Command{
	Use:   "forgejo",
	Short: "Initialize repo for Forgejo",
	Run: func(cmd *cobra.Command, args []string) {
		runProvider(cmd, "forgejo")
	},
}
