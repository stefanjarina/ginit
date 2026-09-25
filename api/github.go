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

type githubRepoResponse struct {
	CloneURL string `json:"clone_url"`
	SSHURL   string `json:"ssh_url"`
}

type githubUser struct {
	Login string `json:"login"`
	Name  string `json:"name"`
}

type githubOrgMembership struct {
	Role         string `json:"role"`
	Organization struct {
		Login string `json:"login"`
	} `json:"organization"`
}

// githubOrg holds the organization settings that decide what a member may
// create. The member settings are pointers because GitHub only returns them
// to some callers; a missing value is not treated as a restriction.
type githubOrg struct {
	Login string `json:"login"`
	Name  string `json:"name"`
	Plan  *struct {
		Name string `json:"name"`
	} `json:"plan"`
	MembersCanCreateRepositories         *bool `json:"members_can_create_repositories"`
	MembersCanCreatePublicRepositories   *bool `json:"members_can_create_public_repositories"`
	MembersCanCreatePrivateRepositories  *bool `json:"members_can_create_private_repositories"`
	MembersCanCreateInternalRepositories *bool `json:"members_can_create_internal_repositories"`
}

// GithubOwner is an account a repository can be created under: the
// authenticated user or one of their organizations. Visibilities lists what
// can be created there, most restrictive first.
type GithubOwner struct {
	Login        string
	Name         string
	IsOrg        bool
	Visibilities []string
}

type GithubClient struct {
	token   string
	baseUrl string
	http    *http.Client
	user    *githubUser
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
	var u githubUser
	if err := gc.get("user", &u); err != nil {
		return err
	}
	gc.user = &u
	return nil
}

// isEnterpriseServer reports whether the client talks to GitHub Enterprise
// Server, where every organization supports internal repositories.
func (gc *GithubClient) isEnterpriseServer() bool {
	return gc.baseUrl != "https://api.github.com/"
}

// UserOwner returns the authenticated user as a repository owner. Users can
// only own private and public repositories.
func (gc *GithubClient) UserOwner() (GithubOwner, error) {
	if gc.user == nil {
		return GithubOwner{}, fmt.Errorf("not authenticated; call Connect first")
	}
	name := gc.user.Name
	if name == "" {
		name = gc.user.Login
	}
	return GithubOwner{Login: gc.user.Login, Name: name, Visibilities: []string{"private", "public"}}, nil
}

// GetOwners returns the authenticated user followed by the organizations the
// user is an active member of and may create repositories in.
func (gc *GithubClient) GetOwners() ([]GithubOwner, error) {
	user, err := gc.UserOwner()
	if err != nil {
		return nil, err
	}
	owners := []GithubOwner{user}

	const perPage = 100
	for page := 1; ; page++ {
		var memberships []githubOrgMembership
		query := "user/memberships/orgs?state=active&per_page=" + strconv.Itoa(perPage) + "&page=" + strconv.Itoa(page)
		if err := gc.get(query, &memberships); err != nil {
			return nil, err
		}
		for _, m := range memberships {
			if m.Organization.Login == "" {
				continue
			}
			var org githubOrg
			if err := gc.get("orgs/"+url.PathEscape(m.Organization.Login), &org); err != nil {
				return nil, err
			}
			if org.Login == "" {
				org.Login = m.Organization.Login
			}
			if owner, ok := gc.orgOwner(org, m.Role == "admin"); ok {
				owners = append(owners, owner)
			}
		}
		if len(memberships) < perPage {
			break
		}
	}
	return owners, nil
}

// orgOwner turns an organization into an owner. It returns false when the
// organization does not let the user create any repository.
func (gc *GithubClient) orgOwner(org githubOrg, admin bool) (GithubOwner, bool) {
	allowed := func(setting *bool) bool {
		return admin || setting == nil || *setting
	}
	if !allowed(org.MembersCanCreateRepositories) {
		return GithubOwner{}, false
	}

	// Internal repositories need an enterprise: GitHub Enterprise Server, or
	// an organization on the enterprise plan of GitHub Enterprise Cloud. The
	// internal member setting is only returned for such organizations.
	supportsInternal := gc.isEnterpriseServer() ||
		(org.Plan != nil && org.Plan.Name == "enterprise") ||
		org.MembersCanCreateInternalRepositories != nil

	var visibilities []string
	if allowed(org.MembersCanCreatePrivateRepositories) {
		visibilities = append(visibilities, "private")
	}
	if supportsInternal && allowed(org.MembersCanCreateInternalRepositories) {
		visibilities = append(visibilities, "internal")
	}
	if allowed(org.MembersCanCreatePublicRepositories) {
		visibilities = append(visibilities, "public")
	}
	if len(visibilities) == 0 {
		return GithubOwner{}, false
	}

	name := org.Name
	if name == "" {
		name = org.Login
	}
	return GithubOwner{Login: org.Login, Name: name, IsOrg: true, Visibilities: visibilities}, true
}

// CreateRepository creates the repo under owner and returns its clone URLs.
// Organization repositories are created with GitHub's visibility field, so
// "internal" is sent as is. User repositories only take the private flag and
// cannot be internal. An unknown visibility is rejected rather than created
// public.
//
// GitHub takes no default branch on create and refuses to set one to a
// branch that does not exist yet. An empty repository adopts the first
// branch pushed to it, so the init flow pushes the configured branch.
func (gc *GithubClient) CreateRepository(owner GithubOwner, name, description, visibility string) (CloneURLs, error) {
	switch visibility {
	case "private", "public", "internal":
	default:
		return CloneURLs{}, gerrors.NewProvider("github", "create repository", fmt.Errorf("unsupported visibility %q", visibility))
	}

	body := map[string]any{
		"name":        name,
		"description": description,
	}
	path := "user/repos"
	if owner.IsOrg {
		path = "orgs/" + url.PathEscape(owner.Login) + "/repos"
		body["visibility"] = visibility
	} else {
		if visibility == "internal" {
			return CloneURLs{}, gerrors.NewProvider("github", "create repository",
				fmt.Errorf("internal visibility is only available for organization repositories"))
		}
		body["private"] = visibility == "private"
	}

	var resp githubRepoResponse
	if err := gc.post(path, body, &resp); err != nil {
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
