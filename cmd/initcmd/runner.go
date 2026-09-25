package initcmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stefanjarina/ginit/config"
	"github.com/stefanjarina/ginit/console"
	"github.com/stefanjarina/ginit/gitops"
	"github.com/stefanjarina/ginit/prompts"
	"github.com/stefanjarina/ginit/service"
)

// accessibilityFromRoot reads the --accessibility flag set on rootCmd.
func accessibilityFromRoot(cmd *cobra.Command) bool {
	v, _ := cmd.Root().PersistentFlags().GetBool("accessibility")
	return v
}

// runProvider implements the full init flow for one provider
func runProvider(cmd *cobra.Command, provider string) {
	if err := gitops.CheckInstalled(); err != nil {
		console.Error("check git", err)
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		console.Error("get working directory", err)
		os.Exit(1)
	}
	accessible := accessibilityFromRoot(cmd)

	// 1. Existing .git handling.
	if gitops.HasGitDir(cwd) && !flagOnlyPush && !flagOnlyRemote {
		remove := flagForce
		if !remove {
			ok, err := prompts.AskToDeleteCurrentLocalRepo(accessible)
			if err != nil {
				console.Error("prompt", err)
				os.Exit(1)
			}
			remove = ok
		}
		if !remove {
			console.Warning("Operation cancelled. Existing git repository found.")
			return
		}
		if err := gitops.RemoveGitDir(cwd); err != nil {
			console.Error("remove existing .git", err)
			os.Exit(1)
		}
	} else if gitops.HasGitDir(cwd) && (flagOnlyPush || flagOnlyRemote) {
		console.Info("Existing git repository found. --only-* option detected. Proceeding.")
	}

	svc := service.New(config.Current, config.CurrentPath, accessible)

	// 2. --only-push: just push and exit.
	if flagOnlyPush {
		if err := svc.PushToRemote(); err != nil {
			console.Error("push to remote", err)
			os.Exit(1)
		}
		return
	}

	// 3. Local git init + identity check, before any prompt or provider call.
	initLocal := !flagOnlyRemote
	if initLocal {
		if err := svc.PrepareLocalGit(cwd); err != nil {
			_ = gitops.RemoveGitDir(cwd)
			console.Error("initialize local git", err)
			os.Exit(1)
		}
	}

	// 4. After the prompts and before the create call: write .gitignore and
	// create the initial commit, so a local failure leaves no remote behind.
	var localErr error
	svc.BeforeCreate = func(pi *service.ProjectInfo) error {
		localErr = prepareBeforeCreate(svc, pi, cwd, initLocal)
		return localErr
	}

	pi, err := svc.CreateRemoteRepo(provider)
	if err != nil {
		if initLocal {
			_ = gitops.RemoveGitDir(cwd)
		}
		if localErr != nil {
			console.Error("prepare local repository (no remote repository was created)", err)
		} else {
			console.Error("create remote repo", err)
		}
		os.Exit(1)
	}

	// 5. Remote exists from here on; failures must name it.
	if err := svc.CreateRemote(pi.RemoteUrl); err != nil {
		failAfterCreate(provider, pi, "configure remote", err)
	}

	if err := svc.PushToRemote(); err != nil {
		failAfterCreate(provider, pi, "push to remote", err)
	}
}

// prepareBeforeCreate writes .gitignore and, unless only the remote is
// requested, commits the working tree. It runs before the provider create call.
func prepareBeforeCreate(svc *service.RepoService, pi *service.ProjectInfo, dir string, commit bool) error {
	if err := svc.CreateGitignoreFile(pi); err != nil {
		return err
	}
	if !commit {
		return nil
	}
	return svc.CommitLocalGit(dir)
}

// failAfterCreate reports a failure that happened after the repository was
// created on the host, names the repository that was left behind, and exits.
func failAfterCreate(provider string, pi *service.ProjectInfo, step string, err error) {
	console.Error(step, err)
	console.Warning(fmt.Sprintf(
		"The repository %q was already created on %s and was left in place: %s\nFix the problem above and push to it yourself (add it as the origin remote if it is missing), or delete it on %s.",
		pi.Name, provider, pi.RemoteUrl, provider))
	os.Exit(1)
}
