package initcmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stefanjarina/ginit/config"
	"github.com/stefanjarina/ginit/gitops"
	"github.com/stefanjarina/ginit/service"
)

func TestPrepareBeforeCreateRefusesEmptyTree(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")

	dir := t.TempDir()
	t.Chdir(dir) // CreateGitignoreFile writes into the working directory.

	svc := service.New(&config.Config{DefaultBranch: "main"}, "", true)
	if err := svc.PrepareLocalGit(dir); err != nil {
		t.Fatalf("PrepareLocalGit() error = %v", err)
	}

	err := prepareBeforeCreate(svc, &service.ProjectInfo{Name: "demo"}, dir)
	if err == nil || !strings.Contains(err.Error(), "nothing to commit") {
		t.Fatalf("prepareBeforeCreate() error = %v, want nothing to commit error", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, ".gitignore")); !os.IsNotExist(statErr) {
		t.Fatalf(".gitignore should not exist, stat error = %v", statErr)
	}
}

// fakeSteps records which service steps ran; nothing touches the network.
type fakeSteps struct {
	calls      []string
	remoteTo   string
	before     func(pi *service.ProjectInfo) error
	prepareErr error
	commitErr  error
}

func (f *fakeSteps) PrepareLocalGit(string) error {
	f.calls = append(f.calls, "PrepareLocalGit")
	return f.prepareErr
}

func (f *fakeSteps) CommitLocalGit(string) error {
	f.calls = append(f.calls, "CommitLocalGit")
	return f.commitErr
}

func (f *fakeSteps) SetBeforeCreate(fn func(pi *service.ProjectInfo) error) { f.before = fn }

func (f *fakeSteps) CreateRemoteRepo(provider string) (*service.ProjectInfo, error) {
	pi := &service.ProjectInfo{Name: "demo", RemoteUrl: "git@example.com:me/demo.git"}
	if f.before != nil {
		if err := f.before(pi); err != nil {
			return nil, err
		}
	}
	f.calls = append(f.calls, "CreateRemoteRepo:"+provider)
	return pi, nil
}

func (f *fakeSteps) CreateGitignoreFile(*service.ProjectInfo) error {
	f.calls = append(f.calls, "CreateGitignoreFile")
	return nil
}

func (f *fakeSteps) CreateRemote(url string) error {
	f.calls = append(f.calls, "CreateRemote")
	f.remoteTo = url
	return nil
}

func (f *fakeSteps) PushToRemote() error {
	f.calls = append(f.calls, "PushToRemote")
	return nil
}

func (f *fakeSteps) PushInitialBranch() error {
	f.calls = append(f.calls, "PushInitialBranch")
	return nil
}

type fakeGit struct {
	entry   gitops.GitEntry
	isRepo  bool
	origin  string // empty means no origin configured
	removed bool
}

func newRunner(svc initSteps, g *fakeGit, out *bytes.Buffer) *runner {
	return &runner{
		svc:          svc,
		dir:          "/work/demo",
		out:          out,
		force:        true,
		findGit:      func(string) gitops.GitEntry { return g.entry },
		removeGitDir: func(string) error { g.removed = true; g.entry = gitops.GitNone; return nil },
		isRepository: func(string) bool { return g.isRepo },
		remoteURL: func(_, name string) (string, error) {
			if name != "origin" || g.origin == "" {
				return "", errors.New("no such remote")
			}
			return g.origin, nil
		},
		askDelete: func(bool, bool) (bool, error) { return true, nil },
	}
}

func TestOnlyRemoteCreatesRepoAndPrintsCloneURL(t *testing.T) {
	for _, hasGit := range []bool{false, true} {
		svc := &fakeSteps{}
		g := &fakeGit{isRepo: hasGit}
		if hasGit {
			g.entry = gitops.GitDirectory
		}
		var out bytes.Buffer

		if err := newRunner(svc, g, &out).run("github", modeOnlyRemote); err != nil {
			t.Fatalf("hasGit=%v: unexpected error: %v", hasGit, err)
		}
		if want := []string{"CreateRemoteRepo:github"}; !reflect.DeepEqual(svc.calls, want) {
			t.Errorf("hasGit=%v: calls = %v, want %v", hasGit, svc.calls, want)
		}
		if g.removed {
			t.Errorf("hasGit=%v: --only-remote must not remove .git", hasGit)
		}
		if !strings.Contains(out.String(), "git@example.com:me/demo.git") {
			t.Errorf("hasGit=%v: clone URL not printed, got %q", hasGit, out.String())
		}
	}
}

func TestOnlyPushPushesExistingRepoWithOrigin(t *testing.T) {
	svc := &fakeSteps{}
	g := &fakeGit{entry: gitops.GitDirectory, isRepo: true, origin: "git@example.com:me/demo.git"}

	if err := newRunner(svc, g, &bytes.Buffer{}).run("github", modeOnlyPush); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []string{"PushToRemote"}; !reflect.DeepEqual(svc.calls, want) {
		t.Errorf("calls = %v, want %v", svc.calls, want)
	}
	if g.removed {
		t.Error("--only-push must not remove .git")
	}
}

