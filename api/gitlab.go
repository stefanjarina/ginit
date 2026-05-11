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

// GetGroups returns the user's personal namespace plus every group with the
// requested visibility. Mirrors GitlabService.GetGroups.
func (gc *GitlabClient) GetGroups(visibility string) ([]GitlabGroup, error) {
	if gc.user == nil {
		return nil, fmt.Errorf("not authenticated; call Connect first")
	}

	var nss []GitlabNamespace
	q := url.Values{"search": {gc.user.Name}}
	if err := gc.get("namespaces?"+q.Encode(), &nss); err != nil {
		return nil, err
	}

	var groups []GitlabGroup
	if visibility != "" {
		gq := url.Values{"visibility": {visibility}}
		if err := gc.get("groups?"+gq.Encode(), &groups); err != nil {
			return nil, err
		}
	} else {
		if err := gc.get("groups", &groups); err != nil {
			return nil, err
		}
	}

	out := make([]GitlabGroup, 0, len(groups)+1)
	if len(nss) > 0 {
		out = append(out, GitlabGroup{ID: nss[0].ID, Name: gc.user.Name, Path: nss[0].Path})
	}
	out = append(out, groups...)
	return out, nil
}

// CreateRepository creates the project under the given namespace and returns
// the SSH URL on Unix (HTTPS on Windows).
func (gc *GitlabClient) CreateRepository(namespaceId int, name, description, visibility string) (string, error) {
	body := map[string]any{
		"name":         name,
		"description":  description,
		"namespace_id": namespaceId,
		"visibility":   visibility,
	}
	var resp gitlabProjectResponse
	if err := gc.post("projects", body, &resp); err != nil {
		return "", gerrors.NewProvider("gitlab", "create repository", err)
	}
	if runtime.GOOS == "windows" {
		return resp.HttpUrlToRepo, nil
	}
	return resp.SshUrlToRepo, nil
}

// ----- HTTP helpers -----

func (gc *GitlabClient) get(pathAndQuery string, out any) error {
	return gc.do(http.MethodGet, pathAndQuery, nil, out)
}

func (gc *GitlabClient) post(path string, body, out any) error {
	return gc.do(http.MethodPost, path, body, out)
}

func (gc *GitlabClient) do(method, pathAndQuery string, body, out any) error {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequest(method, gc.baseUrl+pathAndQuery, reader)
	if err != nil {
		return err
	}
	req.Header.Set("PRIVATE-TOKEN", gc.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := gc.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("gitlab %s %s: %d %s", method, pathAndQuery, resp.StatusCode, strings.TrimSpace(string(rb)))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(rb, out)
}
