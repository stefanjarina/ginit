package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"testing"
)

// fakeGitlab serves /user as "sam" (display name "Sam"). Namespace search for
// the display name returns a decoy first; the by-username lookup returns
// userNamespace.
func fakeGitlab(t *testing.T, userNamespace string) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("PRIVATE-TOKEN"); got != "secret" {
			t.Fatalf("PRIVATE-TOKEN = %q, want secret", got)
		}
		switch r.URL.Path {
		case "/api/v4/user":
			_, _ = w.Write([]byte(`{"id":1,"username":"sam","name":"Sam"}`))
		case "/api/v4/namespaces":
			_, _ = w.Write([]byte(`[
				{"id":99,"name":"Samantha","path":"samantha","kind":"user"},
				{"id":10,"name":"Sam","path":"sam","kind":"user"}
			]`))
		case "/api/v4/namespaces/sam":
			_, _ = w.Write([]byte(userNamespace))
		case "/api/v4/groups":
			_, _ = w.Write([]byte(`[{"id":20,"name":"Team","path":"team"}]`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	})
}

func TestGitlabGetGroupsResolvesPersonalNamespaceByUsername(t *testing.T) {
	client := NewGitlabClient("secret", "http://example.test")
	client.http = &http.Client{Transport: handlerTransport(fakeGitlab(t, `{"id":10,"name":"Sam","path":"sam","kind":"user"}`))}
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	groups, err := client.GetGroups()
	if err != nil {
		t.Fatalf("GetGroups() error = %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("groups = %#v, want personal namespace + 1 group", groups)
	}
	if groups[0] != (GitlabGroup{ID: 10, Name: "Sam", Path: "sam"}) {
		t.Fatalf("personal namespace = %#v, want id 10 path sam", groups[0])
	}
	if groups[1].ID != 20 {
		t.Fatalf("groups[1] = %#v, want team group", groups[1])
	}
}

func TestGitlabGetGroupsSkipsNamespaceThatIsNotTheUsers(t *testing.T) {
	cases := map[string]string{
		"group kind":    `{"id":30,"name":"Sam","path":"sam","kind":"group"}`,
		"path mismatch": `{"id":99,"name":"Samantha","path":"samantha","kind":"user"}`,
	}
	for name, ns := range cases {
		t.Run(name, func(t *testing.T) {
			client := NewGitlabClient("secret", "http://example.test")
			client.http = &http.Client{Transport: handlerTransport(fakeGitlab(t, ns))}
			if err := client.Connect(); err != nil {
				t.Fatalf("Connect() error = %v", err)
			}

			groups, err := client.GetGroups()
			if err != nil {
				t.Fatalf("GetGroups() error = %v", err)
			}
			if len(groups) != 1 || groups[0].ID != 20 {
				t.Fatalf("groups = %#v, want only the team group", groups)
			}
		})
	}
}

// fakeGitlabGroups serves /user and the personal namespace, and answers
// /groups from pages (keyed by the "page" query value). A request carrying a
// visibility filter only sees groups of that visibility, as GitLab does.
func fakeGitlabGroups(t *testing.T, pages map[string][]gitlabTestGroup) (http.Handler, *[]url.Values) {
	t.Helper()
	var seen []url.Values
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/user":
			_, _ = w.Write([]byte(`{"id":1,"username":"sam","name":"Sam"}`))
		case "/api/v4/namespaces/sam":
			_, _ = w.Write([]byte(`{"id":10,"name":"Sam","path":"sam","kind":"user"}`))
		case "/api/v4/groups":
			q := r.URL.Query()
			seen = append(seen, q)
			page, ok := pages[q.Get("page")]
			if !ok {
				t.Fatalf("unexpected groups page %q", q.Get("page"))
			}
			var out []GitlabGroup
			for _, g := range page {
				if v := q.Get("visibility"); v != "" && v != g.visibility {
					continue
				}
				out = append(out, g.GitlabGroup)
			}
			if next, _ := strconv.Atoi(q.Get("page")); pages[strconv.Itoa(next+1)] != nil {
				w.Header().Set("X-Next-Page", strconv.Itoa(next+1))
			} else {
				w.Header().Set("X-Next-Page", "")
			}
			_ = json.NewEncoder(w).Encode(out)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}), &seen
}

