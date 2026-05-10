package initcmd

import (
	"github.com/spf13/cobra"
	"github.com/stefanjarina/ginit/api"
)

var gitlabCmd = &cobra.Command{
	Use:   "gitlab [repo_name] [description]",
	Short: "Initialize repo for Gitlab",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		api.GetAnswers("gitlab", "")
	},
}

func init() {
}
