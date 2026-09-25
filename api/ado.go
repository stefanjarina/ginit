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

type adoProjectsResponse struct {
	Value []struct {
		Name string `json:"name"`
	} `json:"value"`
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

// GetProjects returns the names of every project the authenticated user can see.
func (ac *AdoClient) GetProjects() ([]string, error) {
	q := url.Values{"stateFilter": {"WellFormed"}}
	var resp adoProjectsResponse
	if err := ac.get("projects?"+q.Encode(), &resp); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(resp.Value))
	for _, p := range resp.Value {
		if p.Name != "" {
			out = append(out, p.Name)
		}
	}
	return out, nil
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
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
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
		return err
	}
	q := parsed.Query()
	q.Set("api-version", adoAPIVersion)
	parsed.RawQuery = q.Encode()

	req, err := http.NewRequest(method, parsed.String(), reader)
	if err != nil {
		return err
	}
	req.SetBasicAuth("", ac.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := ac.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("azure %s %s: %d %s", method, pathAndQuery, resp.StatusCode, strings.TrimSpace(string(rb)))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(rb, out)
}
