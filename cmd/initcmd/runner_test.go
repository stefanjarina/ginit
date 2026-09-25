package initcmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stefanjarina/ginit/config"
	"github.com/stefanjarina/ginit/service"
)

func TestPrepareBeforeCreateRefusesEmptyTree(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")

	dir := t.TempDir()
	t.Chdir(dir) // CreateGitignoreFile writes into the working directory.

	svc := service.New(&config.Config{DefaultBranch: "main"}, "", true)
	if err := svc.PrepareLocalGit(dir); err != nil {
		t.Fatalf("PrepareLocalGit() error = %v", err)
	}

	err := prepareBeforeCreate(svc, &service.ProjectInfo{Name: "demo"}, dir, true)
	if err == nil || !strings.Contains(err.Error(), "nothing to commit") {
		t.Fatalf("prepareBeforeCreate() error = %v, want nothing to commit error", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, ".gitignore")); !os.IsNotExist(statErr) {
		t.Fatalf(".gitignore should not exist, stat error = %v", statErr)
	}
}