func TestOnlyPushRequiresRepository(t *testing.T) {
	svc := &fakeSteps{}
	g := &fakeGit{}

	err := newRunner(svc, g, &bytes.Buffer{}).run("github", modeOnlyPush)
	if err == nil || !strings.Contains(err.Error(), "existing git repository") {
		t.Fatalf("expected missing-repository error, got %v", err)
	}
	if len(svc.calls) != 0 {
		t.Errorf("no service step should run, got %v", svc.calls)
	}
}

func TestOnlyPushRequiresOrigin(t *testing.T) {
	svc := &fakeSteps{}
	g := &fakeGit{entry: gitops.GitDirectory, isRepo: true}

	err := newRunner(svc, g, &bytes.Buffer{}).run("github", modeOnlyPush)
	if err == nil || !strings.Contains(err.Error(), "origin") {
		t.Fatalf("expected missing-origin error, got %v", err)
	}
	if len(svc.calls) != 0 {
		t.Errorf("no service step should run, got %v", svc.calls)
	}
}

func TestFullInitRunsEveryStep(t *testing.T) {
	svc := &fakeSteps{}
	g := &fakeGit{entry: gitops.GitDirectory}

	if err := newRunner(svc, g, &bytes.Buffer{}).run("gitlab", modeFull); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"PrepareLocalGit", "CreateGitignoreFile", "CommitLocalGit", "CreateRemoteRepo:gitlab", "CreateRemote", "PushInitialBranch"}
	if !reflect.DeepEqual(svc.calls, want) {
		t.Errorf("calls = %v, want %v", svc.calls, want)
	}
	if !g.removed {
		t.Error("--force full init should remove the existing .git")
	}
	if svc.remoteTo != "git@example.com:me/demo.git" {
		t.Errorf("origin set to %q", svc.remoteTo)
	}
}

func TestFullInitLocalFailureCreatesNoRemote(t *testing.T) {
	svc := &fakeSteps{commitErr: errors.New("nothing to commit")}
	g := &fakeGit{}

	err := newRunner(svc, g, &bytes.Buffer{}).run("github", modeFull)
	var se *stepError
	if !errors.As(err, &se) || se.step != "prepare local repository" {
		t.Fatalf("expected prepare local repository error, got %v", err)
	}
	for _, c := range svc.calls {
		if strings.HasPrefix(c, "CreateRemoteRepo") {
			t.Fatalf("remote repository created despite local failure: %v", svc.calls)
		}
	}
	if !g.removed {
		t.Error(".git should be removed after a failed local prepare")
	}
}

// writeGitFile makes dir look like a linked worktree: .git is a file pointing
// at a git directory that does not exist, so git commands in dir fail.
func writeGitFile(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, ".git")
	if err := os.WriteFile(p, []byte("gitdir: /nonexistent/.git/worktrees/demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestFullInitAsksBeforeReplacingGitFile(t *testing.T) {
	dir := t.TempDir()
	gitFile := writeGitFile(t, dir)
	svc := &fakeSteps{}
	var askedFile, asked bool

	r := newRunner(svc, &fakeGit{}, &bytes.Buffer{})
	r.dir = dir
	r.force = false
	r.findGit = gitops.FindGit
	r.removeGitDir = gitops.RemoveGitDir
	r.askDelete = func(isFile, _ bool) (bool, error) {
		asked, askedFile = true, isFile
		return false, nil
	}

	if err := r.run("github", modeFull); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !asked || !askedFile {
		t.Fatalf("expected a prompt about a .git file, asked=%v isFile=%v", asked, askedFile)
	}
	if len(svc.calls) != 0 {
		t.Errorf("no step should run after the user declines, got %v", svc.calls)
	}
	if _, err := os.Stat(gitFile); err != nil {
		t.Errorf(".git file should still exist: %v", err)
	}
}

func TestFullInitFailureKeepsGitItDidNotCreate(t *testing.T) {
	dir := t.TempDir()
	gitFile := writeGitFile(t, dir)
	svc := &fakeSteps{prepareErr: errors.New("fatal: not a git repository")}

	// The user agrees, but the .git file survives removal; the failed init
	// must not delete it afterwards, since this run did not create it.
	r := newRunner(svc, &fakeGit{}, &bytes.Buffer{})
	r.dir = dir
	r.findGit = gitops.FindGit
	removals := 0
	r.removeGitDir = func(string) error { removals++; return nil }

	err := r.run("github", modeFull)
	var se *stepError
	if !errors.As(err, &se) || se.step != "initialize local git" {
		t.Fatalf("expected initialize local git error, got %v", err)
	}
	if removals != 1 {
		t.Errorf("removeGitDir called %d times, want 1 (the confirmed replacement only)", removals)
	}
	if _, err := os.Stat(gitFile); err != nil {
		t.Errorf(".git file should still exist: %v", err)
	}
}

func TestFullInitFailureRemovesGitItCreated(t *testing.T) {
	svc := &fakeSteps{prepareErr: errors.New("identity missing")}
	g := &fakeGit{}

	err := newRunner(svc, g, &bytes.Buffer{}).run("github", modeFull)
	var se *stepError
	if !errors.As(err, &se) || se.step != "initialize local git" {
		t.Fatalf("expected initialize local git error, got %v", err)
	}
	if !g.removed {
		t.Error(".git created by this run should be removed after a failed init")
	}
}
