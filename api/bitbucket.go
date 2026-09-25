package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	gerrors "github.com/stefanjarina/ginit/errors"
)

type BitbucketUser struct {
	DisplayName string `json:"display_name"`
	Username    string `json:"username"`
}

type BitbucketWorkspace struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	UUID string `json:"uuid"`
}

type BitbucketProject struct {
	Name string `json:"name"`
	Key  string `json:"key"`
	UUID string `json:"uuid"`
}

type bitbucketPage[T any] struct {
	Values []T    `json:"values"`
	Next   string `json:"next"`
}

type bitbucketRepoResponse struct {
	Links struct {
		Clone []struct {
			Name string `json:"name"`
			Href string `json:"href"`
		} `json:"clone"`
	} `json:"links"`
}

type BitbucketClient struct {
	user    string
	token   string
	baseUrl string
	http    *http.Client
}

// NewBitbucketClient builds a client for baseUrl. Bitbucket Cloud
// authenticates with a scoped API token sent as a Bearer token, and user is
// ignored there. On any other host (Data Center / Server) the token is sent
// as a Bearer token too, which is how HTTP access tokens work, unless user is
// set, in which case HTTP Basic auth with user and token is used.
func NewBitbucketClient(user, token, baseUrl string) *BitbucketClient {
	if IsBitbucketCloud(baseUrl) {
		user = ""
	}
	return &BitbucketClient{
		user:    user,
		token:   token,
		baseUrl: NormalizeBitbucketAPIBase(baseUrl),
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

func NormalizeBitbucketAPIBase(baseUrl string) string {
	if baseUrl == "" {
		baseUrl = "https://api.bitbucket.org/2.0"
	}
	baseUrl = strings.TrimRight(baseUrl, "/")
	if !strings.HasSuffix(baseUrl, "/2.0") {
		baseUrl += "/2.0"
	}
	return baseUrl + "/"
}

// IsBitbucketCloud reports whether baseUrl points at Bitbucket Cloud. An
// empty baseUrl means the Cloud default.
func IsBitbucketCloud(baseUrl string) bool {
	if strings.TrimSpace(baseUrl) == "" {
		return true
	}
	u, err := url.Parse(strings.TrimSpace(baseUrl))
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "bitbucket.org" || strings.HasSuffix(host, ".bitbucket.org")
}

func (bc *BitbucketClient) setAuth(req *http.Request) {
	if bc.user != "" {
		req.SetBasicAuth(bc.user, bc.token)
		return
	}
	req.Header.Set("Authorization", "Bearer "+bc.token)
}

func (bc *BitbucketClient) Connect() error {
	var u BitbucketUser
	return bc.get("user", &u)
}

func (bc *BitbucketClient) GetWorkspaces() ([]BitbucketWorkspace, error) {
	var workspaces []BitbucketWorkspace
	if err := getBitbucketPaged(bc, "workspaces?role=member", &workspaces); err != nil {
		return nil, err
	}
	return workspaces, nil
}

func (bc *BitbucketClient) GetProjects(workspace string) ([]BitbucketProject, error) {
	var projects []BitbucketProject
	path := "workspaces/" + url.PathEscape(workspace) + "/projects"
	if err := getBitbucketPaged(bc, path, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func (bc *BitbucketClient) CreateRepository(workspace, projectKey, name, description, visibility string) (string, error) {
	body := map[string]any{
		"scm":         "git",
		"name":        name,
		"is_private":  visibility != "public",
		"description": description,
	}
	if projectKey != "" {
		body["project"] = map[string]string{"key": projectKey}
	}
	path := "repositories/" + url.PathEscape(workspace) + "/" + url.PathEscape(BitbucketSlug(name))
	var resp bitbucketRepoResponse
	if err := bc.post(path, body, &resp); err != nil {
		return "", gerrors.NewProvider("bitbucket", "create repository", err)
	}
	return pickCloneURL(resp.Links.Clone)
}

// BitbucketSlug turns a repository name into the slug Bitbucket expects in
// repositories/{workspace}/{repo_slug}: lowercase ASCII letters, digits,
// '.', '_' and '-'. Any other run of characters, hyphens included,
// becomes a single '-'.
func BitbucketSlug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_' {
			b.WriteRune(r)
			dash = false
			continue
		}
		if !dash {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func getBitbucketPaged[T any](bc *BitbucketClient, path string, out *[]T) error {
	next := path
	for next != "" {
		var page bitbucketPage[T]
		if err := bc.get(next, &page); err != nil {
			return err
		}
		*out = append(*out, page.Values...)
		next = page.Next
	}
	return nil
}

func (bc *BitbucketClient) get(pathAndQuery string, out any) error {
	return bc.do(http.MethodGet, pathAndQuery, nil, out)
}

func (bc *BitbucketClient) post(path string, body, out any) error {
	return bc.do(http.MethodPost, path, body, out)
}

func (bc *BitbucketClient) do(method, pathAndQuery string, body, out any) error {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(buf)
	}
	endpoint := pathAndQuery
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = bc.baseUrl + pathAndQuery
	}
	req, err := http.NewRequest(method, endpoint, reader)
	if err != nil {
		return err
	}
	bc.setAuth(req)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := bc.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("bitbucket %s %s: %d %s", method, pathAndQuery, resp.StatusCode, strings.TrimSpace(string(rb)))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(rb, out)
}

func pickCloneURL(clones []struct {
	Name string `json:"name"`
	Href string `json:"href"`
}) (string, error) {
	preferred := "ssh"
	if runtime.GOOS == "windows" {
		preferred = "https"
	}
	var fallback string
	for _, clone := range clones {
		if clone.Name == preferred {
			return clone.Href, nil
		}
		if fallback == "" {
			fallback = clone.Href
		}
	}
	if fallback != "" {
		return fallback, nil
	}
	return "", gerrors.NewProvider("bitbucket", "created repo has no URL", nil)
}
