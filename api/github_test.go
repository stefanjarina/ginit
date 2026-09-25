package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"reflect"
	"strings"
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

	owner, err := client.UserOwner()
	if err != nil {
		t.Fatalf("UserOwner() error = %v", err)
	}
	remoteURLs, err := client.CreateRepository(owner, "demo", "description", "private")
	if err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}

	wantURLs := CloneURLs{SSH: "git@github.com:alice/demo.git", HTTPS: "https://github.com/alice/demo.git"}
	if remoteURLs != wantURLs {
		t.Fatalf("clone URLs = %#v, want %#v", remoteURLs, wantURLs)
	}
	want := map[string]any{"name": "demo", "description": "description", "private": true}
	if !reflect.DeepEqual(createBody, want) {
		t.Fatalf("create body = %#v, want %#v", createBody, want)
	}
}

func TestGithubCreateRepositoryBody(t *testing.T) {
	org := GithubOwner{Login: "acme", IsOrg: true}
	user := GithubOwner{Login: "alice"}
	tests := []struct {
		name       string
		owner      GithubOwner
		visibility string
		wantPath   string
		wantBody   map[string]any
	}{
		{name: "user private", owner: user, visibility: "private", wantPath: "/user/repos",
			wantBody: map[string]any{"name": "demo", "description": "d", "private": true}},
		{name: "user public", owner: user, visibility: "public", wantPath: "/user/repos",
			wantBody: map[string]any{"name": "demo", "description": "d", "private": false}},
		{name: "org private", owner: org, visibility: "private", wantPath: "/orgs/acme/repos",
			wantBody: map[string]any{"name": "demo", "description": "d", "visibility": "private"}},
		{name: "org public", owner: org, visibility: "public", wantPath: "/orgs/acme/repos",
			wantBody: map[string]any{"name": "demo", "description": "d", "visibility": "public"}},
		{name: "org internal", owner: org, visibility: "internal", wantPath: "/orgs/acme/repos",
			wantBody: map[string]any{"name": "demo", "description": "d", "visibility": "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			var gotBody map[string]any
			client := NewGithubClient("secret", "")
			client.http = &http.Client{Transport: handlerTransport(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("method = %s, want POST", r.Method)
				}
				gotPath = r.URL.Path
				if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
					t.Fatalf("decode create body: %v", err)
				}
				_, _ = w.Write([]byte(`{"clone_url":"https://github.com/x/demo.git","ssh_url":"git@github.com:x/demo.git"}`))
			}))}

			if _, err := client.CreateRepository(tt.owner, "demo", "d", tt.visibility); err != nil {
				t.Fatalf("CreateRepository() error = %v", err)
			}
			if gotPath != tt.wantPath {
				t.Errorf("path = %q, want %q", gotPath, tt.wantPath)
			}
			if !reflect.DeepEqual(gotBody, tt.wantBody) {
				t.Errorf("body = %#v, want %#v", gotBody, tt.wantBody)
			}
		})
	}
}

func TestGithubCreateRepositoryRejectsWithoutRequest(t *testing.T) {
	tests := []struct {
		name       string
		owner      GithubOwner
		visibility string
	}{
		{name: "internal user repo", owner: GithubOwner{Login: "alice"}, visibility: "internal"},
		{name: "unknown visibility", owner: GithubOwner{Login: "acme", IsOrg: true}, visibility: "limited"},
		{name: "empty visibility", owner: GithubOwner{Login: "alice"}, visibility: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewGithubClient("secret", "")
			client.http = &http.Client{Transport: handlerTransport(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
			}))}
			if _, err := client.CreateRepository(tt.owner, "demo", "", tt.visibility); err == nil {
				t.Fatal("CreateRepository() error = nil, want error")
			}
		})
	}
}

