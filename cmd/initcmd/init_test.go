package initcmd

import "testing"

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
