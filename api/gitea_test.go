package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestGiteaClientOwnersAndCreateBody(t *testing.T) {
	var createBody map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "token secret" {
			t.Fatalf("Authorization = %q, want token secret", got)
		}

		switch r.URL.Path {
		case "/api/v1/user":
			_, _ = w.Write([]byte(`{"login":"alice","full_name":"Alice"}`))
		case "/api/v1/user/orgs":
			_, _ = w.Write([]byte(`[{"username":"team","full_name":"Team"}]`))
		case "/api/v1/orgs/team/repos":
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			_, _ = w.Write([]byte(`{"clone_url":"https://gitea.example.com/team/demo.git","ssh_url":"git@gitea.example.com:team/demo.git"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	})

	client := NewGiteaClient("gitea", "secret", "http://example.test")
	client.http = &http.Client{Transport: handlerTransport(handler)}
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	owners, err := client.GetOwners()
	if err != nil {
		t.Fatalf("GetOwners() error = %v", err)
	}
	if len(owners) != 2 || owners[0].Username != "alice" || owners[0].IsOrg || owners[1].Username != "team" || !owners[1].IsOrg {
		t.Fatalf("owners = %#v", owners)
	}

	remoteURLs, err := client.CreateRepository(owners[1], "demo", "description", "limited")
	if err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}
	wantURLs := CloneURLs{SSH: "git@gitea.example.com:team/demo.git", HTTPS: "https://gitea.example.com/team/demo.git"}
	if remoteURLs != wantURLs {
		t.Fatalf("clone URLs = %#v, want %#v", remoteURLs, wantURLs)
	}
	if createBody["name"] != "demo" || createBody["description"] != "description" || createBody["private"] != false || createBody["internal"] != true {
		t.Fatalf("create body = %#v", createBody)
	}
}
