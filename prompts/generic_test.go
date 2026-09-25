package prompts

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
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
	if g, err := GetGitIgnoreGroup(nil, &files, &types); err != nil || g != nil {
		t.Errorf("GetGitIgnoreGroup(nil) in empty dir = %v, %v, want nil, nil", g, err)
	}

	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	if g, err := GetGitIgnoreGroup(nil, &files, &types); err != nil || g == nil {
		t.Errorf("GetGitIgnoreGroup(nil) with local files = %v, %v, want custom-file prompt", g, err)
	}
}

func TestGetGitIgnoreGroupMissingWorkingDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the working directory cannot be removed on Windows")
	}
	dir := filepath.Join(t.TempDir(), "gone")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}

	var files, types []string
	g, err := GetGitIgnoreGroup([]string{"go"}, &files, &types)
	if err == nil {
		t.Fatalf("GetGitIgnoreGroup() in removed dir = %v, nil, want error", g)
	}
	if g != nil {
		t.Errorf("GetGitIgnoreGroup() group = %v, want nil on error", g)
	}
}

func TestGetListOfFilesOmitsGitDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := getListOfFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"README.md"}; !reflect.DeepEqual(got, want) {
		t.Errorf("getListOfFiles() = %#v, want %#v", got, want)
	}
}

func TestGetListOfFilesErrors(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	tests := map[string]string{
		"missing directory": filepath.Join(dir, "missing"),
		// Opening a file succeeds, so this exercises the Readdirnames failure.
		"not a directory": file,
	}
	for name, path := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := getListOfFiles(path)
			if err == nil {
				t.Fatalf("getListOfFiles(%q) = %#v, nil, want error", path, got)
			}
			if got != nil {
				t.Errorf("getListOfFiles(%q) list = %#v, want nil on error", path, got)
			}
		})
	}
}
