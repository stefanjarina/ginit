package prompts

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stefanjarina/ginit/api"
)

func TestAskForGithubOwnerSingleChoiceSkipsPrompt(t *testing.T) {
	user := api.GithubOwner{Login: "alice", Visibilities: []string{"private", "public"}}
	got, err := AskForGithubOwner([]api.GithubOwner{user}, true)
	if err != nil {
		t.Fatalf("AskForGithubOwner() error = %v", err)
	}
	if !reflect.DeepEqual(got, user) {
		t.Errorf("AskForGithubOwner() = %#v, want %#v", got, user)
	}
}

func TestFindGithubOwnerUnmatchedSelection(t *testing.T) {
	owners := []api.GithubOwner{{Login: "alice"}, {Login: "acme", IsOrg: true}}
	got, err := findGithubOwner(owners, "bob")
	if err == nil {
		t.Fatalf("findGithubOwner() = %#v, want an error for an unknown login", got)
	}
	if msg := err.Error(); !strings.Contains(msg, "[github]") || !strings.Contains(msg, "bob") {
		t.Errorf("error = %q, want it to name github and the picked login", msg)
	}
}

func TestFindGithubOwnerMatch(t *testing.T) {
	owners := []api.GithubOwner{{Login: "alice"}, {Login: "acme", IsOrg: true}}
	got, err := findGithubOwner(owners, "acme")
	if err != nil {
		t.Fatalf("findGithubOwner() error = %v", err)
	}
	if !reflect.DeepEqual(got, owners[1]) {
		t.Errorf("findGithubOwner() = %#v, want %#v", got, owners[1])
	}
}
