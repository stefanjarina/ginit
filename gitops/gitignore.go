package gitops

import (
	"os"
	"path/filepath"
	"strings"

	gerrors "github.com/stefanjarina/ginit/errors"
)

// WriteGitignore composes a .gitignore from a custom-files section and the
// gitignore.io content, then writes it under dir. Mirrors
// novugit's RepoService.CreateGitIgnoreFile.
func WriteGitignore(dir string, customFiles []string, ioContent string) error {
	if len(customFiles) == 0 && strings.TrimSpace(ioContent) == "" {
		return nil
	}

	var b strings.Builder
	if len(customFiles) > 0 {
		b.WriteString("# CUSTOM SETTINGS\n\n")
		for _, f := range customFiles {
			if f == "" {
				continue
			}
			b.WriteString(f)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	if strings.TrimSpace(ioContent) != "" {
		b.WriteString("# GITIGNORE.IO:\n")
		b.WriteString(ioContent)
	}

	path := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return gerrors.New("write .gitignore", err)
	}
	return nil
}
