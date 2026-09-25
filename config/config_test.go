package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stefanjarina/ginit/globals"
)

func TestCreateDefaultIncludesAllProviders(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ginit.yaml")
	if err := CreateDefault(path, globals.SupportedRepos); err != nil {
		t.Fatalf("CreateDefault() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := map[string]string{
		"azure":     "https://dev.azure.com",
		"bitbucket": "https://api.bitbucket.org/2.0",
		"forgejo":   "https://codeberg.org",
		"gitea":     "https://gitea.com",
		"github":    "https://github.com",
		"gitlab":    "https://gitlab.com",
	}
	if len(cfg.Providers) != len(want) {
		t.Fatalf("provider count = %d, want %d", len(cfg.Providers), len(want))
	}
	for provider, baseURL := range want {
		if got := cfg.GetValue(provider, "baseurl"); got != baseURL {
			t.Errorf("%s baseurl = %q, want %q", provider, got, baseURL)
		}
	}
}

func TestSetRemoveSaveLowercaseYAML(t *testing.T) {
	cfg := &Config{DefaultBranch: "main", Providers: []Provider{{Name: "github", Options: map[string]string{}}}}
	if err := cfg.SetValue("github", "token", "secret"); err != nil {
		t.Fatalf("SetValue(token) error = %v", err)
	}
	if err := cfg.SetValue("github", "OrgName", "example"); err != nil {
		t.Fatalf("SetValue(option) error = %v", err)
	}
	if err := cfg.RemoveValue("github", "token"); err != nil {
		t.Fatalf("RemoveValue(token) error = %v", err)
	}

	path := filepath.Join(t.TempDir(), "ginit.yaml")
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	got := string(data)
	for _, key := range []string{"defaultbranch:", "providers:", "baseurl:", "options:"} {
		if !strings.Contains(got, key) {
			t.Errorf("saved YAML missing lowercase key %q:\n%s", key, got)
		}
	}
	if strings.Contains(got, "DefaultBranch") || strings.Contains(got, "BaseUrl") {
		t.Errorf("saved YAML contains Go field names:\n%s", got)
	}
}

func TestDefaultBranchRoundTripsThroughSaveAndLoad(t *testing.T) {
	cfg := &Config{DefaultBranch: "main", Providers: []Provider{{Name: "github", Options: map[string]string{}}}}
	if err := cfg.SetDefaultBranch("trunk"); err != nil {
		t.Fatalf("SetDefaultBranch() error = %v", err)
	}

	path := filepath.Join(t.TempDir(), "ginit.yaml")
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.DefaultBranch != "trunk" {
		t.Fatalf("DefaultBranch = %q, want trunk", loaded.DefaultBranch)
	}
}

func TestSetDefaultBranchRejectsEmpty(t *testing.T) {
	cfg := &Config{DefaultBranch: "main"}
	for _, value := range []string{"", "   "} {
		if err := cfg.SetDefaultBranch(value); err == nil {
			t.Fatalf("SetDefaultBranch(%q) error = nil, want error", value)
		}
	}
	if cfg.DefaultBranch != "main" {
		t.Fatalf("DefaultBranch = %q, want main", cfg.DefaultBranch)
	}
}

func TestEffectiveProtocol(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "", want: DefaultProtocol()},
		{in: "ssh", want: "ssh"},
		{in: "HTTPS", want: "https"},
		{in: "git", wantErr: true},
	}
	for _, tt := range tests {
		got, err := (&Config{Protocol: tt.in}).EffectiveProtocol()
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("EffectiveProtocol(%q) = %q, %v; want %q, error %v", tt.in, got, err, tt.want, tt.wantErr)
		}
	}
}

func TestSetProtocolRejectsUnknownValue(t *testing.T) {
	cfg := &Config{Protocol: "ssh"}
	if err := cfg.SetProtocol("ftp"); err == nil {
		t.Fatal("SetProtocol(ftp) error = nil, want error")
	}
	if cfg.Protocol != "ssh" {
		t.Fatalf("protocol = %q, want ssh unchanged", cfg.Protocol)
	}
}

func TestCreateDefaultSeedsSecretPatterns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ginit.yaml")
	if err := CreateDefault(path, globals.SupportedRepos); err != nil {
		t.Fatalf("CreateDefault() error = %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !slices.Equal(cfg.SecretPatterns, DefaultSecretPatterns()) {
		t.Errorf("secretpatterns = %q, want %q", cfg.SecretPatterns, DefaultSecretPatterns())
	}
}

func TestEffectiveSecretPatterns(t *testing.T) {
	c := &Config{}
	if got := c.EffectiveSecretPatterns(); !slices.Equal(got, DefaultSecretPatterns()) {
		t.Errorf("unset secretpatterns = %q, want defaults %q", got, DefaultSecretPatterns())
	}
	c.SecretPatterns = []string{"*.secret"}
	if got := c.EffectiveSecretPatterns(); !slices.Equal(got, []string{"*.secret"}) {
		t.Errorf("configured secretpatterns = %q, want [*.secret]", got)
	}
}
