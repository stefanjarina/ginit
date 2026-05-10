package gitops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteGitignoreCompositionOrder(t *testing.T) {
	dir := t.TempDir()
	err := WriteGitignore(dir, []string{"node_modules", ".env"}, "# Generated\nbin/\n")
	if err != nil {
		t.Fatalf("WriteGitignore() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("ReadFile(.gitignore) error = %v", err)
	}
	want := "# CUSTOM SETTINGS\n\nnode_modules\n.env\n\n# GITIGNORE.IO:\n# Generated\nbin/\n"
	if string(data) != want {
		t.Fatalf(".gitignore = %q, want %q", string(data), want)
	}
}
