package console

import (
	stderrors "errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	gerrors "github.com/stefanjarina/ginit/errors"
)

var (
	Verbose bool
	NoColor bool
)

var (
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
)

func render(style lipgloss.Style, msg string) string {
	if NoColor || os.Getenv("NO_COLOR") != "" {
		return msg
	}
	return style.Render(msg)
}

func Info(msg string)    { fmt.Fprintln(os.Stdout, render(infoStyle, msg)) }
func Success(msg string) { fmt.Fprintln(os.Stdout, render(successStyle, msg)) }
func Warning(msg string) { fmt.Fprintln(os.Stderr, render(warnStyle, msg)) }

// Error prints a failure to stderr. The headline always carries the primary
// error text; with --verbose each wrapped cause from the unwrap chain follows on
// its own line. Errors already reported by Run are skipped so a single failure
// is never printed twice.
func Error(msg string, err error) {
	if IsReported(err) {
		return
	}
	headline, causes := formatError(msg, err, Verbose)
	fmt.Fprintln(os.Stderr, render(errorStyle, headline))
	for _, c := range causes {
		fmt.Fprintln(os.Stderr, c)
	}
}

// formatError builds the headline and, when verbose, the "caused by" lines.
func formatError(msg string, err error, verbose bool) (string, []string) {
	prefix := "Error"
	var ge *gerrors.GinitError
	if stderrors.As(err, &ge) && ge.Provider != "" {
		prefix = "[" + ge.Provider + "] error"
	}
	if err == nil {
		return fmt.Sprintf("%s: %s", prefix, msg), nil
	}

	text := err.Error()
	if ge != nil && ge.Provider != "" {
		text = strings.TrimPrefix(text, "["+ge.Provider+"] ")
	}
	headline := fmt.Sprintf("%s: %s: %s", prefix, msg, text)
	if msg == "" || strings.HasPrefix(text, msg) {
		headline = fmt.Sprintf("%s: %s", prefix, text)
	}

	if !verbose {
		return headline, nil
	}
	var causes []string
	for e := stderrors.Unwrap(err); e != nil; e = stderrors.Unwrap(e) {
		causes = append(causes, "  caused by: "+e.Error())
	}
	return headline, causes
}

// reportedError marks an error that has already been printed to the user.
type reportedError struct{ err error }

func (e *reportedError) Error() string { return e.err.Error() }
func (e *reportedError) Unwrap() error { return e.err }

// markReported wraps err so later calls to Error do not print it again.
func markReported(err error) error {
	if err == nil || IsReported(err) {
		return err
	}
	return &reportedError{err: err}
}

// IsReported reports whether err (or anything it wraps) was already printed.
func IsReported(err error) bool {
	var re *reportedError
	return stderrors.As(err, &re)
}
