package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	gerrors "github.com/stefanjarina/ginit/errors"
)

// GitlabUser is the minimal subset of /user response we care about.
type GitlabUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

type GitlabNamespace struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
	Kind string `json:"kind"`
}

type GitlabGroup struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}

type gitlabProjectResponse struct {
	HttpUrlToRepo string `json:"http_url_to_repo"`
	SshUrlToRepo  string `json:"ssh_url_to_repo"`
}

type GitlabClient struct {
	token   string
	baseUrl string // normalized to <base>/api/v4/
	http    *http.Client
	user    *GitlabUser
}

func NewGitlabClient(token, baseUrl string) *GitlabClient {
	if baseUrl == "" {
		baseUrl = "https://gitlab.com"
	}
	if !strings.HasSuffix(baseUrl, "/") {
		baseUrl += "/"
	}
	if !strings.HasSuffix(baseUrl, "api/v4/") {
		baseUrl += "api/v4/"
	}
	return &GitlabClient{
		token:   token,
		baseUrl: baseUrl,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (gc *GitlabClient) Connect() error {
	var u GitlabUser
	if err := gc.get("user", &u); err != nil {
		return err
	}
	gc.user = &u
	return nil
}

// gitlabGroupsPerPage is the page size requested from GET /groups; GitLab's
// maximum is 100 (the default of 20 would mean more round trips).
const gitlabGroupsPerPage = 100

// gitlabDeveloperAccess is GitLab's Developer access level, the lowest role a
// group can allow to create projects.
const gitlabDeveloperAccess = 30

// GetGroups returns the user's personal namespace plus every group the user
// can create projects in. Groups are not filtered by the new project's
// visibility: a private project may live in a public group.
func (gc *GitlabClient) GetGroups() ([]GitlabGroup, error) {
	if gc.user == nil {
		return nil, fmt.Errorf("not authenticated; call Connect first")
	}

	// Look the personal namespace up by username rather than searching by
	// display name: search is a partial match and can return someone else's
	// namespace first.
	var ns GitlabNamespace
	if err := gc.get("namespaces/"+url.PathEscape(gc.user.Username), &ns); err != nil {
		return nil, err
	}

	groups, err := gc.listGroups()
	if err != nil {
		return nil, err
	}

	out := make([]GitlabGroup, 0, len(groups)+1)
	if ns.Kind == "user" && ns.Path == gc.user.Username {
		out = append(out, GitlabGroup{ID: ns.ID, Name: gc.user.Name, Path: ns.Path})
	}
	out = append(out, groups...)
	return out, nil
}

// CreateRepository creates the project under the given namespace and returns
// its clone URLs. defaultBranch is sent as the project's default branch; on an
// empty project GitLab also adopts the first branch pushed to it.
func (gc *GitlabClient) CreateRepository(namespaceId int, name, description, visibility, defaultBranch string) (CloneURLs, error) {
	body := map[string]any{
		"name":           name,
		"description":    description,
		"namespace_id":   namespaceId,
		"visibility":     visibility,
		"default_branch": defaultBranch,
	}
	var resp gitlabProjectResponse
	if err := gc.post("projects", body, &resp); err != nil {
		return CloneURLs{}, gerrors.NewProvider("gitlab", "create repository", err)
	}
	return CloneURLs{SSH: resp.SshUrlToRepo, HTTPS: resp.HttpUrlToRepo}, nil
}

// listGroups follows GitLab's X-Next-Page header until every page of groups
// the user has at least Developer access to has been read.
func (gc *GitlabClient) listGroups() ([]GitlabGroup, error) {
	var all []GitlabGroup
	page := "1"
	for page != "" {
		q := url.Values{
			"min_access_level": {strconv.Itoa(gitlabDeveloperAccess)},
			"per_page":         {strconv.Itoa(gitlabGroupsPerPage)},
			"page":             {page},
		}
		var groups []GitlabGroup
		hdr, err := gc.do(http.MethodGet, "groups?"+q.Encode(), nil, &groups)
		if err != nil {
			return nil, err
		}
		all = append(all, groups...)
		page = strings.TrimSpace(hdr.Get("X-Next-Page"))
	}
	return all, nil
}

// ----- HTTP helpers -----

func (gc *GitlabClient) get(pathAndQuery string, out any) error {
	_, err := gc.do(http.MethodGet, pathAndQuery, nil, out)
	return err
}

func (gc *GitlabClient) post(path string, body, out any) error {
	_, err := gc.do(http.MethodPost, path, body, out)
	return err
}

// do sends the request, decodes a successful JSON response into out and
// returns the response headers (used for pagination).
func (gc *GitlabClient) do(method, pathAndQuery string, body, out any) (http.Header, error) {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequest(method, gc.baseUrl+pathAndQuery, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", gc.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := gc.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("gitlab %s %s: %d %s", method, pathAndQuery, resp.StatusCode, strings.TrimSpace(string(rb)))
	}
	if out == nil {
		return resp.Header, nil
	}
	return resp.Header, json.Unmarshal(rb, out)
}
