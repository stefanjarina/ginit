package gitops

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"
	gerrors "github.com/stefanjarina/ginit/errors"
)

// ErrNothingToCommit is returned by Commit when the index is empty.
var ErrNothingToCommit = errors.New("nothing to commit")

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

// output runs a git command and returns its trimmed stdout.
func output(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// CheckInstalled reports an error when git is not on PATH.
func CheckInstalled() error {
	if _, err := exec.LookPath("git"); err != nil {
		return gerrors.NewHint("git was not found on PATH", "install git and make sure it is on PATH", err)
	}
	return nil
}

// CheckIdentity verifies that a commit author and committer identity is
// configured for dir, either through user.name / user.email or through the
// GIT_AUTHOR_* / GIT_COMMITTER_* environment variables. Run it inside the
// repository that will be committed to so includeIf sections apply.
func CheckIdentity(dir string) error {
	name, _ := output(dir, "config", "--get", "user.name")
	email, _ := output(dir, "config", "--get", "user.email")
	if email == "" {
		email = os.Getenv("EMAIL")
	}

	var missing []string
	if name == "" && (os.Getenv("GIT_AUTHOR_NAME") == "" || os.Getenv("GIT_COMMITTER_NAME") == "") {
		missing = append(missing, "user.name")
	}
	if email == "" && (os.Getenv("GIT_AUTHOR_EMAIL") == "" || os.Getenv("GIT_COMMITTER_EMAIL") == "") {
		missing = append(missing, "user.email")
	}
	if len(missing) == 0 {
		return nil
	}

	cmds := make([]string, 0, len(missing))
	for _, k := range missing {
		cmds = append(cmds, fmt.Sprintf("git config --global %s \"...\"", k))
	}
	return gerrors.NewHint(
		fmt.Sprintf("git commit identity is not configured (missing %s)", strings.Join(missing, ", ")),
		"set it with: "+strings.Join(cmds, " && "), nil)
}

// HasStagedFiles reports whether the index of dir contains any entries.
func HasStagedFiles(dir string) (bool, error) {
	out, err := output(dir, "ls-files", "--cached")
	if err != nil {
		return false, gerrors.New("git ls-files", err)
	}
	return out != "", nil
}

// SigningEnabled reports whether commit.gpgsign is on for dir.
func SigningEnabled(dir string) bool {
	v, _ := output(dir, "config", "--type=bool", "--get", "commit.gpgsign")
	return v == "true"
}

// Commit creates a commit from the index. When commit signing is enabled git
// runs attached to the terminal so the signing program can prompt for a
// passphrase; callers must not have a spinner running in that case.
func Commit(dir, msg string) error {
	staged, err := HasStagedFiles(dir)
	if err != nil {
		return err
	}
	if !staged {
		return ErrNothingToCommit
	}

	if !SigningEnabled(dir) {
		return run(dir, "commit", "-m", msg)
	}

	if err := runInteractive(dir, "commit", "-m", msg); err != nil {
		reason := "the signing program could not complete"
		if !isatty.IsTerminal(os.Stdin.Fd()) {
			reason = "stdin is not a terminal, so the signing program could not prompt for a passphrase"
		}
		return gerrors.NewHint(
			"commit signing (commit.gpgsign) failed: "+reason,
			"make sure your signing key can be unlocked (e.g. export GPG_TTY=$(tty), or cache the passphrase in your agent), or turn off commit.gpgsign", err)
	}
	return nil
}

func AddRemote(dir, name, url string) error {
	return run(dir, "remote", "add", name, url)
}

// SetRemoteURL points an existing remote at url.
func SetRemoteURL(dir, name, url string) error {
	return run(dir, "remote", "set-url", name, url)
}

// RemoteStatus describes how a configured remote relates to a wanted URL.
type RemoteStatus int

const (
	// RemoteMissing means no remote with that name exists.
	RemoteMissing RemoteStatus = iota
	// RemoteMatches means the remote exists and already has the wanted URL.
	RemoteMatches
	// RemoteDiffers means the remote exists with a different URL.
	RemoteDiffers
)

// InspectRemote reports whether the named remote exists in dir and whether
// it points at url. current is the configured URL when the remote exists.
func InspectRemote(dir, name, url string) (status RemoteStatus, current string, err error) {
	names, err := output(dir, "remote")
	if err != nil {
		return RemoteMissing, "", gerrors.New("git remote", err)
	}
	found := false
	for _, n := range strings.Split(names, "\n") {
		if strings.TrimSpace(n) == name {
			found = true
			break
		}
	}
	if !found {
		return RemoteMissing, "", nil
	}
	current, err = RemoteURL(dir, name)
	if err != nil {
		return RemoteMissing, "", err
	}
	if current == url {
		return RemoteMatches, current, nil
	}
	return RemoteDiffers, current, nil
}

// Push pushes branch to remote and sets it as the upstream.
func Push(dir, remote, branch string) error {
	return runInteractive(dir, "push", "--set-upstream", remote, branch)
}

// CurrentBranch returns the name of the branch HEAD points to. It also works
// on an unborn branch (no commits yet) and fails when HEAD is detached.
func CurrentBranch(dir string) (string, error) {
	cmd := exec.Command("git", "symbolic-ref", "--quiet", "--short", "HEAD")
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		// symbolic-ref --quiet exits 1 without output when HEAD is not a symbolic ref.
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 && stderr.Len() == 0 {
			return "", gerrors.NewHint("HEAD is detached, so there is no branch to push",
				"check out a branch first (e.g. git switch <branch>)", nil)
		}
		msg := "git symbolic-ref"
		if stderr.Len() > 0 {
			msg = fmt.Sprintf("%s: %s", msg, bytes.TrimSpace(stderr.Bytes()))
		}
		return "", gerrors.New(msg, err)
	}
	return string(bytes.TrimSpace(out)), nil
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
