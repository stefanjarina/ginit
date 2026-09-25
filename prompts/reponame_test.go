package prompts

import (
	"strings"
	"testing"
)

func TestValidateRepoName(t *testing.T) {
	tests := []struct {
		provider string
		name     string
		wantErr  string
	}{
		{"github", "my-repo_1.0", ""},
		{"github", "My Repo", "a space"},
		{"github", "demo.git", "must not end with .git"},
		{"github", "demo.GIT", "must not end with .git"},
		{"github", "..", `must not be ".."`},
		{"github", strings.Repeat("a", 101), "at most 100"},
		{"github", "  ", "required"},
		{"gitea", "My Repo", "a space"},
		{"forgejo", "demo.wiki", "must not end with .wiki"},
		{"gitea", "demo.git", "must not end with .git"},
		{"gitlab", "My Repo", ""},
		{"gitlab", "demo.git", "must not end with .git"},
		{"gitlab", "-demo", "must start with"},
		{"gitlab", "demo/x", `'/'`},
		{"azure", "My Repo", ""},
		{"azure", "demo.git", "must not end with .git"},
		{"azure", "_demo", "must not start with"},
		{"azure", "a:b", `':'`},
		{"bitbucket", "My Repo", ""},
		{"bitbucket", "MyRepo", ""},
		{"bitbucket", "My Repo.git", "must not end with .git"},
		{"bitbucket", "!!!", "at least one letter or digit"},
		{"bitbucket", strings.Repeat("a", 63), "at most 62"},
	}
	for _, tt := range tests {
		err := ValidateRepoName(tt.provider, tt.name)
		switch {
		case tt.wantErr == "" && err != nil:
			t.Errorf("ValidateRepoName(%q, %q) = %v, want nil", tt.provider, tt.name, err)
		case tt.wantErr != "" && err == nil:
			t.Errorf("ValidateRepoName(%q, %q) = nil, want error containing %q", tt.provider, tt.name, tt.wantErr)
		case tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr):
			t.Errorf("ValidateRepoName(%q, %q) = %q, want error containing %q", tt.provider, tt.name, err, tt.wantErr)
		}
	}
}
