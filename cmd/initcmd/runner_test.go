package initcmd

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/stefanjarina/ginit/service"
)

// fakeSteps records which service steps ran; nothing touches the network.
type fakeSteps struct {
	calls    []string
	remoteTo string
}

func (f *fakeSteps) CreateRemoteRepo(provider string) (*service.ProjectInfo, error) {
	f.calls = append(f.calls, "CreateRemoteRepo:"+provider)
	return &service.ProjectInfo{Name: "demo", RemoteUrl: "git@example.com:me/demo.git"}, nil
}

func (f *fakeSteps) CreateGitignoreFile(*service.ProjectInfo) error {
	f.calls = append(f.calls, "CreateGitignoreFile")
	return nil
}

func (f *fakeSteps) InitializeLocalGit() error {
	f.calls = append(f.calls, "InitializeLocalGit")
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

type fakeGit struct {
	hasGitDir bool
	isRepo    bool
	origin    string // empty means no origin configured
	removed   bool
}

func newRunner(svc initSteps, g *fakeGit, out *bytes.Buffer) *runner {
	return &runner{
		svc:          svc,
		dir:          "/work/demo",
		out:          out,
		force:        true,
		hasGitDir:    func(string) bool { return g.hasGitDir },
		removeGitDir: func(string) error { g.removed = true; return nil },
		isRepository: func(string) bool { return g.isRepo },
		remoteURL: func(_, name string) (string, error) {
			if name != "origin" || g.origin == "" {
				return "", errors.New("no such remote")
			}
			return g.origin, nil
		},
		askDelete: func(bool) (bool, error) { return true, nil },
	}
}

func TestOnlyRemoteCreatesRepoAndPrintsCloneURL(t *testing.T) {
	for _, hasGit := range []bool{false, true} {
		svc := &fakeSteps{}
		g := &fakeGit{hasGitDir: hasGit, isRepo: hasGit}
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
	g := &fakeGit{hasGitDir: true, isRepo: true, origin: "git@example.com:me/demo.git"}

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
	g := &fakeGit{hasGitDir: true, isRepo: true}

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
	g := &fakeGit{hasGitDir: true}

	if err := newRunner(svc, g, &bytes.Buffer{}).run("gitlab", modeFull); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"CreateRemoteRepo:gitlab", "CreateGitignoreFile", "InitializeLocalGit", "CreateRemote", "PushToRemote"}
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
