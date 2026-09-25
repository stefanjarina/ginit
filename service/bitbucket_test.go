package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stefanjarina/ginit/config"
	"github.com/stefanjarina/ginit/console"
)

func TestHandleBitbucketNothingToPick(t *testing.T) {
	console.Accessible = true

	tests := []struct {
		name       string
		workspaces string
		projects   string
		wantErr    string
	}{
		{name: "no workspace", workspaces: `{"values":[]}`, wantErr: "no Bitbucket workspace"},
		{name: "no project", workspaces: `{"values":[{"name":"Team","slug":"team"}]}`, projects: `{"values":[]}`, wantErr: "no Bitbucket project"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			created := false
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					created = true
					http.Error(w, "unexpected create", http.StatusTeapot)
					return
				}
				switch r.URL.Path {
				case "/2.0/user":
					_, _ = w.Write([]byte(`{"username":"alice"}`))
				case "/2.0/workspaces":
					_, _ = w.Write([]byte(tt.workspaces))
				case "/2.0/workspaces/team/projects":
					_, _ = w.Write([]byte(tt.projects))
				default:
					http.NotFound(w, r)
				}
			}))
			defer srv.Close()

			svc := newTestService()
			svc.GitignoreIo = &fakeGitignore{}
			svc.Cfg.Providers = []config.Provider{{Name: "bitbucket", BaseUrl: srv.URL, Token: "secret"}}

			_, err := svc.handleBitbucket()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("handleBitbucket() error = %v, want it to contain %q", err, tt.wantErr)
			}
			if created {
				t.Error("a create request was sent after the empty prompt")
			}
		})
	}
}
