package service

import (
	"errors"
	"os"
	"os/exec"
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

func TestCommitLocalGitSensitiveFiles(t *testing.T) {
	console.Accessible = true
	for _, tc := range []struct {
		name       string
		answer     bool
		wantCommit bool
	}{
		{name: "confirm", answer: true, wantCommit: true},
		{name: "decline", answer: false, wantCommit: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateGit(t)
			t.Setenv("GIT_AUTHOR_NAME", "Test")
			t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
			t.Setenv("GIT_COMMITTER_NAME", "Test")
			t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
			dir := t.TempDir()
			for name, content := range map[string]string{
				".env":         "TOKEN=secret\n",
				".env.example": "TOKEN=\n",
				"main.go":      "package main\n",
			} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			svc := newTestService()
			if err := svc.PrepareLocalGit(dir); err != nil {
				t.Fatalf("PrepareLocalGit() error = %v", err)
			}
			var asked []string
			svc.ConfirmSensitiveFiles = func(paths []string) (bool, error) {
				asked = paths
				return tc.answer, nil
			}

			err := svc.CommitLocalGit(dir)
			if !reflect.DeepEqual(asked, []string{".env"}) {
				t.Errorf("asked about %q, want [.env]", asked)
			}
			committed := exec.Command("git", "-C", dir, "rev-parse", "--verify", "--quiet", "HEAD").Run() == nil
			if committed != tc.wantCommit {
				t.Errorf("commit created = %v, want %v", committed, tc.wantCommit)
			}
			if tc.wantCommit && err != nil {
				t.Fatalf("CommitLocalGit() error = %v", err)
			}
			if !tc.wantCommit && (err == nil || !strings.Contains(err.Error(), "secrets")) {
				t.Fatalf("CommitLocalGit() error = %v, want cancelled error", err)
			}
		})
	}
}

func TestCommitLocalGitNoSensitiveFilesDoesNotAsk(t *testing.T) {
	console.Accessible = true
	isolateGit(t)
	t.Setenv("GIT_AUTHOR_NAME", "Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env.example"), []byte("TOKEN=\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := newTestService()
	if err := svc.PrepareLocalGit(dir); err != nil {
		t.Fatalf("PrepareLocalGit() error = %v", err)
	}
	svc.ConfirmSensitiveFiles = func([]string) (bool, error) {
		t.Error("ConfirmSensitiveFiles called without sensitive files")
		return false, nil
	}
	if err := svc.CommitLocalGit(dir); err != nil {
		t.Fatalf("CommitLocalGit() error = %v", err)
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

// repoWithOrigin creates a git repository in a temp dir, makes it the working
// directory and, when origin is not empty, adds it as the origin remote.
func repoWithOrigin(t *testing.T, origin string) string {
	t.Helper()
	isolateGit(t)
	dir := t.TempDir()
	t.Chdir(dir)
	if err := gitops.Init(dir, "main"); err != nil {
		t.Fatal(err)
	}
	if origin != "" {
		if err := gitops.AddRemote(dir, "origin", origin); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// recordingConfirm returns a ConfirmRemoteUpdate that records whether it was
// called and answers with answer.
func recordingConfirm(answer bool, called *bool) func(string, string) (bool, error) {
	return func(string, string) (bool, error) {
		*called = true
		return answer, nil
	}
}

const (
	oldOrigin = "git@example.com:me/old.git"
	newOrigin = "git@example.com:me/new.git"
)

func TestCreateRemote(t *testing.T) {
	console.Accessible = true

	tests := []struct {
		name       string
		existing   string
		force      bool
		answer     bool
		wantAsked  bool
		wantURL    string
		wantErrSub string
	}{
		{name: "missing origin is added", wantURL: newOrigin},
		{name: "identical origin is kept", existing: newOrigin, wantURL: newOrigin},
		{name: "identical origin is kept with force", existing: newOrigin, force: true, wantURL: newOrigin},
		{name: "different origin is updated when confirmed", existing: oldOrigin, answer: true, wantAsked: true, wantURL: newOrigin},
		{name: "different origin is kept when declined", existing: oldOrigin, answer: false, wantAsked: true, wantURL: oldOrigin, wantErrSub: "left unchanged"},
		{name: "different origin is updated with force without asking", existing: oldOrigin, force: true, wantURL: newOrigin},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := repoWithOrigin(t, tt.existing)
			asked := false
			svc := newTestService()
			svc.Force = tt.force
			svc.ConfirmRemoteUpdate = recordingConfirm(tt.answer, &asked)

			err := svc.CreateRemote(newOrigin)
			if tt.wantErrSub == "" && err != nil {
				t.Fatalf("CreateRemote() error = %v", err)
			}
			if tt.wantErrSub != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErrSub)) {
				t.Fatalf("CreateRemote() error = %v, want it to contain %q", err, tt.wantErrSub)
			}
			if asked != tt.wantAsked {
				t.Errorf("confirmation asked = %v, want %v", asked, tt.wantAsked)
			}
			if got, err := gitops.RemoteURL(dir, "origin"); err != nil || got != tt.wantURL {
				t.Errorf("origin = %q, %v; want %q", got, err, tt.wantURL)
			}
		})
	}
}

func TestCreateRemotePromptError(t *testing.T) {
	console.Accessible = true
	dir := repoWithOrigin(t, oldOrigin)
	svc := newTestService()
	svc.ConfirmRemoteUpdate = func(string, string) (bool, error) { return false, os.ErrClosed }

	if err := svc.CreateRemote(newOrigin); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("CreateRemote() error = %v, want prompt error", err)
	}
	if got, _ := gitops.RemoteURL(dir, "origin"); got != oldOrigin {
		t.Errorf("origin = %q, want unchanged %q", got, oldOrigin)
	}
}
