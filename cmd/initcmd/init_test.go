package initcmd

import (
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestInitExposesAllProviderSubcommands(t *testing.T) {
	want := map[string]bool{
		"azure":     false,
		"bitbucket": false,
		"forgejo":   false,
		"gitea":     false,
		"github":    false,
		"gitlab":    false,
	}
	for _, cmd := range InitCmd.Commands() {
		if _, ok := want[cmd.Name()]; ok {
			want[cmd.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("init subcommand %q not registered", name)
		}
	}
}

func TestOnlyRemoteAndOnlyPushAreMutuallyExclusive(t *testing.T) {
	var ran bool
	probe := &cobra.Command{
		Use: "probe",
		Run: func(cmd *cobra.Command, args []string) { ran = true },
	}
	InitCmd.AddCommand(probe)
	t.Cleanup(func() {
		InitCmd.RemoveCommand(probe)
		flagOnlyRemote, flagOnlyPush = false, false
		for _, name := range []string{"only-remote", "only-push"} {
			f := InitCmd.PersistentFlags().Lookup(name)
			_ = f.Value.Set("false")
			f.Changed = false
		}
		InitCmd.SetArgs(nil)
	})

	InitCmd.SetOut(io.Discard)
	InitCmd.SetErr(io.Discard)
	InitCmd.SetArgs([]string{"probe", "--only-remote", "--only-push"})
	err := InitCmd.Execute()
	if err == nil {
		t.Fatal("expected a usage error when both --only-remote and --only-push are set")
	}
	if !strings.Contains(err.Error(), "only-remote") || !strings.Contains(err.Error(), "only-push") {
		t.Errorf("error should name both flags, got: %v", err)
	}
	if ran {
		t.Error("command ran despite conflicting flags")
	}
}
