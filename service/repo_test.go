package service

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stefanjarina/ginit/config"
	"github.com/stefanjarina/ginit/console"
	"github.com/stefanjarina/ginit/gitops"
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

func isolateGit(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, k := range []string{
		"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL",
		"EMAIL", "GIT_CONFIG_COUNT",
	} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
}

func newTestService() *RepoService {
	return New(&config.Config{DefaultBranch: "main"}, "", true)
}

func TestPrepareLocalGitMissingIdentity(t *testing.T) {
	isolateGit(t)
	dir := t.TempDir()

	err := newTestService().PrepareLocalGit(dir)
	if err == nil {
		t.Fatal("PrepareLocalGit() error = nil, want missing identity error")
	}
	if !strings.Contains(err.Error(), "user.name") {
		t.Fatalf("PrepareLocalGit() error = %q, want it to name user.name", err)
	}
}

func TestCommitLocalGitEmptyTree(t *testing.T) {
	isolateGit(t)
	t.Setenv("GIT_AUTHOR_NAME", "Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	dir := t.TempDir()
	svc := newTestService()

	if err := svc.PrepareLocalGit(dir); err != nil {
		t.Fatalf("PrepareLocalGit() error = %v", err)
	}
	// No templates and no custom files: nothing is written.
	if err := gitops.WriteGitignore(dir, nil, ""); err != nil {
		t.Fatal(err)
	}
	err := svc.CommitLocalGit(dir)
	if err == nil || !strings.Contains(err.Error(), "nothing to commit") {
		t.Fatalf("CommitLocalGit() error = %v, want nothing to commit error", err)
	}
}

func TestBeforeCreateErrorAborts(t *testing.T) {
	svc := newTestService()
	called := false
	svc.BeforeCreate = func(*ProjectInfo) error {
		called = true
		return os.ErrInvalid
	}
	if err := svc.beforeCreate(&ProjectInfo{}); err != os.ErrInvalid || !called {
		t.Fatalf("beforeCreate() = %v (called=%v), want hook error", err, called)
	}
}
