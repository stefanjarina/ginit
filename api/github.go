package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	gerrors "github.com/stefanjarina/ginit/errors"
)

type githubRepoResponse struct {
	CloneURL string `json:"clone_url"`
	SSHURL   string `json:"ssh_url"`
}

type GithubClient struct {
	token   string
	baseUrl string
	http    *http.Client
}

func NewGithubClient(token, baseUrl string) *GithubClient {
	return &GithubClient{
		token:   token,
		baseUrl: NormalizeGithubAPIBase(baseUrl),
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

func NormalizeGithubAPIBase(baseUrl string) string {
	if baseUrl == "" {
		baseUrl = "https://github.com"
	}
	baseUrl = strings.TrimRight(baseUrl, "/")
	if baseUrl == "https://github.com" || baseUrl == "http://github.com" ||
		baseUrl == "https://api.github.com" || baseUrl == "http://api.github.com" {
		return "https://api.github.com/"
	}
	if !strings.HasSuffix(baseUrl, "/api/v3") {
		baseUrl += "/api/v3"
	}
	return baseUrl + "/"
}

func (gc *GithubClient) Connect() error {
	return gc.get("user", nil)
}

// CreateRepository creates the repo on github.com under the authenticated user
// and returns its clone URLs.
func (gc *GithubClient) CreateRepository(name, description, visibility string) (CloneURLs, error) {
	body := map[string]any{
		"name":        name,
		"description": description,
		"private":     visibility == "private",
	}
	var resp githubRepoResponse
	if err := gc.post("user/repos", body, &resp); err != nil {
		return CloneURLs{}, gerrors.NewProvider("github", "create repository", err)
	}
	return CloneURLs{SSH: resp.SSHURL, HTTPS: resp.CloneURL}, nil
}

func (gc *GithubClient) get(pathAndQuery string, out any) error {
	return gc.do(http.MethodGet, pathAndQuery, nil, out)
}

func (gc *GithubClient) post(path string, body, out any) error {
	return gc.do(http.MethodPost, path, body, out)
}

func (gc *GithubClient) do(method, pathAndQuery string, body, out any) error {
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
	req.Header.Set("Authorization", "token "+gc.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := gc.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("github %s %s: %d %s", method, pathAndQuery, resp.StatusCode, strings.TrimSpace(string(rb)))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(rb, out)
}
