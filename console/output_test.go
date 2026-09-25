package console

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	gerrors "github.com/stefanjarina/ginit/errors"
)

func TestFormatError(t *testing.T) {
	root := errors.New("exit status 128: Author identity unknown")
	wrapped := fmt.Errorf("git commit: %w", root)
	provider := gerrors.NewProvider("github", "create repo", errors.New("HTTP 422: name already exists"))

	tests := []struct {
		name     string
		msg      string
		err      error
		verbose  bool
		headline string
		causes   []string
	}{
		{
			name:     "nil error",
			msg:      "prompt",
			headline: "Error: prompt",
		},
		{
			name:     "non-verbose includes primary error text",
			msg:      "Initializing local git failed",
			err:      wrapped,
			headline: "Error: Initializing local git failed: git commit: exit status 128: Author identity unknown",
		},
		{
			name:     "verbose adds unwrap chain",
			msg:      "Initializing local git failed",
			err:      wrapped,
			verbose:  true,
			headline: "Error: Initializing local git failed: git commit: exit status 128: Author identity unknown",
			causes:   []string{"  caused by: exit status 128: Author identity unknown"},
		},
		{
			name:     "provider prefix is not repeated",
			msg:      "create remote repo",
			err:      provider,
			headline: "[github] error: create remote repo: create repo: HTTP 422: name already exists",
		},
		{
			name:     "label already in error text is not repeated",
			msg:      "push to remote",
			err:      gerrors.New("push to remote", errors.New("rejected")),
			verbose:  true,
			headline: "Error: push to remote: rejected",
			causes:   []string{"  caused by: rejected"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headline, causes := formatError(tt.msg, tt.err, tt.verbose)
			if headline != tt.headline {
				t.Errorf("headline = %q, want %q", headline, tt.headline)
			}
			if strings.Join(causes, "\n") != strings.Join(tt.causes, "\n") {
				t.Errorf("causes = %q, want %q", causes, tt.causes)
			}
		})
	}
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = orig }()
	fn()
	w.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}

func TestRunReportsFailureOnce(t *testing.T) {
	Accessible = true
	NoColor = true
	defer func() { Accessible, NoColor = false, false }()

	cause := errors.New("HTTP 422: name already exists")
	out := captureStderr(t, func() {
		err := Run("Creating repo on GitHub", func() error { return cause })
		if !errors.Is(err, cause) {
			t.Fatalf("Run error = %v, want wrapping %v", err, cause)
		}
		// Callers wrapping and reporting the same error must not print it again.
		Error("create remote repo", gerrors.NewProvider("github", "create", err))
	})

	want := "Error: Creating repo on GitHub failed: HTTP 422: name already exists\n"
	if out != want {
		t.Errorf("stderr = %q, want %q", out, want)
	}
}
