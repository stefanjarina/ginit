package configcmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stefanjarina/ginit/config"
	"github.com/stefanjarina/ginit/console"
)

var removeCmd = &cobra.Command{
	Use:   "remove (protocol | <provider> <key>)",
	Short: "Remove a configuration key",
	Long: `Remove a configuration key.

Removing "protocol" restores the platform default: ssh, or https on Windows.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 && config.IsDefaultBranchKey(args[0]) {
			return fmt.Errorf("%s cannot be removed; change it with \"ginit config set %s <branch>\"", config.DefaultBranchKey, config.DefaultBranchKey)
		}
		if len(args) > 0 && config.IsProtocolKey(args[0]) {
			return cobra.ExactArgs(1)(cmd, args)
		}
		return cobra.ExactArgs(2)(cmd, args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		if config.IsProtocolKey(args[0]) {
			config.Current.Protocol = ""
		} else if err := config.Current.RemoveValue(args[0], args[1]); err != nil {
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
