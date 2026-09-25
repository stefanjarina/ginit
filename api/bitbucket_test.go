package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
)

func TestBitbucketClientEndpointsAuthAndCreateBody(t *testing.T) {
	var createBody map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("Authorization = %q, want %q", got, "Bearer secret")
		}

		switch r.URL.Path {
		case "/2.0/user":
			_, _ = w.Write([]byte(`{"display_name":"Alice","username":"alice"}`))
		case "/2.0/workspaces":
			if r.URL.Query().Get("role") != "member" {
				t.Fatalf("role query = %q, want member", r.URL.Query().Get("role"))
			}
			_, _ = w.Write([]byte(`{"values":[{"name":"Team","slug":"team"}]}`))
		case "/2.0/workspaces/team/projects":
			_, _ = w.Write([]byte(`{"values":[{"name":"Project","key":"PRJ"}]}`))
		case "/2.0/repositories/team/my-demo":
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			_, _ = w.Write([]byte(`{"links":{"clone":[{"name":"https","href":"https://bitbucket.org/team/demo.git"},{"name":"ssh","href":"git@bitbucket.org:team/demo.git"}]}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	})

	// A saved username must not turn Cloud requests into Basic auth.
	client := NewBitbucketClient("alice", "secret", "https://api.bitbucket.org/2.0")
	client.http = &http.Client{Transport: handlerTransport(handler)}
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	workspaces, err := client.GetWorkspaces()
	if err != nil {
		t.Fatalf("GetWorkspaces() error = %v", err)
	}
	if len(workspaces) != 1 || workspaces[0].Slug != "team" {
		t.Fatalf("workspaces = %#v", workspaces)
	}
	projects, err := client.GetProjects("team")
	if err != nil {
		t.Fatalf("GetProjects() error = %v", err)
	}
	if len(projects) != 1 || projects[0].Key != "PRJ" {
		t.Fatalf("projects = %#v", projects)
	}
	remoteURL, err := client.CreateRepository("team", "PRJ", "My Demo", "description", "private")
	if err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}

	wantURL := "git@bitbucket.org:team/demo.git"
	if runtime.GOOS == "windows" {
		wantURL = "https://bitbucket.org/team/demo.git"
	}
	if remoteURL != wantURL {
		t.Fatalf("remoteURL = %q, want %q", remoteURL, wantURL)
	}
	if createBody["scm"] != "git" || createBody["name"] != "My Demo" || createBody["description"] != "description" || createBody["is_private"] != true {
		t.Fatalf("create body = %#v", createBody)
	}
	project, ok := createBody["project"].(map[string]any)
	if !ok || project["key"] != "PRJ" {
		t.Fatalf("project body = %#v", createBody["project"])
	}
}

func TestBitbucketClientAuthorizationHeader(t *testing.T) {
	basic := "Basic " + base64.StdEncoding.EncodeToString([]byte("alice:secret"))
	tests := []struct {
		name    string
		user    string
		baseUrl string
		want    string
	}{
		{"cloud default", "", "", "Bearer secret"},
		{"cloud ignores user", "alice", "https://api.bitbucket.org/2.0", "Bearer secret"},
		{"server bearer", "", "https://bitbucket.example.com", "Bearer secret"},
		{"server basic", "alice", "https://bitbucket.example.com", basic},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			client := NewBitbucketClient(tt.user, "secret", tt.baseUrl)
			client.http = &http.Client{Transport: handlerTransport(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = r.Header.Get("Authorization")
				_, _ = w.Write([]byte(`{}`))
			}))}
			if err := client.Connect(); err != nil {
				t.Fatalf("Connect() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Authorization = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsBitbucketCloud(t *testing.T) {
	tests := map[string]bool{
		"":                              true,
		"https://api.bitbucket.org/2.0": true,
		"https://API.Bitbucket.org":     true,
		"https://bitbucket.example.com": false,
		"https://notbitbucket.org":      false,
		"http://example.test":           false,
	}
	for baseUrl, want := range tests {
		if got := IsBitbucketCloud(baseUrl); got != want {
			t.Errorf("IsBitbucketCloud(%q) = %v, want %v", baseUrl, got, want)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func handlerTransport(handler http.Handler) http.RoundTripper {
	return roundTripFunc(func(r *http.Request) (*http.Response, error) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, r)
		return rec.Result(), nil
	})
}

func TestBitbucketSlug(t *testing.T) {
	tests := map[string]string{
		"demo":            "demo",
		"MyRepo":          "myrepo",
		"My Repo":         "my-repo",
		"  My  Repo!  ":   "my-repo",
		"a_b.c-d":         "a_b.c-d",
		"Ünïcode -- Name": "n-code-name",
		"!!!":             "",
	}
	for name, want := range tests {
		if got := BitbucketSlug(name); got != want {
			t.Errorf("BitbucketSlug(%q) = %q, want %q", name, got, want)
		}
	}
}
