package initcmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/stefanjarina/ginit/config"
	"github.com/stefanjarina/ginit/console"
	"github.com/stefanjarina/ginit/gitops"
	"github.com/stefanjarina/ginit/prompts"
	"github.com/stefanjarina/ginit/service"
)

// localPreparer writes .gitignore and creates the initial commit.
type localPreparer interface {
	CreateGitignoreFile(pi *service.ProjectInfo) error
	CommitLocalGit(dir string) error
}

// initSteps is the subset of service.RepoService the init flow drives.
type initSteps interface {
	localPreparer
	PrepareLocalGit(dir string) error
	SetBeforeCreate(fn func(pi *service.ProjectInfo) error)
	CreateRemoteRepo(provider string) (*service.ProjectInfo, error)
	CreateRemote(remoteUrl string) error
	PushToRemote() error
	PushInitialBranch() error
}

// repoSteps adapts *service.RepoService to initSteps.
type repoSteps struct{ *service.RepoService }

func (s repoSteps) SetBeforeCreate(fn func(pi *service.ProjectInfo) error) { s.BeforeCreate = fn }

type initMode int

const (
	modeFull initMode = iota
	modeOnlyRemote
	modeOnlyPush
)

// runner holds everything one init invocation needs, so tests can swap
// out the service and git/prompt side effects.
type runner struct {
	svc        initSteps
	dir        string
	out        io.Writer
	force      bool
	accessible bool

	findGit      func(dir string) gitops.GitEntry
	removeGitDir func(dir string) error
	isRepository func(dir string) bool
	remoteURL    func(dir, name string) (string, error)
	askDelete    func(isFile, accessible bool) (bool, error)
}

// stepError names the step that failed; the name is what the user sees.
// warning, when set, is printed after the error.
type stepError struct {
	step    string
	err     error
	warning string
}

func (e *stepError) Error() string { return fmt.Sprintf("%s: %v", e.step, e.err) }
func (e *stepError) Unwrap() error { return e.err }

func fail(step string, err error) error { return &stepError{step: step, err: err} }

// accessibilityFromRoot reads the --accessibility flag set on rootCmd.
func accessibilityFromRoot(cmd *cobra.Command) bool {
	v, _ := cmd.Root().PersistentFlags().GetBool("accessibility")
	return v
}

func modeFromFlags() initMode {
	switch {
	case flagOnlyRemote:
		return modeOnlyRemote
	case flagOnlyPush:
		return modeOnlyPush
	default:
		return modeFull
	}
}

