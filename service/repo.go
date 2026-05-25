package service

import (
	"fmt"
	"os"
	"runtime"

	"github.com/stefanjarina/ginit/api"
	"github.com/stefanjarina/ginit/api/gitignoreio"
	"github.com/stefanjarina/ginit/config"
	"github.com/stefanjarina/ginit/console"
	gerrors "github.com/stefanjarina/ginit/errors"
	"github.com/stefanjarina/ginit/gitops"
	"github.com/stefanjarina/ginit/model"
	"github.com/stefanjarina/ginit/prompts"
)

// ProjectInfo is re-exported from the model package for callers' convenience.
type ProjectInfo = model.ProjectInfo

// PreferSshUrl returns true when the platform should default to SSH clone URLs.
func PreferSshUrl() bool { return runtime.GOOS != "windows" }

// RepoService orchestrates the end-to-end init flow.
type RepoService struct {
	Cfg           *config.Config
	CfgPath       string
	Accessibility bool

	GitignoreIo *gitignoreio.GitignoreIo
}

func New(cfg *config.Config, cfgPath string, accessibility bool) *RepoService {
	return &RepoService{
		Cfg:           cfg,
		CfgPath:       cfgPath,
		Accessibility: accessibility,
		GitignoreIo:   gitignoreio.NewClient(),
	}
}

// CreateRemoteRepo dispatches to the provider-specific handler.
func (r *RepoService) CreateRemoteRepo(provider string) (*ProjectInfo, error) {
	switch provider {
	case "github":
		return r.handleGithub()
	case "azure":
		return r.handleAzure()
	case "gitlab":
		return r.handleGitlab()
	case "bitbucket":
		return r.handleBitbucket()
	case "gitea", "forgejo":
		return r.handleGiteaCompatible(provider)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// CreateGitignoreFile fetches templates and writes .gitignore to cwd.
func (r *RepoService) CreateGitignoreFile(pi *ProjectInfo) error {
	return console.Run("Generating .gitignore", func() error {
		var ioContent string
		if len(pi.GitIgnoreConfigs) > 0 {
			content, err := r.GitignoreIo.FetchConfig(pi.GitIgnoreConfigs)
			if err != nil {
				return gerrors.New("fetch from gitignore.io", err)
			}
			ioContent = content
		}
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		return gitops.WriteGitignore(cwd, pi.ExcludedLocalFiles, ioContent)
	})
}

// InitializeLocalGit runs `git init`, stages everything and creates the initial commit.
func (r *RepoService) InitializeLocalGit() error {
	return console.Run("Initializing local git", func() error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		if err := gitops.Init(cwd, r.Cfg.DefaultBranch); err != nil {
			return err
		}
		if err := gitops.AddAll(cwd); err != nil {
			return err
		}
		return gitops.Commit(cwd, "initial commit")
	})
}

// CreateRemote registers the origin remote in the local repo.
func (r *RepoService) CreateRemote(remoteUrl string) error {
	return console.Run("Configuring remote", func() error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		return gitops.AddRemote(cwd, "origin", remoteUrl)
	})
}

