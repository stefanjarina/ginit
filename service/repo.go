package service

import (
	"errors"
	"fmt"
	"os"

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

// PreferSshUrl reports whether origin should use the SSH clone URL. It is the
// single place that reads the protocol setting; unset means SSH off Windows
// and HTTPS on Windows.
func PreferSshUrl(cfg *config.Config) (bool, error) {
	protocol, err := cfg.EffectiveProtocol()
	if err != nil {
		return false, gerrors.NewHint(err.Error(),
			fmt.Sprintf("fix it with: ginit config set %s %s|%s", config.ProtocolKey, config.ProtocolSSH, config.ProtocolHTTPS), nil)
	}
	return protocol == config.ProtocolSSH, nil
}

// SelectRemoteUrl picks the clone URL for origin. When the preferred URL is
// missing it falls back to the other one and warns. It never returns an
// empty URL without an error.
func SelectRemoteUrl(provider string, urls api.CloneURLs, preferSsh bool) (string, error) {
	preferred, other := urls.HTTPS, urls.SSH
	preferredName, otherName := config.ProtocolHTTPS, config.ProtocolSSH
	if preferSsh {
		preferred, other = urls.SSH, urls.HTTPS
		preferredName, otherName = config.ProtocolSSH, config.ProtocolHTTPS
	}
	if preferred != "" {
		return preferred, nil
	}
	if other != "" {
		console.Warning(fmt.Sprintf("%s returned no %s clone URL; using the %s URL %s", provider, preferredName, otherName, other))
		return other, nil
	}
	return "", gerrors.NewProvider(provider, "created repo has no clone URL", nil)
}

// RepoService orchestrates the end-to-end init flow.
type RepoService struct {
	Cfg           *config.Config
	CfgPath       string
	Accessibility bool

	GitignoreIo GitignoreClient

	// BeforeCreate, when set, runs after the provider prompts and right before
	// the repository is created on the host. An error aborts the create, so
	// nothing is left behind on the remote side.
	BeforeCreate func(pi *ProjectInfo) error

	// Force updates an existing origin that points elsewhere without asking.
	Force bool

	// ConfirmRemoteUpdate asks whether an origin currently pointing at
	// current should be changed to url.
	ConfirmRemoteUpdate func(current, url string) (bool, error)
}

// GitignoreClient is the subset of the gitignore.io client used by the init flow.
type GitignoreClient interface {
	List() ([]string, error)
	FetchConfig(names []string) (string, error)
}

func New(cfg *config.Config, cfgPath string, accessibility bool) *RepoService {
	return &RepoService{
		Cfg:           cfg,
		CfgPath:       cfgPath,
		Accessibility: accessibility,
		GitignoreIo:   gitignoreio.NewClient(),
		ConfirmRemoteUpdate: func(current, url string) (bool, error) {
			return prompts.AskToUpdateRemote("origin", current, url, accessibility)
		},
	}
}

// CreateRemoteRepo dispatches to the provider-specific handler.
func (r *RepoService) CreateRemoteRepo(provider string) (*ProjectInfo, error) {
	// Reject an invalid protocol before anything is created on the host.
	if _, err := PreferSshUrl(r.Cfg); err != nil {
		return nil, err
	}
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

// remoteUrl selects the origin URL from the clone URLs the provider returned.
func (r *RepoService) remoteUrl(provider string, urls api.CloneURLs) (string, error) {
	preferSsh, err := PreferSshUrl(r.Cfg)
	if err != nil {
		return "", err
	}
	return SelectRemoteUrl(provider, urls, preferSsh)
}

func (r *RepoService) beforeCreate(pi *ProjectInfo) error {
	if r.BeforeCreate == nil {
		return nil
	}
	return r.BeforeCreate(pi)
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

// PrepareLocalGit runs `git init` in dir and verifies that a commit identity
// is configured there. The identity is checked inside the new repository so
// that includeIf sections of the user's git config apply.
func (r *RepoService) PrepareLocalGit(dir string) error {
	return console.Run("Initializing local git", func() error {
		if err := gitops.Init(dir, r.Cfg.DefaultBranch); err != nil {
			return err
		}
		return gitops.CheckIdentity(dir)
	})
}

// CommitLocalGit stages everything in dir and creates the initial commit.
func (r *RepoService) CommitLocalGit(dir string) error {
	commit := func() error {
		if err := gitops.AddAll(dir); err != nil {
			return err
		}
		if err := gitops.Commit(dir, "initial commit"); err != nil {
			if errors.Is(err, gitops.ErrNothingToCommit) {
				return gerrors.NewHint("nothing to commit: the directory is empty and no .gitignore was generated",
					"add a file (e.g. README.md) or select gitignore templates", nil)
			}
			return err
		}
		return nil
	}

	if !gitops.SigningEnabled(dir) {
		return console.Run("Creating initial commit", commit)
	}

	// Signing may prompt for a passphrase, so git needs the terminal and no
	// spinner may run while it does.
	console.Info("Creating initial commit (commit signing is enabled, you may be asked for a passphrase)")
	if err := commit(); err != nil {
		return err
	}
	console.Success("✓ Creating initial commit")
	return nil
}

// CreateRemote makes origin point at remoteUrl. A missing origin is added and
// one that already has remoteUrl is left alone. An origin with a different
// URL is updated after confirmation, or without asking when Force is set.
func (r *RepoService) CreateRemote(remoteUrl string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	status, current, err := gitops.InspectRemote(cwd, "origin", remoteUrl)
	if err != nil {
		return err
	}

	switch status {
	case gitops.RemoteMatches:
		console.Info(fmt.Sprintf("Remote origin already points to %s", remoteUrl))
		return nil
	case gitops.RemoteDiffers:
		if !r.Force {
			// Prompt outside the spinner so the question is readable.
			ok, err := r.confirmRemoteUpdate(current, remoteUrl)
			if err != nil {
				return err
			}
			if !ok {
				return gerrors.NewHint(
					fmt.Sprintf("remote origin points to %s, not %s; it was left unchanged", current, remoteUrl),
					fmt.Sprintf("update it with: git remote set-url origin %s (or rerun with --force)", remoteUrl), nil)
			}
		}
		return console.Run("Updating remote origin", func() error {
			return gitops.SetRemoteURL(cwd, "origin", remoteUrl)
		})
	default:
		return console.Run("Configuring remote", func() error {
			return gitops.AddRemote(cwd, "origin", remoteUrl)
		})
	}
}

func (r *RepoService) confirmRemoteUpdate(current, url string) (bool, error) {
	if r.ConfirmRemoteUpdate == nil {
		return prompts.AskToUpdateRemote("origin", current, url, r.Accessibility)
	}
	return r.ConfirmRemoteUpdate(current, url)
}

// PushToRemote prompts the user, then runs `git push --set-upstream origin <branch>`
// for the currently checked-out branch.
func (r *RepoService) PushToRemote() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	// Resolve the branch before prompting so a detached HEAD fails early.
	branch, err := gitops.CurrentBranch(cwd)
	if err != nil {
		return err
	}
	push, err := prompts.AskToPushToRemote(r.Accessibility)
	if err != nil {
		return err
	}
	if !push {
		return nil
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

	availableTypes := r.fetchGitignoreList()

	client := api.NewGithubClient(token, baseUrl)
	if err := console.Run("Authenticating to GitHub", client.Connect); err != nil {
		return nil, gerrors.NewProvider("github", "authenticate", err)
	}

	pi, err := prompts.AskForProjectInfo("github", availableTypes, r.Accessibility)
	if err != nil {
		return nil, err
	}

	if err := r.beforeCreate(pi); err != nil {
		return nil, err
	}

	var urls api.CloneURLs
	if err := console.Run("Creating repo on GitHub", func() error {
		u, e := client.CreateRepository(pi.Name, pi.Description, pi.Visibility)
		urls = u
		return e
	}); err != nil {
		return nil, err
	}
	remoteUrl, err := r.remoteUrl("github", urls)
	if err != nil {
		return nil, err
	}
	pi.RemoteUrl = remoteUrl
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

	availableTypes := r.fetchGitignoreList()

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

	if err := r.beforeCreate(pi); err != nil {
		return nil, err
	}

	var urls api.CloneURLs
	if err := console.Run("Creating repo on Azure DevOps", func() error {
		u, e := client.CreateRepository(project, pi.Name)
		urls = u
		return e
	}); err != nil {
		return nil, err
	}
	remoteUrl, err := r.remoteUrl("azure", urls)
	if err != nil {
		return nil, err
	}
	pi.RemoteUrl = remoteUrl
	return pi, nil
}

func (r *RepoService) handleGitlab() (*ProjectInfo, error) {
	if err := r.ensureBaseUrlAndToken("gitlab"); err != nil {
		return nil, err
	}
	token := r.Cfg.GetValue("gitlab", "token")
	baseUrl := r.Cfg.GetValue("gitlab", "baseurl")

	availableTypes := r.fetchGitignoreList()

	// Project info (incl. visibility) first,
	// THEN authenticate + GetGroups.
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
		gs, e := client.GetGroups()
		groups = gs
		return e
	}); err != nil {
		return nil, gerrors.NewProvider("gitlab", "authenticate / list groups", err)
	}

	groupId, err := prompts.AskForGitlabGroup(groups, r.Accessibility)
	if err != nil {
		return nil, err
	}

	if err := r.beforeCreate(pi); err != nil {
		return nil, err
	}

	var urls api.CloneURLs
	if err := console.Run("Creating repo on GitLab", func() error {
		u, e := client.CreateRepository(groupId, pi.Name, pi.Description, pi.Visibility)
		urls = u
		return e
	}); err != nil {
		return nil, err
	}
	remoteUrl, err := r.remoteUrl("gitlab", urls)
	if err != nil {
		return nil, err
	}
	pi.RemoteUrl = remoteUrl
	return pi, nil
}

func (r *RepoService) handleBitbucket() (*ProjectInfo, error) {
	if err := r.ensureBaseUrlAndToken("bitbucket"); err != nil {
		return nil, err
	}
	// Cloud authenticates with a Bearer API token and ignores User. On Data
	// Center / Server an optional User switches to Basic auth; without it the
	// HTTP access token is sent as a Bearer token.
	user := r.Cfg.GetValue("bitbucket", "User")
	token := r.Cfg.GetValue("bitbucket", "token")
	baseUrl := r.Cfg.GetValue("bitbucket", "baseurl")

	availableTypes := r.fetchGitignoreList()

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

	if err := r.beforeCreate(pi); err != nil {
		return nil, err
	}

	var urls api.CloneURLs
	if err := console.Run("Creating repo on Bitbucket", func() error {
		u, e := client.CreateRepository(workspace, project, pi.Name, pi.Description, pi.Visibility)
		urls = u
		return e
	}); err != nil {
		return nil, err
	}
	remoteUrl, err := r.remoteUrl("bitbucket", urls)
	if err != nil {
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

	availableTypes := r.fetchGitignoreList()

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

	if err := r.beforeCreate(pi); err != nil {
		return nil, err
	}

	var urls api.CloneURLs
	if err := console.Run("Creating repo on "+provider, func() error {
		u, e := client.CreateRepository(owner, pi.Name, pi.Description, pi.Visibility)
		urls = u
		return e
	}); err != nil {
		return nil, err
	}
	remoteUrl, err := r.remoteUrl(provider, urls)
	if err != nil {
		return nil, err
	}
	pi.RemoteUrl = remoteUrl
	return pi, nil
}

// fetchGitignoreList returns the gitignore.io template list. A failure is not
// fatal for init: it warns and returns an empty list so the user can still
// create the repo, keep an existing .gitignore or ignore custom files.
func (r *RepoService) fetchGitignoreList() []string {
	var list []string
	err := console.RunOptional("Fetching gitignore.io template list", func() error {
		l, e := r.GitignoreIo.List()
		list = l
		return e
	})
	if err != nil {
		console.Warning("Continuing without gitignore.io templates")
		return nil
	}
	return list
}
