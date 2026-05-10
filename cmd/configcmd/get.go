package configcmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stefanjarina/ginit/config"
)

var getCmd = &cobra.Command{
	Use:   "get <provider> <key>",
	Short: "Get value of a configuration key",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(config.Current.GetValue(args[0], args[1]))
	},
}
