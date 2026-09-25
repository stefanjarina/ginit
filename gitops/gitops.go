package gitops

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	gerrors "github.com/stefanjarina/ginit/errors"
)

// run executes a non-interactive git command, capturing stderr for diagnostics.
func run(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := fmt.Sprintf("git %s", args[0])
		if stderr.Len() > 0 {
			msg = fmt.Sprintf("%s: %s", msg, bytes.TrimSpace(stderr.Bytes()))
		}
		return gerrors.New(msg, err)
	}
	return nil
}

// runInteractive lets git inherit stdin/stdout/stderr — needed for push so that
// SSH passphrase prompts and credential helpers work.
func runInteractive(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return gerrors.New(fmt.Sprintf("git %s", args[0]), err)
	}
	return nil
}

func Init(dir, defaultBranch string) error {
	if defaultBranch == "" {
		defaultBranch = "main"
	}
	// `git init -b` requires git >= 2.28. Acceptable since the user already
	// has git on PATH for everything else.
	return run(dir, "init", "-b", defaultBranch)
}

func AddAll(dir string) error {
	return run(dir, "add", "-A")
}

func Commit(dir, msg string) error {
	return run(dir, "commit", "-m", msg)
}

func AddRemote(dir, name, url string) error {
	return run(dir, "remote", "add", name, url)
}

func Push(dir, remote, branch string) error {
	return runInteractive(dir, "push", "--set-upstream", remote, branch)
}

func HasGitDir(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil && info.IsDir()
}

func RemoveGitDir(dir string) error {
	return os.RemoveAll(filepath.Join(dir, ".git"))
}

// IsRepository reports whether dir is inside a git work tree.
func IsRepository(dir string) bool {
	return run(dir, "rev-parse", "--is-inside-work-tree") == nil
}

// RemoteURL returns the URL configured for the named remote.
func RemoteURL(dir, name string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", name)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := "git remote"
		if stderr.Len() > 0 {
			msg = fmt.Sprintf("%s: %s", msg, bytes.TrimSpace(stderr.Bytes()))
		}
		return "", gerrors.New(msg, err)
	}
	return string(bytes.TrimSpace(out)), nil
}
