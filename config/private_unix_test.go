//go:build !windows

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func assertPrivate(t *testing.T, path string, dir bool) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", path, err)
	}
	want := os.FileMode(0o600)
	if dir {
		want = 0o700
	}
	if got := info.Mode().Perm(); got != want {
		t.Errorf("%s mode = %o, want %o", path, got, want)
	}
}

func TestSaveKeepsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "dotfiles", "ginit.yaml")
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(target, []byte("default_branch: main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	link := filepath.Join(dir, "ginit.yaml")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	if err := Save(link, &Config{DefaultBranch: "trunk"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("Lstat() error = %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("%s is no longer a symlink", link)
	}
	got, err := Load(target)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.DefaultBranch != "trunk" {
		t.Errorf("target DefaultBranch = %q, want trunk", got.DefaultBranch)
	}
	assertPrivate(t, target, false)
}
