package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	gerrors "github.com/stefanjarina/ginit/errors"
)

const adoAPIVersion = "7.1"

// adoContinuationHeader carries the token for the next page of a list call.
const adoContinuationHeader = "x-ms-continuationtoken"

type adoProjectsResponse struct {
	Value []struct {
		Name string `json:"name"`
	} `json:"value"`
	ContinuationToken string `json:"continuationToken"`
}

type adoProject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type adoRepoResponse struct {
	RemoteURL string `json:"remoteUrl"`
	SshURL    string `json:"sshUrl"`
}

type AdoClient struct {
	orgUrl string
	token  string
	http   *http.Client
}

func NewAdoClient(token, baseUrl, orgName string) *AdoClient {
	return &AdoClient{
		orgUrl: BuildAdoOrgURL(baseUrl, orgName),
		token:  token,
		http:   &http.Client{Timeout: 30 * time.Second},
	}
}

func BuildAdoOrgURL(baseUrl, orgName string) string {
	if baseUrl == "" {
		baseUrl = "https://dev.azure.com"
	}
	baseUrl = strings.TrimRight(baseUrl, "/")
	if parsed, err := url.Parse(baseUrl); err == nil && parsed.Scheme == "" {
		baseUrl = "https://" + baseUrl
	}
	orgName = strings.Trim(orgName, "/")
	if orgName == "" {
		return baseUrl
	}
	return baseUrl + "/" + orgName
}

func (ac *AdoClient) Connect() error {
	q := url.Values{
		"stateFilter": {"WellFormed"},
		"$top":        {"1"},
	}
	return ac.get("projects?"+q.Encode(), nil)
}

// GetProjects returns the names of every project the authenticated user can see,
// following continuation tokens until the server stops sending one.
func (ac *AdoClient) GetProjects() ([]string, error) {
	out := []string{}
	seen := map[string]bool{}
	token := ""
	for {
		q := url.Values{"stateFilter": {"WellFormed"}}
		if token != "" {
			q.Set("continuationToken", token)
		}
		var resp adoProjectsResponse
		header, err := ac.send(http.MethodGet, "", "projects?"+q.Encode(), nil, &resp)
		if err != nil {
			return nil, err
		}
		for _, p := range resp.Value {
			if p.Name != "" {
				out = append(out, p.Name)
			}
		}

		token = header.Get(adoContinuationHeader)
		if token == "" {
			token = resp.ContinuationToken
		}
		if token == "" {
			return out, nil
		}
		if seen[token] {
			return nil, fmt.Errorf("azure GET projects: continuation token %q repeated", token)
		}
		seen[token] = true
	}
}

// CreateRepository creates a git repo under the given project and returns its
// clone URLs (sshUrl and the HTTPS remoteUrl).
func (ac *AdoClient) CreateRepository(projectName, repoName string) (CloneURLs, error) {
	var project adoProject
	if err := ac.get("projects/"+url.PathEscape(projectName), &project); err != nil {
		return CloneURLs{}, gerrors.NewProvider("azure", "fetch project '"+projectName+"'", err)
	}

	body := map[string]any{
		"name": repoName,
		"project": map[string]string{
			"id":   project.ID,
			"name": project.Name,
		},
	}
	var created adoRepoResponse
	if err := ac.postProject(projectName, "git/repositories", body, &created); err != nil {
		if strings.Contains(err.Error(), "TF400948") {
			return CloneURLs{}, gerrors.NewProvider("azure", "repository '"+repoName+"' already exists", err)
		}
		return CloneURLs{}, gerrors.NewProvider("azure", "create repository", err)
	}
	return CloneURLs{SSH: created.SshURL, HTTPS: created.RemoteURL}, nil
}

func (ac *AdoClient) get(pathAndQuery string, out any) error {
	return ac.do(http.MethodGet, "", pathAndQuery, nil, out)
}

func (ac *AdoClient) postProject(project, path string, body, out any) error {
	return ac.do(http.MethodPost, project, path, body, out)
}

func (ac *AdoClient) do(method, projectPrefix, pathAndQuery string, body, out any) error {
	_, err := ac.send(method, projectPrefix, pathAndQuery, body, out)
	return err
}

// send performs the request and returns the response headers alongside any error.
func (ac *AdoClient) send(method, projectPrefix, pathAndQuery string, body, out any) (http.Header, error) {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(buf)
	}

	endpoint := ac.orgUrl
	if projectPrefix != "" {
		endpoint += "/" + url.PathEscape(projectPrefix)
	}
	endpoint += "/_apis/" + pathAndQuery
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	q := parsed.Query()
	q.Set("api-version", adoAPIVersion)
	parsed.RawQuery = q.Encode()

	req, err := http.NewRequest(method, parsed.String(), reader)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth("", ac.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := ac.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.Header, fmt.Errorf("azure %s %s: %d %s", method, pathAndQuery, resp.StatusCode, strings.TrimSpace(string(rb)))
	}
	if out == nil {
		return resp.Header, nil
	}
	return resp.Header, json.Unmarshal(rb, out)
}