func TestGithubGetOwners(t *testing.T) {
	const orgs = `{
		"acme":    {"login":"acme","name":"Acme Corp","plan":{"name":"enterprise"}},
		"oss":     {"login":"oss","name":"","members_can_create_repositories":true},
		"locked":  {"login":"locked","members_can_create_repositories":false},
		"nopub":   {"login":"nopub","members_can_create_public_repositories":false,"members_can_create_internal_repositories":true},
		"owned":   {"login":"owned","members_can_create_repositories":false}
	}`
	var orgMap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(orgs), &orgMap); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		baseUrl string
		want    []GithubOwner
	}{
		{
			name: "github.com",
			want: []GithubOwner{
				{Login: "alice", Name: "Alice", Visibilities: []string{"private", "public"}},
				{Login: "acme", Name: "Acme Corp", IsOrg: true, Visibilities: []string{"private", "internal", "public"}},
				{Login: "oss", Name: "oss", IsOrg: true, Visibilities: []string{"private", "public"}},
				{Login: "nopub", Name: "nopub", IsOrg: true, Visibilities: []string{"private", "internal"}},
				{Login: "owned", Name: "owned", IsOrg: true, Visibilities: []string{"private", "public"}},
			},
		},
		{
			name:    "enterprise server",
			baseUrl: "https://github.example.com",
			want: []GithubOwner{
				{Login: "alice", Name: "Alice", Visibilities: []string{"private", "public"}},
				{Login: "acme", Name: "Acme Corp", IsOrg: true, Visibilities: []string{"private", "internal", "public"}},
				{Login: "oss", Name: "oss", IsOrg: true, Visibilities: []string{"private", "internal", "public"}},
				{Login: "nopub", Name: "nopub", IsOrg: true, Visibilities: []string{"private", "internal"}},
				{Login: "owned", Name: "owned", IsOrg: true, Visibilities: []string{"private", "internal", "public"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, err := url.Parse(NormalizeGithubAPIBase(tt.baseUrl))
			if err != nil {
				t.Fatal(err)
			}
			prefix := strings.TrimSuffix(base.Path, "/")
			client := NewGithubClient("secret", tt.baseUrl)
			client.http = &http.Client{Transport: handlerTransport(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := strings.TrimPrefix(r.URL.Path, prefix)
				switch {
				case path == "/user":
					_, _ = w.Write([]byte(`{"login":"alice","name":"Alice"}`))
				case path == "/user/memberships/orgs":
					if r.URL.Query().Get("state") != "active" {
						t.Errorf("state = %q, want active", r.URL.Query().Get("state"))
					}
					_, _ = w.Write([]byte(`[
						{"role":"member","organization":{"login":"acme"}},
						{"role":"member","organization":{"login":"oss"}},
						{"role":"member","organization":{"login":"locked"}},
						{"role":"member","organization":{"login":"nopub"}},
						{"role":"admin","organization":{"login":"owned"}}
					]`))
				case strings.HasPrefix(path, "/orgs/"):
					org, ok := orgMap[strings.TrimPrefix(path, "/orgs/")]
					if !ok {
						t.Fatalf("unexpected org path %s", path)
					}
					_, _ = w.Write(org)
				default:
					t.Fatalf("unexpected path %s", r.URL.Path)
				}
			}))}

			if _, _, err := client.GetOwners(); err == nil {
				t.Fatal("GetOwners() before Connect error = nil, want error")
			}
			if err := client.Connect(); err != nil {
				t.Fatalf("Connect() error = %v", err)
			}
			got, skipped, err := client.GetOwners()
			if err != nil {
				t.Fatalf("GetOwners() error = %v", err)
			}
			if len(skipped) != 0 {
				t.Errorf("GetOwners() skipped = %#v, want none", skipped)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetOwners() =\n%#v\nwant\n%#v", got, tt.want)
			}
		})
	}
}

func TestGithubGetOwnersSkipsUnreadableOrg(t *testing.T) {
	client := NewGithubClient("secret", "")
	client.http = &http.Client{Transport: handlerTransport(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			_, _ = w.Write([]byte(`{"login":"alice","name":"Alice"}`))
		case "/user/memberships/orgs":
			_, _ = w.Write([]byte(`[
				{"role":"member","organization":{"login":"first"}},
				{"role":"member","organization":{"login":"hidden"}},
				{"role":"member","organization":{"login":"third"}}
			]`))
		case "/orgs/first":
			_, _ = w.Write([]byte(`{"login":"first","name":"First"}`))
		case "/orgs/hidden":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"Resource not accessible by personal access token"}`))
		case "/orgs/third":
			_, _ = w.Write([]byte(`{"login":"third","name":"Third"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))}
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	got, skipped, err := client.GetOwners()
	if err != nil {
		t.Fatalf("GetOwners() error = %v", err)
	}
	want := []GithubOwner{
		{Login: "alice", Name: "Alice", Visibilities: []string{"private", "public"}},
		{Login: "first", Name: "First", IsOrg: true, Visibilities: []string{"private", "public"}},
		{Login: "third", Name: "Third", IsOrg: true, Visibilities: []string{"private", "public"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetOwners() =\n%#v\nwant\n%#v", got, want)
	}
	if len(skipped) != 1 || skipped[0].Login != "hidden" || skipped[0].Err == nil ||
		!strings.Contains(skipped[0].Err.Error(), "403") {
		t.Errorf("GetOwners() skipped = %#v, want hidden with a 403 error", skipped)
	}
}

func TestGithubGetOwnersMembershipError(t *testing.T) {
	client := NewGithubClient("secret", "")
	client.http = &http.Client{Transport: handlerTransport(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			_, _ = w.Write([]byte(`{"login":"alice","name":"Alice"}`))
		case "/user/memberships/orgs":
			w.WriteHeader(http.StatusForbidden)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))}
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if _, _, err := client.GetOwners(); err == nil {
		t.Fatal("GetOwners() on membership error = nil, want error")
	}
}
