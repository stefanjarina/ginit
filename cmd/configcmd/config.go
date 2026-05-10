package configcmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/stefanjarina/ginit/globals"
	"github.com/stefanjarina/ginit/utils"
)

var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func addSubcommands() {
	ConfigCmd.AddCommand(allCmd)
	ConfigCmd.AddCommand(getCmd)
	ConfigCmd.AddCommand(setCmd)
	ConfigCmd.AddCommand(removeCmd)
}

func init() {
	// `--repo` is shared by all subcommands but only *required* for the
	// mutating ones (get/set/remove). `all` works without it (lists everything).
	enum := utils.NewEnum(globals.SupportedRepos, "")
	ConfigCmd.PersistentFlags().VarP(enum, "repo", "r",
		fmt.Sprintf("Specify repository <%s>", strings.Join(enum.Allowed, "|")))

	for _, sub := range []*cobra.Command{getCmd, setCmd, removeCmd} {
		s := sub
		_ = cobra.MarkFlagRequired(s.InheritedFlags(), "repo")
		// MarkFlagRequired needs the flag to be registered; for inherited flags
		// we re-mark via PreRunE to fail fast.
		prev := s.PreRunE
		s.PreRunE = func(cmd *cobra.Command, args []string) error {
			if v, _ := cmd.InheritedFlags().GetString("repo"); v == "" {
				return fmt.Errorf("required flag(s) \"repo\" not set")
			}
			if prev != nil {
				return prev(cmd, args)
			}
			return nil
		}
	}

	addSubcommands()
}
