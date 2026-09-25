package prompts

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestVisibilityChoices(t *testing.T) {
	tests := map[string][]string{
		"github":    {"private", "public"},
		"azure":     {"private", "public"},
		"bitbucket": {"private", "public"},
		"gitlab":    {"private", "internal", "public"},
		"gitea":     {"private", "limited", "public"},
		"forgejo":   {"private", "limited", "public"},
	}
	for provider, want := range tests {
		if got := VisibilityChoices(provider); !reflect.DeepEqual(got, want) {
			t.Errorf("VisibilityChoices(%q) = %#v, want %#v", provider, got, want)
		}
	}
}

func TestGetGitIgnoreGroupWithoutTemplates(t *testing.T) {
	var files, types []string

	t.Chdir(t.TempDir())
	if g := GetGitIgnoreGroup(nil, &files, &types); g != nil {
		t.Errorf("GetGitIgnoreGroup(nil) in empty dir = %v, want nil", g)
	}

	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	if g := GetGitIgnoreGroup(nil, &files, &types); g == nil {
		t.Error("GetGitIgnoreGroup(nil) with local files = nil, want custom-file prompt")
	}
}
