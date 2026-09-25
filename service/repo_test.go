package service

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stefanjarina/ginit/console"
)

type fakeGitignore struct {
	list    []string
	listErr error
}

func (f *fakeGitignore) List() ([]string, error) { return f.list, f.listErr }

func (f *fakeGitignore) FetchConfig(names []string) (string, error) {
	return "", errors.New("unexpected FetchConfig call")
}

func TestFetchGitignoreListContinuesOnError(t *testing.T) {
	console.Accessible = true
	r := &RepoService{GitignoreIo: &fakeGitignore{listErr: errors.New("gitignore.io unavailable")}}

	if got := r.fetchGitignoreList(); len(got) != 0 {
		t.Fatalf("fetchGitignoreList() = %#v, want empty list", got)
	}
}

func TestFetchGitignoreListReturnsTemplates(t *testing.T) {
	console.Accessible = true
	want := []string{"go", "node"}
	r := &RepoService{GitignoreIo: &fakeGitignore{list: want}}

	if got := r.fetchGitignoreList(); !reflect.DeepEqual(got, want) {
		t.Fatalf("fetchGitignoreList() = %#v, want %#v", got, want)
	}
}

func TestCreateGitignoreFileWithoutTemplates(t *testing.T) {
	console.Accessible = true
	dir := t.TempDir()
	t.Chdir(dir)
	r := &RepoService{GitignoreIo: &fakeGitignore{listErr: errors.New("gitignore.io unavailable")}}

	pi := &ProjectInfo{ExcludedLocalFiles: []string{"node_modules"}}
	if err := r.CreateGitignoreFile(pi); err != nil {
		t.Fatalf("CreateGitignoreFile() error = %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if !strings.Contains(string(content), "node_modules") {
		t.Errorf(".gitignore = %q, want custom entry node_modules", content)
	}
}
