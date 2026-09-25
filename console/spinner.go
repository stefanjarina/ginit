package console

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

// Accessible mirrors the --accessibility persistent flag. When true (or when
// stdout is not a TTY), Run falls back to plain text output instead of an
// animated spinner.
var Accessible bool

var spinnerTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))

// Run shows an animated huh spinner while fn executes. On success it prints a
// "✓ <title>" line; on failure it surfaces the error via Error and returns it.
// In non-TTY or accessibility modes the spinner degrades to a single line of
// plain text so logs stay readable.
func Run(title string, fn func() error) error {
	return run(title, fn, func(msg string, err error) { Error(msg, err) })
}

// RunOptional behaves like Run but reports a failure as a warning, for steps
// the caller can recover from. The error is still returned.
func RunOptional(title string, fn func() error) error {
	return run(title, fn, func(msg string, err error) { Warning(fmt.Sprintf("%s: %v", msg, err)) })
}

func run(title string, fn func() error, report func(msg string, err error)) error {
	plain := Accessible || NoColor || os.Getenv("NO_COLOR") != "" || !isatty.IsTerminal(os.Stdout.Fd())

	var runErr error
	action := func(_ context.Context) error {
		runErr = fn()
		return nil
	}

	if plain {
		fmt.Fprintf(os.Stdout, "==> %s...\n", title)
		_ = action(context.Background())
	} else {
		err := spinner.New().
			Title(" " + title + "...").
			TitleStyle(spinnerTitleStyle).
			ActionWithErr(action).
			Run()
		if err != nil && runErr == nil {
			runErr = err
		}
	}

	if runErr != nil {
		report(fmt.Sprintf("%s failed", title), runErr)
		return runErr
	}
	Success(fmt.Sprintf("✓ %s", title))
	return nil
}
