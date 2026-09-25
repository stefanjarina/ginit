package prompts

import (
	"reflect"
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
