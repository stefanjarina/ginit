package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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

	remoteURLs, err := client.CreateRepository(owners[1], "demo", "description", "private")
	if err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}
	wantURLs := CloneURLs{SSH: "git@gitea.example.com:team/demo.git", HTTPS: "https://gitea.example.com/team/demo.git"}
	if remoteURLs != wantURLs {
		t.Fatalf("clone URLs = %#v, want %#v", remoteURLs, wantURLs)
	}
	if createBody["name"] != "demo" || createBody["description"] != "description" || createBody["private"] != true {
		t.Fatalf("create body = %#v", createBody)
	}
}

func TestGiteaCreateRepositoryVisibility(t *testing.T) {
	tests := map[string]bool{
		"public":  false,
		"private": true,
		// Not offered for Gitea or Forgejo; must never produce a public repo.
		"limited":  true,
		"internal": true,
		"":         true,
	}
	for visibility, wantPrivate := range tests {
		var createBody map[string]any
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := json.NewDecoder(r.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			_, _ = w.Write([]byte(`{"clone_url":"https://gitea.example.com/alice/demo.git","ssh_url":"git@gitea.example.com:alice/demo.git"}`))
		})
		client := NewGiteaClient("forgejo", "secret", "http://example.test")
		client.http = &http.Client{Transport: handlerTransport(handler)}
		if _, err := client.CreateRepository(GiteaOwner{Username: "alice"}, "demo", "", visibility); err != nil {
			t.Fatalf("CreateRepository(%q) error = %v", visibility, err)
		}
		if createBody["private"] != wantPrivate {
			t.Errorf("CreateRepository(%q) private = %v, want %v", visibility, createBody["private"], wantPrivate)
		}
		if _, ok := createBody["internal"]; ok {
			t.Errorf("CreateRepository(%q) sent internal: %#v", visibility, createBody)
		}
	}
}

// giteaOrgsHandler serves orgs as user/orgs pages of at most pageCap items,
// setting X-Total-Count when withTotal is true.
func giteaOrgsHandler(t *testing.T, orgs []GiteaOrganization, pageCap int, withTotal bool, pages *[]string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/user":
			_, _ = w.Write([]byte(`{"login":"alice","full_name":"Alice"}`))
		case "/api/v1/user/orgs":
			q := r.URL.Query()
			*pages = append(*pages, q.Get("page"))
			page, _ := strconv.Atoi(q.Get("page"))
			limit, _ := strconv.Atoi(q.Get("limit"))
			if page < 1 || limit < 1 {
				t.Fatalf("user/orgs query = %q, want page and limit", r.URL.RawQuery)
			}
			limit = min(limit, pageCap)
			start := min((page-1)*limit, len(orgs))
			end := min(start+limit, len(orgs))
			if withTotal {
				w.Header().Set("X-Total-Count", strconv.Itoa(len(orgs)))
			}
			_ = json.NewEncoder(w).Encode(orgs[start:end])
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	})
}

func giteaTestOrgs(n int) []GiteaOrganization {
	orgs := make([]GiteaOrganization, n)
	for i := range orgs {
		orgs[i] = GiteaOrganization{Username: fmt.Sprintf("org%d", i), FullName: fmt.Sprintf("Org %d", i)}
	}
	return orgs
}

func TestGiteaGetOwnersPaginates(t *testing.T) {
	tests := map[string]struct {
		orgs      int
		pageCap   int
		withTotal bool
		wantPages []string
	}{
		"short last page":            {orgs: giteaPageLimit + 1, pageCap: giteaPageLimit, wantPages: []string{"1", "2"}},
		"server caps page size":      {orgs: 15, pageCap: 10, withTotal: true, wantPages: []string{"1", "2"}},
		"total on exact page":        {orgs: giteaPageLimit * 2, pageCap: giteaPageLimit, withTotal: true, wantPages: []string{"1", "2"}},
		"no total, exact final page": {orgs: giteaPageLimit * 2, pageCap: giteaPageLimit, wantPages: []string{"1", "2", "3"}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			orgs := giteaTestOrgs(tt.orgs)
			var pages []string
			client := NewGiteaClient("forgejo", "secret", "http://example.test")
			client.http = &http.Client{Transport: handlerTransport(giteaOrgsHandler(t, orgs, tt.pageCap, tt.withTotal, &pages))}
			if err := client.Connect(); err != nil {
				t.Fatalf("Connect() error = %v", err)
			}
			owners, err := client.GetOwners()
			if err != nil {
				t.Fatalf("GetOwners() error = %v", err)
			}
			if len(owners) != len(orgs)+1 {
				t.Fatalf("len(owners) = %d, want %d", len(owners), len(orgs)+1)
			}
			if owners[0].Username != "alice" || owners[0].IsOrg {
				t.Fatalf("owners[0] = %#v, want authenticated user", owners[0])
			}
			for i, org := range orgs {
				got := owners[i+1]
				if got.Username != org.Username || got.Name != org.FullName || !got.IsOrg {
					t.Fatalf("owners[%d] = %#v, want org %#v", i+1, got, org)
				}
			}
			if fmt.Sprint(pages) != fmt.Sprint(tt.wantPages) {
				t.Fatalf("requested pages = %v, want %v", pages, tt.wantPages)
			}
		})
	}
}
