package configcmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stefanjarina/ginit/config"
)

func TestConfigCommandsUsePositionalProvider(t *testing.T) {
	config.Current = &config.Config{DefaultBranch: "main", Providers: []config.Provider{{Name: "github", Options: map[string]string{}}}}
	config.CurrentPath = filepath.Join(t.TempDir(), "ginit.yaml")

	if _, err := executeConfig("set", "github", "token", "secret"); err != nil {
		t.Fatalf("config set error = %v", err)
	}
	if got := config.Current.GetValue("github", "token"); got != "secret" {
		t.Fatalf("token = %q, want secret", got)
	}

	out, err := executeConfig("get", "github", "token")
	if err != nil {
		t.Fatalf("config get error = %v", err)
	}
	if strings.TrimSpace(out) != "secret" {
		t.Fatalf("config get output = %q, want secret", out)
	}

	if _, err := executeConfig("remove", "github", "token"); err != nil {
		t.Fatalf("config remove error = %v", err)
	}
	if got := config.Current.GetValue("github", "token"); got != "" {
		t.Fatalf("token after remove = %q, want empty", got)
	}
}

func TestConfigCommandsRejectMissingArgumentsAndRepoFlag(t *testing.T) {
	config.Current = &config.Config{DefaultBranch: "main", Providers: []config.Provider{{Name: "github", Options: map[string]string{}}}}
	config.CurrentPath = filepath.Join(t.TempDir(), "ginit.yaml")

	tests := [][]string{
		{"get", "github"},
		{"set", "github", "token"},
		{"remove", "github"},
		{"get", "--repo", "github", "token"},
	}
	for _, args := range tests {
		if _, err := executeConfig(args...); err == nil {
			t.Fatalf("config %v error = nil, want error", args)
		}
	}
}

