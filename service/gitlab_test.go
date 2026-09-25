package service

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stefanjarina/ginit/config"
	"github.com/stefanjarina/ginit/console"
)

// captureStdout returns what fn writes to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	rd, wr, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = wr
	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(rd)
		done <- string(b)
	}()
	defer func() { os.Stdout = orig }()
	fn()
	_ = wr.Close()
	return <-done
}

func TestHandleGitlabAuthenticatesBeforeProjectPrompt(t *testing.T) {
	console.Accessible = true

	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		http.Error(w, `{"message":"401 Unauthorized"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	svc := newTestService()
	svc.GitignoreIo = &fakeGitignore{}
	svc.Cfg.Providers = []config.Provider{{Name: "gitlab", BaseUrl: srv.URL, Token: "bad"}}

	var err error
	out := captureStdout(t, func() { _, err = svc.handleGitlab() })

	if err == nil || !strings.Contains(err.Error(), "authenticate") {
		t.Fatalf("handleGitlab() error = %v, want an authenticate error", err)
	}
	if strings.Contains(out, "Repository Name") {
		t.Errorf("project prompt was shown before authentication:\n%s", out)
	}
	if len(paths) != 1 || paths[0] != "GET /api/v4/user" {
		t.Errorf("requests = %v, want only GET /api/v4/user", paths)
	}
}
