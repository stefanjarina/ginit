package api

import (
	"net/http"
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

	groups, err := client.GetGroups("")
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

			groups, err := client.GetGroups("")
			if err != nil {
				t.Fatalf("GetGroups() error = %v", err)
			}
			if len(groups) != 1 || groups[0].ID != 20 {
				t.Fatalf("groups = %#v, want only the team group", groups)
			}
		})
	}
}
