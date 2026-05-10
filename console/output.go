package console

import (
	stderrors "errors"
	"fmt"
	"os"

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

func Error(msg string, err error) {
	prefix := "Error"
	var ge *gerrors.GinitError
	if stderrors.As(err, &ge) && ge.Provider != "" {
		prefix = "[" + ge.Provider + "] error"
	}
	fmt.Fprintln(os.Stderr, render(errorStyle, fmt.Sprintf("%s: %s", prefix, msg)))
	if err != nil && Verbose {
		fmt.Fprintf(os.Stderr, "  cause: %+v\n", err)
	}
}
