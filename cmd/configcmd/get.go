package configcmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stefanjarina/ginit/config"
)

var getCmd = &cobra.Command{
	Use:   "get (defaultbranch | protocol | <provider> <key>)",
	Short: "Get value of a configuration key",
	Long: `Get value of a configuration key.

"protocol" prints the effective clone URL protocol, including the platform
default when it is not set.`,
	Example: `  ginit config get defaultbranch
  ginit config get protocol
  ginit config get github baseurl`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 && (config.IsDefaultBranchKey(args[0]) || config.IsProtocolKey(args[0])) {
			return cobra.ExactArgs(1)(cmd, args)
		}
		return cobra.ExactArgs(2)(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		switch {
		case config.IsDefaultBranchKey(args[0]):
			fmt.Println(config.Current.DefaultBranch)
		case config.IsProtocolKey(args[0]):
			protocol, err := config.Current.EffectiveProtocol()
			if err != nil {
				return err
			}
			fmt.Println(protocol)
		default:
			fmt.Println(config.Current.GetValue(args[0], args[1]))
		}
		return nil
	},
}
