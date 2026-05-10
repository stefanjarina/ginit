package configcmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stefanjarina/ginit/config"
	"github.com/stefanjarina/ginit/console"
	"gopkg.in/yaml.v2"
)

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Print configuration (whole file, or one provider with --repo)",
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := cmd.InheritedFlags().GetString("repo")

		var (
			data []byte
			err  error
		)
		if repo == "" {
			data, err = yaml.Marshal(config.Current)
		} else {
			p := config.Current.GetProvider(repo)
			if p == nil {
				console.Error(fmt.Sprintf("unknown provider: %s", repo), nil)
				os.Exit(1)
			}
			data, err = yaml.Marshal(p)
		}
		if err != nil {
			console.Error("marshal config", err)
			os.Exit(1)
		}
		fmt.Print(string(data))
	},
}
