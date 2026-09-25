package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestNormalizeGithubAPIBase(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: "https://api.github.com/"},
		{name: "github.com", in: "https://github.com", want: "https://api.github.com/"},
		{name: "api.github.com", in: "https://api.github.com", want: "https://api.github.com/"},
		{name: "enterprise", in: "https://github.example.com", want: "https://github.example.com/api/v3/"},
		{name: "enterprise with suffix", in: "https://github.example.com/api/v3", want: "https://github.example.com/api/v3/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeGithubAPIBase(tt.in); got != tt.want {
				t.Fatalf("NormalizeGithubAPIBase() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGithubClientConnectAndCreateRepository(t *testing.T) {
	var createBody map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "token secret" {
			t.Fatalf("Authorization = %q, want token secret", got)
		}

		switch r.URL.Path {
		case "/api/v3/user":
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			_, _ = w.Write([]byte(`{"login":"alice"}`))
		case "/api/v3/user/repos":
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			_, _ = w.Write([]byte(`{"clone_url":"https://github.com/alice/demo.git","ssh_url":"git@github.com:alice/demo.git"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	})

	client := NewGithubClient("secret", "https://github.example.com")
	client.http = &http.Client{Transport: handlerTransport(handler)}
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	remoteURLs, err := client.CreateRepository("demo", "description", "private")
	if err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}

	wantURLs := CloneURLs{SSH: "git@github.com:alice/demo.git", HTTPS: "https://github.com/alice/demo.git"}
	if remoteURLs != wantURLs {
		t.Fatalf("clone URLs = %#v, want %#v", remoteURLs, wantURLs)
	}
	if createBody["name"] != "demo" || createBody["description"] != "description" || createBody["private"] != true {
		t.Fatalf("create body = %#v", createBody)
	}
}
