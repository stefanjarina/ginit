package service

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stefanjarina/ginit/api"
)

type fakeGithubOwners struct {
	owners    []api.GithubOwner
	ownersErr error
	user      api.GithubOwner
	userErr   error
}

func (f *fakeGithubOwners) GetOwners() ([]api.GithubOwner, error) { return f.owners, f.ownersErr }
func (f *fakeGithubOwners) UserOwner() (api.GithubOwner, error)   { return f.user, f.userErr }

func TestGithubOwners(t *testing.T) {
	user := api.GithubOwner{Login: "alice", Visibilities: []string{"private", "public"}}
	org := api.GithubOwner{Login: "acme", IsOrg: true, Visibilities: []string{"private", "internal", "public"}}

	got, err := githubOwners(&fakeGithubOwners{owners: []api.GithubOwner{user, org}})
	if err != nil || !reflect.DeepEqual(got, []api.GithubOwner{user, org}) {
		t.Errorf("githubOwners() = %#v, %v, want user and org", got, err)
	}

	// A token that cannot list organizations still creates user repositories.
	got, err = githubOwners(&fakeGithubOwners{ownersErr: errors.New("403"), user: user})
	if err != nil || !reflect.DeepEqual(got, []api.GithubOwner{user}) {
		t.Errorf("githubOwners() on list error = %#v, %v, want user only", got, err)
	}

	if _, err := githubOwners(&fakeGithubOwners{ownersErr: errors.New("403"), userErr: errors.New("not authenticated")}); err == nil {
		t.Error("githubOwners() without user = nil error, want error")
	}
}
