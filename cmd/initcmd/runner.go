package initcmd

import (
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

// runProvider implements the full init flow for one provider.
// Service steps run through console.Run, which already prints their failure;
// console.Error skips those errors, so each failure is reported only once.
func runProvider(cmd *cobra.Command, provider string) {
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

	// 3. Remote create + gitignore + local init + remote-config + push.
	pi, err := svc.CreateRemoteRepo(provider)
	if err != nil {
		console.Error("create remote repo", err)
		os.Exit(1)
	}

	if err := svc.CreateGitignoreFile(pi); err != nil {
		console.Error("create .gitignore", err)
		os.Exit(1)
	}

	if !flagOnlyRemote {
		if err := svc.InitializeLocalGit(); err != nil {
			console.Error("initialize local git", err)
			os.Exit(1)
		}
	}

	if err := svc.CreateRemote(pi.RemoteUrl); err != nil {
		console.Error("configure remote", err)
		os.Exit(1)
	}

	if err := svc.PushToRemote(); err != nil {
		console.Error("push to remote", err)
		os.Exit(1)
	}
}
