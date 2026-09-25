package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	gerrors "github.com/stefanjarina/ginit/errors"
	"gopkg.in/yaml.v2"
)

type Config struct {
	DefaultBranch string `yaml:"defaultbranch"`
	// Protocol selects the clone URL used for origin: "ssh" or "https".
	// Empty means the platform default, see DefaultProtocol.
	Protocol  string     `yaml:"protocol,omitempty"`
	Providers []Provider `yaml:"providers"`
}

// providerDefaults seeds known providers with sensible BaseUrls.
var providerDefaults = map[string]string{
	"azure":     "https://dev.azure.com",
	"bitbucket": "https://api.bitbucket.org/2.0",
	"forgejo":   "https://codeberg.org",
	"gitea":     "https://gitea.com",
	"github":    "https://github.com",
	"gitlab":    "https://gitlab.com",
}

// Module-level state set once by Load/CreateDefault. Commands read from here
// instead of re-loading. Keep simple — single-binary, single-config CLI.
var (
	Current     *Config
	CurrentPath string
)

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, gerrors.New(fmt.Sprintf("read config %s", path), err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, gerrors.New(fmt.Sprintf("parse config %s", path), err)
	}
	for i := range c.Providers {
		if c.Providers[i].Options == nil {
			c.Providers[i].Options = map[string]string{}
		}
	}
	return &c, nil
}

func Save(path string, c *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return gerrors.New("create config dir", err)
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return gerrors.New("marshal config", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return gerrors.New(fmt.Sprintf("write config %s", path), err)
	}
	return nil
}

func CreateDefault(path string, supported []string) error {
	providers := make([]Provider, 0, len(supported))
	for _, name := range supported {
		providers = append(providers, Provider{
			Name:    name,
			BaseUrl: providerDefaults[name],
			Options: map[string]string{},
		})
	}
	c := &Config{DefaultBranch: "main", Providers: providers}
	return Save(path, c)
}

func (c *Config) GetProvider(name string) *Provider {
	for i := range c.Providers {
		if strings.EqualFold(c.Providers[i].Name, name) {
			return &c.Providers[i]
		}
	}
	return nil
}

// special-cases "token" and "baseurl"; everything else lives in Options.
func (c *Config) GetValue(provider, key string) string {
	p := c.GetProvider(provider)
	if p == nil {
		return ""
	}
	switch strings.ToLower(key) {
	case "token":
		return p.Token
	case "baseurl":
		return p.BaseUrl
	default:
		if p.Options == nil {
			return ""
		}
		return p.Options[key]
	}
}

func (c *Config) SetValue(provider, key, value string) error {
	p := c.GetProvider(provider)
	if p == nil {
		return fmt.Errorf("unknown provider: %s", provider)
	}
	switch strings.ToLower(key) {
	case "token":
		p.Token = value
	case "baseurl":
		p.BaseUrl = value
	default:
		if p.Options == nil {
			p.Options = map[string]string{}
		}
		p.Options[key] = value
	}
	return nil
}

func (c *Config) RemoveValue(provider, key string) error {
	p := c.GetProvider(provider)
	if p == nil {
		return fmt.Errorf("unknown provider: %s", provider)
	}
	switch strings.ToLower(key) {
	case "token":
		p.Token = ""
	case "baseurl":
		p.BaseUrl = ""
	default:
		delete(p.Options, key)
	}
	return nil
}

// DefaultBranchKey is the config command key for the top-level branch that
// `git init -b` uses. It is addressed without a provider.
const DefaultBranchKey = "defaultbranch"

// IsDefaultBranchKey reports whether key names the top-level default branch.
func IsDefaultBranchKey(key string) bool {
	return strings.EqualFold(key, DefaultBranchKey)
}

func (c *Config) SetDefaultBranch(branch string) error {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return fmt.Errorf("%s must not be empty", DefaultBranchKey)
	}
	c.DefaultBranch = branch
	return nil
}

// ProtocolKey is the config command key for the top-level clone URL protocol.
// It is addressed without a provider.
const ProtocolKey = "protocol"

const (
	ProtocolSSH   = "ssh"
	ProtocolHTTPS = "https"
)

// IsProtocolKey reports whether key names the top-level clone URL protocol.
func IsProtocolKey(key string) bool {
	return strings.EqualFold(key, ProtocolKey)
}

// DefaultProtocol is used when protocol is not set: SSH everywhere except
// Windows, where HTTPS works with the bundled credential manager.
func DefaultProtocol() string {
	if runtime.GOOS == "windows" {
		return ProtocolHTTPS
	}
	return ProtocolSSH
}

// NormalizeProtocol lowercases and validates a protocol value.
func NormalizeProtocol(protocol string) (string, error) {
	p := strings.ToLower(strings.TrimSpace(protocol))
	if p != ProtocolSSH && p != ProtocolHTTPS {
		return "", fmt.Errorf("%s must be %q or %q, got %q", ProtocolKey, ProtocolSSH, ProtocolHTTPS, protocol)
	}
	return p, nil
}

func (c *Config) SetProtocol(protocol string) error {
	p, err := NormalizeProtocol(protocol)
	if err != nil {
		return err
	}
	c.Protocol = p
	return nil
}

// EffectiveProtocol returns the configured protocol, or DefaultProtocol when
// it is unset. An invalid value in the file is an error rather than a guess.
func (c *Config) EffectiveProtocol() (string, error) {
	if c.Protocol == "" {
		return DefaultProtocol(), nil
	}
	return NormalizeProtocol(c.Protocol)
}
