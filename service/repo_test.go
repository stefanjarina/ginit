package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stefanjarina/ginit/config"
	"github.com/stefanjarina/ginit/gitops"
)

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
