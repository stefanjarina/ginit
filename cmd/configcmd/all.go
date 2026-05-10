package configcmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Lists whole configuration",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := cmd.InheritedFlags().GetString("repo")
		fmt.Println("Repo:", repo)

	},
}

func init() {
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// allCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// allCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
