package gitops

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isolateGit points git at empty global/system config and clears identity
// environment variables, so tests do not depend on the machine's setup.
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

func setIdentity(t *testing.T) {
	t.Helper()
	for k, v := range map[string]string{
		"GIT_AUTHOR_NAME":     "Test",
		"GIT_AUTHOR_EMAIL":    "test@example.com",
		"GIT_COMMITTER_NAME":  "Test",
		"GIT_COMMITTER_EMAIL": "test@example.com",
	} {
		t.Setenv(k, v)
	}
}

func TestCheckIdentityMissing(t *testing.T) {
	isolateGit(t)
	dir := t.TempDir()
	if err := Init(dir, "main"); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	err := CheckIdentity(dir)
	if err == nil {
		t.Fatal("CheckIdentity() error = nil, want missing identity error")
	}
	for _, want := range []string{"user.name", "user.email", "git config --global"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("CheckIdentity() error = %q, want it to mention %q", err, want)
		}
	}
}

func TestCheckIdentityFromRepoConfig(t *testing.T) {
	isolateGit(t)
	dir := t.TempDir()
	if err := Init(dir, "main"); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if err := run(dir, "config", "user.name", "Test"); err != nil {
		t.Fatal(err)
	}
	if err := run(dir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := CheckIdentity(dir); err != nil {
		t.Fatalf("CheckIdentity() error = %v", err)
	}
}

func TestCheckIdentityFromEnv(t *testing.T) {
	isolateGit(t)
	setIdentity(t)
	if err := CheckIdentity(t.TempDir()); err != nil {
		t.Fatalf("CheckIdentity() error = %v", err)
	}
}

func TestCommitEmptyTree(t *testing.T) {
	isolateGit(t)
	setIdentity(t)
	dir := t.TempDir()
	if err := Init(dir, "main"); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if err := AddAll(dir); err != nil {
		t.Fatalf("AddAll() error = %v", err)
	}
	if err := Commit(dir, "initial commit"); !errors.Is(err, ErrNothingToCommit) {
		t.Fatalf("Commit() error = %v, want ErrNothingToCommit", err)
	}
}

func TestCommitWithFile(t *testing.T) {
	isolateGit(t)
	setIdentity(t)
	dir := t.TempDir()
	if err := Init(dir, "main"); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := AddAll(dir); err != nil {
		t.Fatalf("AddAll() error = %v", err)
	}
	if err := Commit(dir, "initial commit"); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
}

func TestSigningEnabled(t *testing.T) {
	isolateGit(t)
	dir := t.TempDir()
	if err := Init(dir, "main"); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if SigningEnabled(dir) {
		t.Fatal("SigningEnabled() = true, want false")
	}
	if err := run(dir, "config", "commit.gpgsign", "yes"); err != nil {
		t.Fatal(err)
	}
	if !SigningEnabled(dir) {
		t.Fatal("SigningEnabled() = false, want true")
	}
}

func TestCommitSigningErrorMentionsSigning(t *testing.T) {
	isolateGit(t)
	setIdentity(t)
	dir := t.TempDir()
	if err := Init(dir, "main"); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	// A signing program that always fails stands in for a prompt that cannot run.
	for k, v := range map[string]string{
		"commit.gpgsign": "true",
		"gpg.program":    "false",
	} {
		if err := run(dir, "config", k, v); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := AddAll(dir); err != nil {
		t.Fatalf("AddAll() error = %v", err)
	}
	err := Commit(dir, "initial commit")
	if err == nil || !strings.Contains(err.Error(), "commit signing") {
		t.Fatalf("Commit() error = %v, want commit signing error", err)
	}
}

func TestIsRepositoryAndRemoteURL(t *testing.T) {
	isolateGit(t)
	dir := t.TempDir()

	if IsRepository(dir) {
		t.Fatal("empty temp dir reported as a repository")
	}
	if err := Init(dir, "main"); err != nil {
		t.Fatal(err)
	}
	if !IsRepository(dir) {
		t.Fatal("initialized dir not reported as a repository")
	}
	if _, err := RemoteURL(dir, "origin"); err == nil {
		t.Fatal("expected error for missing origin")
	}
	const url = "git@example.com:me/demo.git"
	if err := AddRemote(dir, "origin", url); err != nil {
		t.Fatal(err)
	}
	got, err := RemoteURL(dir, "origin")
	if err != nil || got != url {
		t.Fatalf("RemoteURL = %q, %v; want %q", got, err, url)
	}
}
