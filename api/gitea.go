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

type GiteaUser struct {
	Login    string `json:"login"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
}

func (u GiteaUser) ownerUsername() string {
	if u.Login != "" {
		return u.Login
	}
	return u.Username
}

func (u GiteaUser) ownerName() string {
	if u.FullName != "" {
		return u.FullName
	}
	return u.ownerUsername()
}

type GiteaOrganization struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

type GiteaOwner struct {
	Username string
	Name     string
	IsOrg    bool
}

type giteaRepoResponse struct {
	CloneURL string `json:"clone_url"`
	SSHURL   string `json:"ssh_url"`
}

type GiteaClient struct {
	provider string
	token    string
	baseUrl  string
	http     *http.Client
	user     *GiteaUser
}

func NewGiteaClient(provider, token, baseUrl string) *GiteaClient {
	return &GiteaClient{
		provider: provider,
		token:    token,
		baseUrl:  NormalizeGiteaAPIBase(baseUrl),
		http:     &http.Client{Timeout: 30 * time.Second},
	}
}

func NormalizeGiteaAPIBase(baseUrl string) string {
	if baseUrl == "" {
		baseUrl = "https://gitea.com"
	}
	baseUrl = strings.TrimRight(baseUrl, "/")
	if !strings.HasSuffix(baseUrl, "/api/v1") {
		baseUrl += "/api/v1"
	}
	return baseUrl + "/"
}

func (gc *GiteaClient) Connect() error {
	var u GiteaUser
	if err := gc.get("user", &u); err != nil {
		return err
	}
	gc.user = &u
	return nil
}

func (gc *GiteaClient) GetOwners() ([]GiteaOwner, error) {
	if gc.user == nil {
		return nil, fmt.Errorf("not authenticated; call Connect first")
	}
	owners := []GiteaOwner{{
		Username: gc.user.ownerUsername(),
		Name:     gc.user.ownerName(),
		IsOrg:    false,
	}}

	orgs, err := gc.listOrgs()
	if err != nil {
		return nil, err
	}
	for _, org := range orgs {
		username := org.Username
		if username == "" {
			username = org.Name
		}
		name := org.FullName
		if name == "" {
			name = org.Name
		}
		if name == "" {
			name = username
		}
		owners = append(owners, GiteaOwner{Username: username, Name: name, IsOrg: true})
	}
	return owners, nil
}

// giteaPageLimit is the page size requested from list endpoints. It matches
// Gitea's and Forgejo's default MAX_RESPONSE_ITEMS.
const giteaPageLimit = 50

// listOrgs follows user/orgs pagination. It stops once X-Total-Count items
// have been read, or, when the header is absent, on an empty or short page.
// The total is preferred because a server whose MAX_RESPONSE_ITEMS is below
// giteaPageLimit returns full pages that look short.
func (gc *GiteaClient) listOrgs() ([]GiteaOrganization, error) {
	var all []GiteaOrganization
	for page := 1; ; page++ {
		var orgs []GiteaOrganization
		path := fmt.Sprintf("user/orgs?page=%d&limit=%d", page, giteaPageLimit)
		header, err := gc.doWithHeader(http.MethodGet, path, nil, &orgs)
		if err != nil {
			return nil, err
		}
		all = append(all, orgs...)
		if len(orgs) == 0 {
			return all, nil
		}
		if total, err := strconv.Atoi(header.Get("X-Total-Count")); err == nil {
			if len(all) >= total {
				return all, nil
			}
			continue
		}
		if len(orgs) < giteaPageLimit {
			return all, nil
		}
	}
}

// CreateRepository creates a repository for owner. Gitea and Forgejo accept
// only a private flag when creating a repository (limited visibility exists
// for users and organizations, not repositories), so anything other than
// "public" is created private.
func (gc *GiteaClient) CreateRepository(owner GiteaOwner, name, description, visibility string) (CloneURLs, error) {
	body := map[string]any{
		"name":        name,
		"description": description,
		"private":     visibility != "public",
	}

	path := "user/repos"
	if owner.IsOrg {
		path = "orgs/" + url.PathEscape(owner.Username) + "/repos"
	}
	var resp giteaRepoResponse
	if err := gc.post(path, body, &resp); err != nil {
		return CloneURLs{}, gerrors.NewProvider(gc.provider, "create repository", err)
	}
	return CloneURLs{SSH: resp.SSHURL, HTTPS: resp.CloneURL}, nil
}

func (gc *GiteaClient) get(pathAndQuery string, out any) error {
	return gc.do(http.MethodGet, pathAndQuery, nil, out)
}

func (gc *GiteaClient) post(path string, body, out any) error {
	return gc.do(http.MethodPost, path, body, out)
}

func (gc *GiteaClient) do(method, pathAndQuery string, body, out any) error {
	_, err := gc.doWithHeader(method, pathAndQuery, body, out)
	return err
}

func (gc *GiteaClient) doWithHeader(method, pathAndQuery string, body, out any) (http.Header, error) {
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
	req.Header.Set("Authorization", "token "+gc.token)
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
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.Header, fmt.Errorf("%s %s %s: %d %s", gc.provider, method, pathAndQuery, resp.StatusCode, strings.TrimSpace(string(rb)))
	}
	if out == nil {
		return resp.Header, nil
	}
	return resp.Header, json.Unmarshal(rb, out)
}
