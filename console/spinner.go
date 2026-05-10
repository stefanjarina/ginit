package console

import (
	"fmt"
	"os"
)

// Run prints a "==> title..." line, runs fn, then prints success or failure.
// Plain text (no animation) — keeps the dependency surface minimal and
// behaves predictably in non-TTY environments.
func Run(title string, fn func() error) error {
	fmt.Fprintf(os.Stdout, "==> %s\n", title)
	if err := fn(); err != nil {
		Error(fmt.Sprintf("%s failed", title), err)
		return err
	}
	Success(fmt.Sprintf("    %s — done", title))
	return nil
}
