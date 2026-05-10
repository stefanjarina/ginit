package configcmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/stefanjarina/ginit/config"
	"github.com/stefanjarina/ginit/console"
)

var removeCmd = &cobra.Command{
	Use:   "remove <key>",
	Short: "Remove a configuration key",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := cmd.InheritedFlags().GetString("repo")
		if err := config.Current.RemoveValue(repo, args[0]); err != nil {
			console.Error("remove value", err)
			os.Exit(1)
		}
		if err := config.Save(config.CurrentPath, config.Current); err != nil {
			console.Error("save config", err)
			os.Exit(1)
		}
		console.Success("Removed")
	},
}
