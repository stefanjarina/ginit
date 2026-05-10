package initcmd

import "github.com/spf13/cobra"

var (
	flagForce      bool
	flagOnlyRemote bool
	flagOnlyPush   bool
)

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize repo",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func addSubcommands() {
	InitCmd.AddCommand(githubCmd)
	InitCmd.AddCommand(azureCmd)
	InitCmd.AddCommand(gitlabCmd)
	InitCmd.AddCommand(bitbucketCmd)
	InitCmd.AddCommand(giteaCmd)
	InitCmd.AddCommand(forgejoCmd)
}

func init() {
	InitCmd.PersistentFlags().BoolVarP(&flagForce, "force", "f", false, "Force init even if a git repository already exists")
	InitCmd.PersistentFlags().BoolVar(&flagOnlyRemote, "only-remote", false, "Only create the remote repository, skip local git")
	InitCmd.PersistentFlags().BoolVar(&flagOnlyPush, "only-push", false, "Only push the existing local repository to its remote")
	addSubcommands()
}
