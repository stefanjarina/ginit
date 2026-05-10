package configcmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stefanjarina/ginit/config"
)

var getCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get value of a configuration key",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := cmd.InheritedFlags().GetString("repo")
		fmt.Println(config.Current.GetValue(repo, args[0]))
	},
}
