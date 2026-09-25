package configcmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stefanjarina/ginit/config"
)

var getCmd = &cobra.Command{
	Use:   "get (defaultbranch | <provider> <key>)",
	Short: "Get value of a configuration key",
	Example: `  ginit config get defaultbranch
  ginit config get github baseurl`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 && config.IsDefaultBranchKey(args[0]) {
			return cobra.ExactArgs(1)(cmd, args)
		}
		return cobra.ExactArgs(2)(cmd, args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		if config.IsDefaultBranchKey(args[0]) {
			fmt.Println(config.Current.DefaultBranch)
			return
		}
		fmt.Println(config.Current.GetValue(args[0], args[1]))
	},
}
