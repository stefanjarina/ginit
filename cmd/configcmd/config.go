package configcmd

import "github.com/spf13/cobra"

var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func addSubcommands() {
	ConfigCmd.AddCommand(allCmd)
	ConfigCmd.AddCommand(getCmd)
	ConfigCmd.AddCommand(setCmd)
	ConfigCmd.AddCommand(removeCmd)
}

func init() {
	addSubcommands()
}
