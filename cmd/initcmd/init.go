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
	InitCmd.PersistentFlags().BoolVarP(&flagForce, "force", "f", false, "Replace an existing git repository and update an existing origin without asking")
	InitCmd.PersistentFlags().BoolVar(&flagOnlyRemote, "only-remote", false, "Only create the remote repository and print its clone URL; do not touch the local directory")
	InitCmd.PersistentFlags().BoolVar(&flagOnlyPush, "only-push", false, "Only push the existing local repository to its configured origin; do not create a remote repository")
	InitCmd.MarkFlagsMutuallyExclusive("only-remote", "only-push")
	addSubcommands()
}
