package gitops

import (
	"os/exec"
	"testing"
)

func TestIsRepositoryAndRemoteURL(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
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
