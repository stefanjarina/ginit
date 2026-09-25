package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stefanjarina/ginit/api"
	"github.com/stefanjarina/ginit/config"
	"github.com/stefanjarina/ginit/console"
)

// fakeProvider answers every POST with body, standing in for the create
// repository endpoint of a provider.
func fakeProvider(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func createGithub(t *testing.T, body string) api.CloneURLs {
	srv := fakeProvider(t, body)
	urls, err := api.NewGithubClient("secret", srv.URL).CreateRepository(api.GithubOwner{Login: "alice"}, "demo", "", "private")
	if err != nil {
		t.Fatalf("github CreateRepository() error = %v", err)
	}
	return urls
}

func createGitlab(t *testing.T, body string) api.CloneURLs {
	srv := fakeProvider(t, body)
	urls, err := api.NewGitlabClient("secret", srv.URL).CreateRepository(1, "demo", "", "private", "main")
	if err != nil {
		t.Fatalf("gitlab CreateRepository() error = %v", err)
	}
	return urls
}

func TestRemoteUrlFollowsProtocolSetting(t *testing.T) {
	console.Accessible = true

	providers := []struct {
		name       string
		create     func(t *testing.T, body string) api.CloneURLs
		both       string
		noSsh      string
		noHttps    string
		none       string
		ssh, https string
	}{
		{
			name:    "github",
			create:  createGithub,
			both:    `{"ssh_url":"git@github.com:alice/demo.git","clone_url":"https://github.com/alice/demo.git"}`,
			noSsh:   `{"ssh_url":"","clone_url":"https://github.com/alice/demo.git"}`,
			noHttps: `{"ssh_url":"git@github.com:alice/demo.git","clone_url":""}`,
			none:    `{}`,
			ssh:     "git@github.com:alice/demo.git",
			https:   "https://github.com/alice/demo.git",
		},
		{
			name:    "gitlab",
			create:  createGitlab,
			both:    `{"ssh_url_to_repo":"git@gitlab.com:alice/demo.git","http_url_to_repo":"https://gitlab.com/alice/demo.git"}`,
			noSsh:   `{"ssh_url_to_repo":"","http_url_to_repo":"https://gitlab.com/alice/demo.git"}`,
			noHttps: `{"ssh_url_to_repo":"git@gitlab.com:alice/demo.git"}`,
			none:    `{"ssh_url_to_repo":"","http_url_to_repo":""}`,
			ssh:     "git@gitlab.com:alice/demo.git",
			https:   "https://gitlab.com/alice/demo.git",
		},
	}

	for _, p := range providers {
		cases := []struct {
			name     string
			protocol string
			body     string
			want     string
		}{
			{name: "ssh", protocol: "ssh", body: p.both, want: p.ssh},
			{name: "https", protocol: "https", body: p.both, want: p.https},
			{name: "ssh missing falls back to https", protocol: "ssh", body: p.noSsh, want: p.https},
			{name: "https missing falls back to ssh", protocol: "https", body: p.noHttps, want: p.ssh},
			{name: "ssh with no urls", protocol: "ssh", body: p.none},
			{name: "https with no urls", protocol: "https", body: p.none},
		}
		for _, tc := range cases {
			t.Run(p.name+"/"+tc.name, func(t *testing.T) {
				r := &RepoService{Cfg: &config.Config{Protocol: tc.protocol}}
				got, err := r.remoteUrl(p.name, p.create(t, tc.body))
				if tc.want == "" {
					if err == nil || got != "" {
						t.Fatalf("remoteUrl() = %q, %v; want empty URL and an error", got, err)
					}
					if !strings.Contains(err.Error(), p.name) {
						t.Fatalf("error %q does not name the provider", err)
					}
					return
				}
				if err != nil {
					t.Fatalf("remoteUrl() error = %v", err)
				}
				if got != tc.want {
					t.Fatalf("remoteUrl() = %q, want %q", got, tc.want)
				}
			})
		}
	}
}

func TestPreferSshUrl(t *testing.T) {
	tests := []struct {
		protocol string
		want     bool
		wantErr  bool
	}{
		{protocol: "", want: config.DefaultProtocol() == config.ProtocolSSH},
		{protocol: "ssh", want: true},
		{protocol: "https", want: false},
		{protocol: "Https", want: false},
		{protocol: "git", wantErr: true},
	}
	for _, tt := range tests {
		got, err := PreferSshUrl(&config.Config{Protocol: tt.protocol})
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("PreferSshUrl(%q) = %v, %v; want %v, error %v", tt.protocol, got, err, tt.want, tt.wantErr)
		}
	}
}

func TestCreateRemoteRepoRejectsInvalidProtocol(t *testing.T) {
	r := &RepoService{Cfg: &config.Config{Protocol: "ftp"}}
	if _, err := r.CreateRemoteRepo("github"); err == nil || !strings.Contains(err.Error(), "protocol") {
		t.Fatalf("CreateRemoteRepo() error = %v, want protocol error", err)
	}
}