// runProvider wires the real dependencies and runs the init flow for one provider.
// Service steps run through console.Run, which already prints their failure;
// console.Error skips those errors, so each failure is reported only once.
func runProvider(cmd *cobra.Command, provider string) {
	mode := modeFromFlags()
	if mode != modeOnlyRemote {
		if err := gitops.CheckInstalled(); err != nil {
			console.Error("check git", err)
			os.Exit(1)
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		console.Error("get working directory", err)
		os.Exit(1)
	}
	accessible := accessibilityFromRoot(cmd)

	svc := service.New(config.Current, config.CurrentPath, accessible)
	svc.Force = flagForce

	r := &runner{
		svc:          repoSteps{svc},
		dir:          cwd,
		out:          os.Stdout,
		force:        flagForce,
		accessible:   accessible,
		findGit:      gitops.FindGit,
		removeGitDir: gitops.RemoveGitDir,
		isRepository: gitops.IsRepository,
		remoteURL:    gitops.RemoteURL,
		askDelete:    prompts.AskToDeleteCurrentLocalRepo,
	}

	if err := r.run(provider, mode); err != nil {
		var se *stepError
		if errors.As(err, &se) {
			console.Error(se.step, se.err)
			if se.warning != "" {
				console.Warning(se.warning)
			}
		} else {
			console.Error(err.Error(), err)
		}
		os.Exit(1)
	}
}

func (r *runner) run(provider string, mode initMode) error {
	switch mode {
	case modeOnlyRemote:
		return r.onlyRemote(provider)
	case modeOnlyPush:
		return r.onlyPush()
	default:
		return r.full(provider)
	}
}

// onlyRemote creates the hosting repository and prints its clone URL.
// It never touches the local directory.
func (r *runner) onlyRemote(provider string) error {
	pi, err := r.svc.CreateRemoteRepo(provider)
	if err != nil {
		return fail("create remote repo", err)
	}
	console.Success("Remote repository created")
	fmt.Fprintf(r.out, "Clone URL: %s\n", pi.RemoteUrl)
	return nil
}

// onlyPush pushes an existing local repository to its configured origin.
func (r *runner) onlyPush() error {
	if !r.isRepository(r.dir) {
		return fail("--only-push requires an existing git repository", fmt.Errorf("%s is not a git repository", r.dir))
	}
	if _, err := r.remoteURL(r.dir, "origin"); err != nil {
		return fail("--only-push requires a remote named 'origin'", err)
	}
	if err := r.svc.PushToRemote(); err != nil {
		return fail("push to remote", err)
	}
	return nil
}

// full initializes local git, writes .gitignore and commits before the remote
// is created, then configures origin and pushes.
func (r *runner) full(provider string) error {
	// A .git file (linked worktree or submodule) counts as an existing
	// repository too: it is removed only once the user has agreed.
	if entry := r.findGit(r.dir); entry != gitops.GitNone {
		remove := r.force
		if !remove {
			ok, err := r.askDelete(entry == gitops.GitFile, r.accessible)
			if err != nil {
				return fail("prompt", err)
			}
			remove = ok
		}
		if !remove {
			console.Warning("Operation cancelled. Existing git repository found.")
			return nil
		}
		if err := r.removeGitDir(r.dir); err != nil {
			return fail("remove existing .git", err)
		}
	}

	// Any .git present from here on was created by this run, so the failure
	// paths below may remove it. Check again rather than assume: if something
	// is still there, it is not ours to delete.
	owned := r.findGit(r.dir) == gitops.GitNone
	cleanup := func() {
		if owned {
			_ = r.removeGitDir(r.dir)
		}
	}

	// Local git init + identity check, before any prompt or provider call.
	if err := r.svc.PrepareLocalGit(r.dir); err != nil {
		cleanup()
		return fail("initialize local git", err)
	}

	// After the prompts and before the create call: write .gitignore and
	// create the initial commit, so a local failure leaves no remote behind.
	var localErr error
	committed := false
	r.svc.SetBeforeCreate(func(pi *service.ProjectInfo) error {
		localErr = prepareBeforeCreate(r.svc, pi, r.dir)
		committed = localErr == nil
		return localErr
	})

	pi, err := r.svc.CreateRemoteRepo(provider)
	if err != nil {
		// Once the initial commit exists, keep it: the user can retry the
		// remote without redoing the local work.
		if committed {
			return &stepError{step: "create remote repo", err: err, warning: fmt.Sprintf(
				"No remote repository was created. The local repository and its initial commit were kept in %s.", r.dir)}
		}
		cleanup()
		if localErr != nil {
			return &stepError{step: "prepare local repository", err: err, warning: "No remote repository was created."}
		}
		return fail("create remote repo", err)
	}

	// Remote exists from here on; failures must name it.
	if err := r.svc.CreateRemote(pi.RemoteUrl); err != nil {
		return failAfterCreate(provider, pi, "configure remote", err)
	}
	if err := r.svc.PushInitialBranch(); err != nil {
		return failAfterCreate(provider, pi, "push to remote", err)
	}
	return nil
}

// prepareBeforeCreate writes .gitignore and commits the working tree.
// It runs before the provider create call.
func prepareBeforeCreate(svc localPreparer, pi *service.ProjectInfo, dir string) error {
	if err := svc.CreateGitignoreFile(pi); err != nil {
		return err
	}
	return svc.CommitLocalGit(dir)
}

// failAfterCreate builds the error for a failure that happened after the
// repository was created on the host, naming the repository left behind.
func failAfterCreate(provider string, pi *service.ProjectInfo, step string, err error) error {
	return &stepError{step: step, err: err, warning: fmt.Sprintf(
		"The repository %q was already created on %s and was left in place: %s\nFix the problem above and push to it yourself (add it as the origin remote if it is missing), or delete it on %s.",
		pi.Name, provider, pi.RemoteUrl, provider)}
}
