package configcmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/stefanjarina/ginit/config"
	"github.com/stefanjarina/ginit/console"
)

var setCmd = &cobra.Command{
	Use:   "set (defaultbranch <branch> | <provider> <key> <value>)",
	Short: "Set a configuration key to a value",
	Long: `Set a configuration key to a value.

Use "defaultbranch <branch>" to set the branch that git init uses.
Otherwise set <key> on <provider> (token, baseurl, or a provider option).`,
	Example: `  ginit config set defaultbranch trunk
  ginit config set github token <token>`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 && config.IsDefaultBranchKey(args[0]) {
			if err := cobra.ExactArgs(2)(cmd, args); err != nil {
				return err
			}
			// Reject an empty branch before touching the loaded config.
			return (&config.Config{}).SetDefaultBranch(args[1])
		}
		return cobra.ExactArgs(3)(cmd, args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		if config.IsDefaultBranchKey(args[0]) {
			err = config.Current.SetDefaultBranch(args[1])
		} else {
			err = config.Current.SetValue(args[0], args[1], args[2])
		}
		if err != nil {
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
