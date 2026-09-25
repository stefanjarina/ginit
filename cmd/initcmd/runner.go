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

// initSteps is the subset of service.RepoService the init flow drives.
type initSteps interface {
	CreateRemoteRepo(provider string) (*service.ProjectInfo, error)
	CreateGitignoreFile(pi *service.ProjectInfo) error
	InitializeLocalGit() error
	CreateRemote(remoteUrl string) error
	PushToRemote() error
}

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

	hasGitDir    func(dir string) bool
	removeGitDir func(dir string) error
	isRepository func(dir string) bool
	remoteURL    func(dir, name string) (string, error)
	askDelete    func(accessible bool) (bool, error)
}

// stepError names the step that failed; the name is what the user sees.
type stepError struct {
	step string
	err  error
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
func runProvider(cmd *cobra.Command, provider string) {
	cwd, err := os.Getwd()
	if err != nil {
		console.Error("get working directory", err)
		os.Exit(1)
	}
	accessible := accessibilityFromRoot(cmd)

	r := &runner{
		svc:          service.New(config.Current, config.CurrentPath, accessible),
		dir:          cwd,
		out:          os.Stdout,
		force:        flagForce,
		accessible:   accessible,
		hasGitDir:    gitops.HasGitDir,
		removeGitDir: gitops.RemoveGitDir,
		isRepository: gitops.IsRepository,
		remoteURL:    gitops.RemoteURL,
		askDelete:    prompts.AskToDeleteCurrentLocalRepo,
	}

	if err := r.run(provider, modeFromFlags()); err != nil {
		var se *stepError
		if errors.As(err, &se) {
			console.Error(se.step, se.err)
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

// full creates the remote, writes .gitignore, initializes local git,
// configures origin and pushes.
func (r *runner) full(provider string) error {
	if r.hasGitDir(r.dir) {
		remove := r.force
		if !remove {
			ok, err := r.askDelete(r.accessible)
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

	pi, err := r.svc.CreateRemoteRepo(provider)
	if err != nil {
		return fail("create remote repo", err)
	}
	if err := r.svc.CreateGitignoreFile(pi); err != nil {
		return fail("create .gitignore", err)
	}
	if err := r.svc.InitializeLocalGit(); err != nil {
		return fail("initialize local git", err)
	}
	if err := r.svc.CreateRemote(pi.RemoteUrl); err != nil {
		return fail("configure remote", err)
	}
	if err := r.svc.PushToRemote(); err != nil {
		return fail("push to remote", err)
	}
	return nil
}
