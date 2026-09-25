package configcmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/stefanjarina/ginit/config"
	"github.com/stefanjarina/ginit/console"
)

var setCmd = &cobra.Command{
	Use:   "set <provider> <key> <value>",
	Short: "Set a configuration key to a value",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		repo, key, value := args[0], args[1], args[2]

		if err := config.Current.SetValue(repo, key, value); err != nil {
			console.Error("set value", err)
			os.Exit(1)
		}
		if err := config.Save(config.CurrentPath, config.Current); err != nil {
			console.Error("save config", err)
			os.Exit(1)
		}
		console.Success("Saved")
	},
}
