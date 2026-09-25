//go:build !windows

package config

import (
	"os"
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