type gitlabTestGroup struct {
	GitlabGroup
	visibility string
}

func connectedGitlab(t *testing.T, handler http.Handler) *GitlabClient {
	t.Helper()
	client := NewGitlabClient("secret", "http://example.test")
	client.http = &http.Client{Transport: handlerTransport(handler)}
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	return client
}

func TestGitlabGetGroupsReadsEveryPage(t *testing.T) {
	handler, seen := fakeGitlabGroups(t, map[string][]gitlabTestGroup{
		"1": {
			{GitlabGroup{ID: 20, Name: "Team", Path: "team"}, "private"},
			{GitlabGroup{ID: 21, Name: "Ops", Path: "ops"}, "internal"},
		},
		"2": {
			{GitlabGroup{ID: 22, Name: "Late", Path: "late"}, "private"},
		},
	})
	client := connectedGitlab(t, handler)

	groups, err := client.GetGroups()
	if err != nil {
		t.Fatalf("GetGroups() error = %v", err)
	}
	var ids []int
	for _, g := range groups {
		ids = append(ids, g.ID)
	}
	if want := []int{10, 20, 21, 22}; !slices.Equal(ids, want) {
		t.Fatalf("group ids = %v, want %v", ids, want)
	}
	if len(*seen) != 2 {
		t.Fatalf("groups requests = %d, want 2", len(*seen))
	}
	for _, q := range *seen {
		if q.Get("min_access_level") != "30" {
			t.Fatalf("min_access_level = %q, want 30 (Developer)", q.Get("min_access_level"))
		}
		if q.Get("per_page") != "100" {
			t.Fatalf("per_page = %q, want 100", q.Get("per_page"))
		}
	}
}

func TestGitlabGetGroupsIsNotFilteredByProjectVisibility(t *testing.T) {
	// The picker runs for a private project; a public group is still a legal
	// home for it and must be listed.
	handler, seen := fakeGitlabGroups(t, map[string][]gitlabTestGroup{
		"1": {
			{GitlabGroup{ID: 20, Name: "Team", Path: "team"}, "private"},
			{GitlabGroup{ID: 30, Name: "Open", Path: "open"}, "public"},
		},
	})
	client := connectedGitlab(t, handler)

	groups, err := client.GetGroups()
	if err != nil {
		t.Fatalf("GetGroups() error = %v", err)
	}
	if !slices.ContainsFunc(groups, func(g GitlabGroup) bool { return g.ID == 30 }) {
		t.Fatalf("groups = %#v, want the public group listed", groups)
	}
	for _, q := range *seen {
		if q.Has("visibility") {
			t.Fatalf("groups query has visibility=%q, want no visibility filter", q.Get("visibility"))
		}
	}
}

func TestGitlabGetGroupsMayReturnNoGroups(t *testing.T) {
	// A namespace lookup that is not the user's leaves no personal entry, and
	// GetGroups must not invent one.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/user":
			_, _ = w.Write([]byte(`{"id":1,"username":"sam","name":"Sam"}`))
		case "/api/v4/namespaces/sam":
			_, _ = w.Write([]byte(`{"id":30,"name":"Sam","path":"sam","kind":"group"}`))
		case "/api/v4/groups":
			_, _ = w.Write([]byte(`[]`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	})
	client := connectedGitlab(t, handler)

	groups, err := client.GetGroups()
	if err != nil {
		t.Fatalf("GetGroups() error = %v", err)
	}
	if len(groups) != 0 {
		t.Fatalf("groups = %#v, want none", groups)
	}
}
