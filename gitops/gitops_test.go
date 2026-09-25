package gitops

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
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

// commitOn creates a repository in a temp dir whose HEAD is branch, with one commit.
func commitOn(t *testing.T, branch string) string {
	t.Helper()
	dir := t.TempDir()
	if err := Init(dir, branch); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := AddAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := Commit(dir, "initial commit"); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	return dir
}

func TestCurrentBranchUnborn(t *testing.T) {
	isolateGit(t)
	dir := t.TempDir()
	if err := Init(dir, "trunk"); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	got, err := CurrentBranch(dir)
	if err != nil {
		t.Fatalf("CurrentBranch() error = %v", err)
	}
	if got != "trunk" {
		t.Fatalf("CurrentBranch() = %q, want %q", got, "trunk")
	}
}

func TestCurrentBranchDetached(t *testing.T) {
	isolateGit(t)
	setIdentity(t)
	dir := commitOn(t, "main")
	if err := run(dir, "checkout", "--detach"); err != nil {
		t.Fatal(err)
	}
	_, err := CurrentBranch(dir)
	if err == nil || !strings.Contains(err.Error(), "detached") {
		t.Fatalf("CurrentBranch() error = %v, want detached HEAD error", err)
	}
}

func TestPushCurrentBranchNotMain(t *testing.T) {
	isolateGit(t)
	setIdentity(t)
	dir := commitOn(t, "main")
	// Switch to a different branch and drop main, so pushing "main" would fail.
	if err := run(dir, "checkout", "-b", "feature"); err != nil {
		t.Fatal(err)
	}
	if err := run(dir, "branch", "-D", "main"); err != nil {
		t.Fatal(err)
	}

	remote := t.TempDir()
	if err := run(remote, "init", "--bare"); err != nil {
		t.Fatal(err)
	}
	if err := AddRemote(dir, "origin", remote); err != nil {
		t.Fatal(err)
	}

	branch, err := CurrentBranch(dir)
	if err != nil {
		t.Fatalf("CurrentBranch() error = %v", err)
	}
	if branch != "feature" {
		t.Fatalf("CurrentBranch() = %q, want %q", branch, "feature")
	}
	if err := Push(dir, "origin", branch); err != nil {
		t.Fatalf("Push() error = %v", err)
	}
	if err := run(remote, "rev-parse", "--verify", "refs/heads/feature"); err != nil {
		t.Fatalf("remote has no feature branch: %v", err)
	}
	upstream, err := output(dir, "rev-parse", "--abbrev-ref", "feature@{upstream}")
	if err != nil || upstream != "origin/feature" {
		t.Fatalf("upstream = %q (%v), want origin/feature", upstream, err)
	}
}

func TestInspectRemote(t *testing.T) {
	isolateGit(t)
	dir := t.TempDir()
	if err := Init(dir, "main"); err != nil {
		t.Fatal(err)
	}
	const url = "git@example.com:me/demo.git"

	status, current, err := InspectRemote(dir, "origin", url)
	if err != nil || status != RemoteMissing || current != "" {
		t.Fatalf("missing origin: InspectRemote = %v, %q, %v; want RemoteMissing", status, current, err)
	}

	// A remote whose name only starts with "origin" must not count.
	if err := AddRemote(dir, "origin2", url); err != nil {
		t.Fatal(err)
	}
	if status, _, _ := InspectRemote(dir, "origin", url); status != RemoteMissing {
		t.Fatalf("origin2 only: status = %v, want RemoteMissing", status)
	}

	if err := AddRemote(dir, "origin", url); err != nil {
		t.Fatal(err)
	}
	status, current, err = InspectRemote(dir, "origin", url)
	if err != nil || status != RemoteMatches || current != url {
		t.Fatalf("same URL: InspectRemote = %v, %q, %v; want RemoteMatches", status, current, err)
	}

	const other = "https://example.com/me/demo.git"
	status, current, err = InspectRemote(dir, "origin", other)
	if err != nil || status != RemoteDiffers || current != url {
		t.Fatalf("different URL: InspectRemote = %v, %q, %v; want RemoteDiffers with %q", status, current, err, url)
	}

	if err := SetRemoteURL(dir, "origin", other); err != nil {
		t.Fatal(err)
	}
	if got, err := RemoteURL(dir, "origin"); err != nil || got != other {
		t.Fatalf("after SetRemoteURL: RemoteURL = %q, %v; want %q", got, err, other)
	}
}

func TestInspectRemoteOutsideRepository(t *testing.T) {
	isolateGit(t)
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(t.TempDir()))
	if _, _, err := InspectRemote(t.TempDir(), "origin", "x"); err == nil {
		t.Fatal("expected an error outside a git repository")
	}
}

// defaultPatterns mirrors config.DefaultSecretPatterns; gitops cannot import config.
var defaultPatterns = []string{
	".env", ".env.*", "!.env.example",
	"*.pem", "*.p12", "*.key",
	"id_rsa", "id_dsa", "id_ed25519",
}

func TestSensitivePaths(t *testing.T) {
	paths := []string{
		".env", ".env.local", ".env.example", "config/.env.production", "app/.env.example",
		"server.pem", "cert.p12", "tls/private.key", "id_rsa", ".ssh/id_dsa", "id_ed25519",
		"id_rsa.pub", "main.go", "README.md", "env", ".envrc", "keys.go",
	}
	want := []string{
		".env", ".env.local", "config/.env.production",
		"server.pem", "cert.p12", "tls/private.key", "id_rsa", ".ssh/id_dsa", "id_ed25519",
	}
	got, err := SensitivePaths(paths, defaultPatterns)
	if err != nil {
		t.Fatalf("SensitivePaths() error = %v", err)
	}
	if !slices.Equal(got, want) {
		t.Errorf("SensitivePaths() = %q, want %q", got, want)
	}
}

func TestSensitivePathsCustomPatterns(t *testing.T) {
	paths := []string{"secrets/prod.yml", "config/secrets/dev.yml", "creds.json", "creds.json.example"}
	got, err := SensitivePaths(paths, []string{"secrets/*", "creds.*", "!*.example"})
	if err != nil {
		t.Fatalf("SensitivePaths() error = %v", err)
	}
	if want := []string{"secrets/prod.yml", "creds.json"}; !slices.Equal(got, want) {
		t.Errorf("SensitivePaths() = %q, want %q", got, want)
	}
}

func TestSensitivePathsInvalidPattern(t *testing.T) {
	if _, err := SensitivePaths([]string{"a"}, []string{"[a-"}); err == nil {
		t.Fatal("SensitivePaths() error = nil, want invalid pattern error")
	}
}

func TestStagedFiles(t *testing.T) {
	isolateGit(t)
	dir := t.TempDir()
	if err := Init(dir, "main"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{".env", "sub dir/a.txt"} {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := AddAll(dir); err != nil {
		t.Fatal(err)
	}
	got, err := StagedFiles(dir)
	if err != nil {
		t.Fatalf("StagedFiles() error = %v", err)
	}
	if want := []string{".env", "sub dir/a.txt"}; !slices.Equal(got, want) {
		t.Errorf("StagedFiles() = %q, want %q", got, want)
	}
}
