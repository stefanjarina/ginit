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
	DefaultBranch string `yaml:"default_branch"`
	// Protocol selects the clone URL used for origin: "ssh" or "https".
	// Empty means the platform default, see DefaultProtocol.
	Protocol string `yaml:"protocol,omitempty"`
	// SecretPatterns lists the file patterns that make init ask before the
	// initial commit. Empty means DefaultSecretPatterns, see
	// EffectiveSecretPatterns.
	SecretPatterns []string   `yaml:"secret_patterns,omitempty"`
	Providers      []Provider `yaml:"providers"`
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
	if err := checkLegacyKeys(data); err != nil {
		return nil, gerrors.NewHint(fmt.Sprintf("config %s: %v", path, err), "rename the keys to snake_case", nil)
	}
	for i := range c.Providers {
		c.Providers[i].Options = canonicalOptions(c.Providers[i].Options)
	}
	return &c, nil
}

// Save writes c to path. The file, and the config directory when Save creates
// it, are restricted to the current user because the config holds tokens.
func Save(path string, c *Config) error {
	if err := mkdirPrivate(filepath.Dir(path)); err != nil {
		return gerrors.New("create config dir", err)
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return gerrors.New("marshal config", err)
	}
	if err := writePrivate(path, data); err != nil {
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
	c := &Config{DefaultBranch: "main", SecretPatterns: DefaultSecretPatterns(), Providers: providers}
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

// GetValue reads key from provider. "token" and "base_url" are fields of the
// provider; everything else lives in Options. Keys match as described in
// CanonicalKey, so "OrgName", "orgname" and "org_name" are the same key.
func (c *Config) GetValue(provider, key string) string {
	p := c.GetProvider(provider)
	if p == nil {
		return ""
	}
	switch CanonicalKey(key) {
	case TokenKey:
		return p.Token
	case BaseUrlKey:
		return p.BaseUrl
	default:
		return p.Options[CanonicalKey(key)]
	}
}

// SetValue stores value under the canonical form of key.
func (c *Config) SetValue(provider, key, value string) error {
	p := c.GetProvider(provider)
	if p == nil {
		return fmt.Errorf("unknown provider: %s", provider)
	}
	switch k := CanonicalKey(key); k {
	case "":
		return fmt.Errorf("key must not be empty")
	case TokenKey:
		p.Token = value
	case BaseUrlKey:
		p.BaseUrl = value
	default:
		if p.Options == nil {
			p.Options = map[string]string{}
		}
		p.Options[k] = value
	}
	return nil
}

func (c *Config) RemoveValue(provider, key string) error {
	p := c.GetProvider(provider)
	if p == nil {
		return fmt.Errorf("unknown provider: %s", provider)
	}
	switch k := CanonicalKey(key); k {
	case TokenKey:
		p.Token = ""
	case BaseUrlKey:
		p.BaseUrl = ""
	default:
		delete(p.Options, k)
	}
	return nil
}

// DefaultBranchKey is the config command key for the top-level branch that
// `git init -b` uses. It is addressed without a provider.
const DefaultBranchKey = "default_branch"

// IsDefaultBranchKey reports whether key names the top-level default branch.
func IsDefaultBranchKey(key string) bool {
	return CanonicalKey(key) == DefaultBranchKey
}

// FallbackDefaultBranch is the branch used when default_branch is not set.
const FallbackDefaultBranch = "main"

// EffectiveDefaultBranch returns the configured default branch, or
// FallbackDefaultBranch when it is empty.
func (c *Config) EffectiveDefaultBranch() string {
	if branch := strings.TrimSpace(c.DefaultBranch); branch != "" {
		return branch
	}
	return FallbackDefaultBranch
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
	return CanonicalKey(key) == ProtocolKey
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

// DefaultSecretPatterns returns the file patterns that likely hold secrets.
// A new config is seeded with them, and they apply whenever the config does
// not list any. See gitops.SensitivePaths for the pattern syntax.
func DefaultSecretPatterns() []string {
	return []string{
		".env", ".env.*", "!.env.example",
		"*.pem", "*.p12", "*.key",
		"id_rsa", "id_dsa", "id_ed25519",
	}
}

// EffectiveSecretPatterns returns the configured secret patterns, or
// DefaultSecretPatterns when none are configured.
func (c *Config) EffectiveSecretPatterns() []string {
	if len(c.SecretPatterns) == 0 {
		return DefaultSecretPatterns()
	}
	return c.SecretPatterns
}
