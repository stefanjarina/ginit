package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestBuildAdoOrgURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		org     string
		want    string
	}{
		{name: "default cloud", baseURL: "https://dev.azure.com", org: "contoso", want: "https://dev.azure.com/contoso"},
		{name: "trailing slash", baseURL: "https://dev.azure.com/", org: "/contoso/", want: "https://dev.azure.com/contoso"},
		{name: "custom server", baseURL: "https://ado.example.com/tfs", org: "DefaultCollection", want: "https://ado.example.com/tfs/DefaultCollection"},
		{name: "scheme added", baseURL: "ado.example.com/tfs", org: "DefaultCollection", want: "https://ado.example.com/tfs/DefaultCollection"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildAdoOrgURL(tt.baseURL, tt.org); got != tt.want {
				t.Fatalf("BuildAdoOrgURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAdoClientConnectGetProjectsAndCreateRepository(t *testing.T) {
	var createBody map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, token, ok := r.BasicAuth()
		if !ok || user != "" || token != "secret" {
			t.Fatalf("BasicAuth() = %q/%q/%v, want empty/secret/true", user, token, ok)
		}
		if got := r.URL.Query().Get("api-version"); got != adoAPIVersion {
			t.Fatalf("api-version = %q, want %q", got, adoAPIVersion)
		}

		switch r.URL.Path {
		case "/contoso/_apis/projects":
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			if r.URL.Query().Get("$top") == "1" {
				_, _ = w.Write([]byte(`{"value":[{"name":"Alpha"}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"value":[{"name":"Alpha"},{"name":"Beta"}]}`))
		case "/contoso/_apis/projects/MyProject":
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			_, _ = w.Write([]byte(`{"id":"proj-id","name":"MyProject"}`))
		case "/contoso/MyProject/_apis/git/repositories":
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			_, _ = w.Write([]byte(`{"remoteUrl":"https://dev.azure.com/contoso/MyProject/_git/demo","sshUrl":"git@ssh.dev.azure.com:v3/contoso/MyProject/demo"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	})

	client := NewAdoClient("secret", "http://example.test", "contoso")
	client.http = &http.Client{Transport: handlerTransport(handler)}
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	projects, err := client.GetProjects()
	if err != nil {
		t.Fatalf("GetProjects() error = %v", err)
	}
	if len(projects) != 2 || projects[0] != "Alpha" || projects[1] != "Beta" {
		t.Fatalf("projects = %#v", projects)
	}

	remoteURLs, err := client.CreateRepository("MyProject", "demo")
	if err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}

	wantURLs := CloneURLs{SSH: "git@ssh.dev.azure.com:v3/contoso/MyProject/demo", HTTPS: "https://dev.azure.com/contoso/MyProject/_git/demo"}
	if remoteURLs != wantURLs {
		t.Fatalf("clone URLs = %#v, want %#v", remoteURLs, wantURLs)
	}
	project, ok := createBody["project"].(map[string]any)
	if createBody["name"] != "demo" || !ok || project["id"] != "proj-id" || project["name"] != "MyProject" {
		t.Fatalf("create body = %#v", createBody)
	}
}