func TestConfigDefaultBranch(t *testing.T) {
	config.Current = &config.Config{DefaultBranch: "main", Providers: []config.Provider{{Name: "github", Options: map[string]string{}}}}
	config.CurrentPath = filepath.Join(t.TempDir(), "ginit.yaml")

	if _, err := executeConfig("set", "default_branch", "trunk"); err != nil {
		t.Fatalf("config set default_branch error = %v", err)
	}
	loaded, err := config.Load(config.CurrentPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.DefaultBranch != "trunk" {
		t.Fatalf("saved default_branch = %q, want trunk", loaded.DefaultBranch)
	}

	out, err := executeConfig("get", "default_branch")
	if err != nil {
		t.Fatalf("config get default_branch error = %v", err)
	}
	if strings.TrimSpace(out) != "trunk" {
		t.Fatalf("config get default_branch output = %q, want trunk", out)
	}

	rejected := [][]string{
		{"set", "default_branch", ""},
		{"set", "default_branch", "  "},
		{"set", "default_branch"},
		{"set", "default_branch", "a", "b"},
		{"get", "default_branch", "extra"},
		{"remove", "default_branch"},
	}
	for _, args := range rejected {
		if _, err := executeConfig(args...); err == nil {
			t.Fatalf("config %q error = nil, want error", args)
		}
	}
	if config.Current.DefaultBranch != "trunk" {
		t.Fatalf("default_branch after rejected commands = %q, want trunk", config.Current.DefaultBranch)
	}
}

func TestConfigSetRejectsRemovedEncryptFlag(t *testing.T) {
	config.Current = &config.Config{DefaultBranch: "main", Providers: []config.Provider{{Name: "github", Options: map[string]string{}}}}
	config.CurrentPath = filepath.Join(t.TempDir(), "ginit.yaml")

	for _, flag := range []string{"--encrypt", "-e"} {
		_, err := executeConfig("set", flag, "github", "token", "secret")
		if err == nil || !strings.Contains(err.Error(), "unknown") {
			t.Fatalf("config set %s error = %v, want unknown flag error", flag, err)
		}
		if got := config.Current.GetValue("github", "token"); got != "" {
			t.Fatalf("config set %s stored token = %q, want nothing stored", flag, got)
		}
	}
}

func TestConfigAllOptionalProvider(t *testing.T) {
	config.Current = &config.Config{
		DefaultBranch: "main",
		Providers: []config.Provider{
			{Name: "azure", Token: "azure-token", BaseUrl: "https://dev.azure.com", Options: map[string]string{"org_name": "contoso"}},
			{Name: "github", Token: "github-token", BaseUrl: "https://github.com", Options: map[string]string{}},
		},
	}
	config.CurrentPath = filepath.Join(t.TempDir(), "ginit.yaml")

	out, err := executeConfig("all")
	if err != nil {
		t.Fatalf("config all error = %v", err)
	}
	if !strings.Contains(out, "default_branch: main") || !strings.Contains(out, "name: azure") || !strings.Contains(out, "name: github") {
		t.Fatalf("config all output missing full config fields:\n%s", out)
	}

	out, err = executeConfig("all", "azure")
	if err != nil {
		t.Fatalf("config all azure error = %v", err)
	}
	if !strings.Contains(out, "name: azure") || !strings.Contains(out, "org_name: contoso") || strings.Contains(out, "name: github") || strings.Contains(out, "default_branch:") {
		t.Fatalf("config all azure output did not isolate provider:\n%s", out)
	}

	if _, err := executeConfig("all", "unknown"); err == nil {
		t.Fatal("config all unknown error = nil, want error")
	}
	if _, err := executeConfig("all", "azure", "extra"); err == nil {
		t.Fatal("config all azure extra error = nil, want error")
	}
}

func TestConfigAllRedactsTokens(t *testing.T) {
	const fixtureToken = "ghp_fixture-token-must-not-leak"
	config.Current = &config.Config{
		DefaultBranch: "main",
		Providers: []config.Provider{
			{Name: "github", Token: fixtureToken, BaseUrl: "https://github.com", Options: map[string]string{}},
			{Name: "gitlab", BaseUrl: "https://gitlab.com", Options: map[string]string{}},
		},
	}
	config.CurrentPath = filepath.Join(t.TempDir(), "ginit.yaml")

	for _, args := range [][]string{{"all"}, {"all", "github"}} {
		out, err := executeConfig(args...)
		if err != nil {
			t.Fatalf("config %v error = %v", args, err)
		}
		if strings.Contains(out, fixtureToken) {
			t.Fatalf("config %v leaked token:\n%s", args, out)
		}
		if !strings.Contains(out, "token: <redacted>") {
			t.Fatalf("config %v output missing redacted marker:\n%s", args, out)
		}
	}

	out, err := executeConfig("all", "gitlab")
	if err != nil {
		t.Fatalf("config all gitlab error = %v", err)
	}
	if !strings.Contains(out, `token: ""`) {
		t.Fatalf("config all gitlab should show an unset token as empty:\n%s", out)
	}

	if got := config.Current.GetValue("github", "token"); got != fixtureToken {
		t.Fatalf("config all mutated stored token = %q", got)
	}
	out, err = executeConfig("get", "github", "token")
	if err != nil {
		t.Fatalf("config get error = %v", err)
	}
	if strings.TrimSpace(out) != fixtureToken {
		t.Fatalf("config get token output = %q, want %q", out, fixtureToken)
	}
}

func executeConfig(args ...string) (string, error) {
	var errBuf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	ConfigCmd.SetArgs(args)
	ConfigCmd.SetOut(io.Discard)
	ConfigCmd.SetErr(&errBuf)
	ConfigCmd.SilenceUsage = true
	ConfigCmd.SilenceErrors = true
	err := ConfigCmd.Execute()

	_ = w.Close()
	out, _ := io.ReadAll(r)
	return string(out), err
}

func TestConfigProtocol(t *testing.T) {
	config.Current = &config.Config{DefaultBranch: "main", Providers: []config.Provider{{Name: "github", Options: map[string]string{}}}}
	config.CurrentPath = filepath.Join(t.TempDir(), "ginit.yaml")

	out, err := executeConfig("get", "protocol")
	if err != nil {
		t.Fatalf("config get protocol error = %v", err)
	}
	if strings.TrimSpace(out) != config.DefaultProtocol() {
		t.Fatalf("unset protocol = %q, want platform default %q", out, config.DefaultProtocol())
	}

	if _, err := executeConfig("set", "protocol", "HTTPS"); err != nil {
		t.Fatalf("config set protocol error = %v", err)
	}
	loaded, err := config.Load(config.CurrentPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Protocol != "https" {
		t.Fatalf("saved protocol = %q, want https", loaded.Protocol)
	}

	for _, args := range [][]string{
		{"set", "protocol", "git"},
		{"set", "protocol"},
		{"get", "protocol", "extra"},
		{"remove", "protocol", "extra"},
	} {
		if _, err := executeConfig(args...); err == nil {
			t.Fatalf("config %v error = nil, want error", args)
		}
	}
	if config.Current.Protocol != "https" {
		t.Fatalf("protocol after rejected commands = %q, want https", config.Current.Protocol)
	}

	if _, err := executeConfig("remove", "protocol"); err != nil {
		t.Fatalf("config remove protocol error = %v", err)
	}
	if config.Current.Protocol != "" {
		t.Fatalf("protocol after remove = %q, want empty", config.Current.Protocol)
	}
}

func TestConfigSetOptionKeyIgnoresCase(t *testing.T) {
	config.Current = &config.Config{DefaultBranch: "main", Providers: []config.Provider{{Name: "azure", Options: map[string]string{}}}}
	config.CurrentPath = filepath.Join(t.TempDir(), "ginit.yaml")

	if _, err := executeConfig("set", "azure", "orgname", "contoso"); err != nil {
		t.Fatalf("config set azure orgname error = %v", err)
	}
	loaded, err := config.Load(config.CurrentPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := loaded.GetValue("azure", config.OrgNameKey); got != "contoso" {
		t.Fatalf("saved org_name = %q, want contoso", got)
	}
	out, err := executeConfig("get", "azure", "OrgName")
	if err != nil || strings.TrimSpace(out) != "contoso" {
		t.Fatalf("config get azure OrgName = %q, %v, want contoso", out, err)
	}
}