// PushToRemote prompts the user, then runs `git push --set-upstream origin <branch>`.
func (r *RepoService) PushToRemote() error {
	push, err := prompts.AskToPushToRemote(r.Accessibility)
	if err != nil {
		return err
	}
	if !push {
		return nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	branch := r.Cfg.DefaultBranch
	if branch == "" {
		branch = "main"
	}
	if err := gitops.Push(cwd, "origin", branch); err != nil {
		return gerrors.New("push to remote", err)
	}
	console.Success("Pushed to remote")
	return nil
}

// ----- provider handlers -----

// ensureBaseUrlAndToken prompts for missing token / base URL and persists them.
// always checks baseurl then token. Defaults seeded by config.CreateDefault keep github/azure quiet.
func (r *RepoService) ensureBaseUrlAndToken(provider string) error {
	if r.Cfg.GetValue(provider, "baseurl") == "" {
		baseUrl, err := prompts.AskForBaseUrl(provider, r.Accessibility)
		if err != nil {
			return err
		}
		_ = r.Cfg.SetValue(provider, "baseurl", baseUrl)
		if err := config.Save(r.CfgPath, r.Cfg); err != nil {
			return err
		}
	}
	if r.Cfg.GetValue(provider, "token") == "" {
		token, err := prompts.AskForToken(provider, r.Accessibility)
		if err != nil {
			return err
		}
		_ = r.Cfg.SetValue(provider, "token", token)
		if err := config.Save(r.CfgPath, r.Cfg); err != nil {
			return err
		}
	}
	return nil
}

func (r *RepoService) handleGithub() (*ProjectInfo, error) {
	if err := r.ensureBaseUrlAndToken("github"); err != nil {
		return nil, err
	}
	token := r.Cfg.GetValue("github", "token")
	baseUrl := r.Cfg.GetValue("github", "baseurl")

	availableTypes, err := r.fetchGitignoreList()
	if err != nil {
		return nil, err
	}

	client := api.NewGithubClient(token, baseUrl)
	if err := console.Run("Authenticating to GitHub", client.Connect); err != nil {
		return nil, gerrors.NewProvider("github", "authenticate", err)
	}

	pi, err := prompts.AskForProjectInfo("github", availableTypes, r.Accessibility)
	if err != nil {
		return nil, err
	}

	var url string
	if err := console.Run("Creating repo on GitHub", func() error {
		u, e := client.CreateRepository(pi.Name, pi.Description, pi.Visibility)
		url = u
		return e
	}); err != nil {
		return nil, err
	}
	pi.RemoteUrl = url
	return pi, nil
}

func (r *RepoService) handleAzure() (*ProjectInfo, error) {
	// Azure also requires OrgName option.
	if r.Cfg.GetValue("azure", "OrgName") == "" {
		org, err := prompts.AskForAzureOrgName(r.Accessibility)
		if err != nil {
			return nil, err
		}
		_ = r.Cfg.SetValue("azure", "OrgName", org)
		if err := config.Save(r.CfgPath, r.Cfg); err != nil {
			return nil, err
		}
	}
	if err := r.ensureBaseUrlAndToken("azure"); err != nil {
		return nil, err
	}

	token := r.Cfg.GetValue("azure", "token")
	org := r.Cfg.GetValue("azure", "OrgName")
	baseUrl := r.Cfg.GetValue("azure", "baseurl")

	availableTypes, err := r.fetchGitignoreList()
	if err != nil {
		return nil, err
	}

	client := api.NewAdoClient(token, baseUrl, org)
	if err := console.Run("Authenticating to Azure DevOps", client.Connect); err != nil {
		return nil, gerrors.NewProvider("azure", "authenticate", err)
	}

	var projects []string
	if err := console.Run("Fetching Azure DevOps projects", func() error {
		ps, e := client.GetProjects()
		projects = ps
		return e
	}); err != nil {
		return nil, gerrors.NewProvider("azure", "list projects", err)
	}

	pi, err := prompts.AskForProjectInfo("azure", availableTypes, r.Accessibility)
	if err != nil {
		return nil, err
	}

	project, err := prompts.AskForAzureProject(projects, r.Accessibility)
	if err != nil {
		return nil, err
	}

	var url string
	if err := console.Run("Creating repo on Azure DevOps", func() error {
		u, e := client.CreateRepository(project, pi.Name)
		url = u
		return e
	}); err != nil {
		return nil, err
	}
	pi.RemoteUrl = url
	return pi, nil
}

func (r *RepoService) handleGitlab() (*ProjectInfo, error) {
	if err := r.ensureBaseUrlAndToken("gitlab"); err != nil {
		return nil, err
	}
	token := r.Cfg.GetValue("gitlab", "token")
	baseUrl := r.Cfg.GetValue("gitlab", "baseurl")

	availableTypes, err := r.fetchGitignoreList()
	if err != nil {
		return nil, err
	}

	// Project info (incl. visibility) first,
	// THEN authenticate + GetGroups, because the group query is filtered by visibility.
	pi, err := prompts.AskForProjectInfo("gitlab", availableTypes, r.Accessibility)
	if err != nil {
		return nil, err
	}

	client := api.NewGitlabClient(token, baseUrl)
	var groups []api.GitlabGroup
	if err := console.Run("Authenticating and fetching GitLab groups", func() error {
		if e := client.Connect(); e != nil {
			return e
		}
		gs, e := client.GetGroups(pi.Visibility)
		groups = gs
		return e
	}); err != nil {
		return nil, gerrors.NewProvider("gitlab", "authenticate / list groups", err)
	}

	groupId, err := prompts.AskForGitlabGroup(groups, r.Accessibility)
	if err != nil {
		return nil, err
	}

	var url string
	if err := console.Run("Creating repo on GitLab", func() error {
		u, e := client.CreateRepository(groupId, pi.Name, pi.Description, pi.Visibility)
		url = u
		return e
	}); err != nil {
		return nil, err
	}
	pi.RemoteUrl = url
	return pi, nil
}

func (r *RepoService) handleBitbucket() (*ProjectInfo, error) {
	if err := r.ensureBaseUrlAndToken("bitbucket"); err != nil {
		return nil, err
	}
	if r.Cfg.GetValue("bitbucket", "User") == "" {
		user, err := prompts.AskForBitbucketUser(r.Accessibility)
		if err != nil {
			return nil, err
		}
		_ = r.Cfg.SetValue("bitbucket", "User", user)
		if err := config.Save(r.CfgPath, r.Cfg); err != nil {
			return nil, err
		}
	}

	user := r.Cfg.GetValue("bitbucket", "User")
	token := r.Cfg.GetValue("bitbucket", "token")
	baseUrl := r.Cfg.GetValue("bitbucket", "baseurl")

	availableTypes, err := r.fetchGitignoreList()
	if err != nil {
		return nil, err
	}

	client := api.NewBitbucketClient(user, token, baseUrl)
	var workspaces []api.BitbucketWorkspace
	if err := console.Run("Authenticating and fetching Bitbucket workspaces", func() error {
		if e := client.Connect(); e != nil {
			return e
		}
		ws, e := client.GetWorkspaces()
		workspaces = ws
		return e
	}); err != nil {
		return nil, gerrors.NewProvider("bitbucket", "authenticate / list workspaces", err)
	}

	workspace, err := prompts.AskForBitbucketWorkspace(workspaces, r.Accessibility)
	if err != nil {
		return nil, err
	}

	var projects []api.BitbucketProject
	if err := console.Run("Fetching Bitbucket projects", func() error {
		ps, e := client.GetProjects(workspace)
		projects = ps
		return e
	}); err != nil {
		return nil, gerrors.NewProvider("bitbucket", "list projects", err)
	}
	project, err := prompts.AskForBitbucketProject(projects, r.Accessibility)
	if err != nil {
		return nil, err
	}

	pi, err := prompts.AskForProjectInfo("bitbucket", availableTypes, r.Accessibility)
	if err != nil {
		return nil, err
	}

	var remoteUrl string
	if err := console.Run("Creating repo on Bitbucket", func() error {
		u, e := client.CreateRepository(workspace, project, pi.Name, pi.Description, pi.Visibility)
		remoteUrl = u
		return e
	}); err != nil {
		return nil, err
	}
	pi.RemoteUrl = remoteUrl
	return pi, nil
}

func (r *RepoService) handleGiteaCompatible(provider string) (*ProjectInfo, error) {
	if err := r.ensureBaseUrlAndToken(provider); err != nil {
		return nil, err
	}
	token := r.Cfg.GetValue(provider, "token")
	baseUrl := r.Cfg.GetValue(provider, "baseurl")

	availableTypes, err := r.fetchGitignoreList()
	if err != nil {
		return nil, err
	}

	client := api.NewGiteaClient(provider, token, baseUrl)
	var owners []api.GiteaOwner
	if err := console.Run("Authenticating and fetching "+provider+" organizations", func() error {
		if e := client.Connect(); e != nil {
			return e
		}
		fetchedOwners, e := client.GetOwners()
		owners = fetchedOwners
		return e
	}); err != nil {
		return nil, gerrors.NewProvider(provider, "authenticate / list organizations", err)
	}

	owner, err := prompts.AskForGiteaOwner(provider, owners, r.Accessibility)
	if err != nil {
		return nil, err
	}

	pi, err := prompts.AskForProjectInfo(provider, availableTypes, r.Accessibility)
	if err != nil {
		return nil, err
	}

	var remoteUrl string
	if err := console.Run("Creating repo on "+provider, func() error {
		u, e := client.CreateRepository(owner, pi.Name, pi.Description, pi.Visibility)
		remoteUrl = u
		return e
	}); err != nil {
		return nil, err
	}
	pi.RemoteUrl = remoteUrl
	return pi, nil
}

func (r *RepoService) fetchGitignoreList() ([]string, error) {
	var list []string
	err := console.Run("Fetching gitignore.io template list", func() error {
		l, e := r.GitignoreIo.List()
		list = l
		return e
	})
	return list, err
}
