package prompts

import (
	"strings"
	"testing"
)

func TestProviderPromptsFailWithNothingToPick(t *testing.T) {
	tests := []struct {
		name     string
		ask      func() error
		provider string
		missing  string
	}{
		{"azure project", func() error { _, err := AskForAzureProject(nil, true); return err }, "azure", "project"},
		{"bitbucket workspace", func() error { _, err := AskForBitbucketWorkspace(nil, true); return err }, "bitbucket", "workspace"},
		{"bitbucket project", func() error { _, err := AskForBitbucketProject(nil, true); return err }, "bitbucket", "project"},
		{"github owner", func() error { _, err := AskForGithubOwner(nil, true); return err }, "github", "account"},
		{"gitlab group", func() error { _, err := AskForGitlabGroup(nil, true); return err }, "gitlab", "group"},
		{"gitea owner", func() error { _, err := AskForGiteaOwner("gitea", nil, true); return err }, "gitea", "owner"},
		{"forgejo owner", func() error { _, err := AskForGiteaOwner("forgejo", nil, true); return err }, "forgejo", "owner"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ask()
			if err == nil {
				t.Fatal("error = nil, want an error for an empty list")
			}
			msg := err.Error()
			if !strings.Contains(msg, "["+tt.provider+"]") || !strings.Contains(msg, tt.missing) {
				t.Errorf("error = %q, want it to name %q and %q", msg, tt.provider, tt.missing)
			}
		})
	}
}
