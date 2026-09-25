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

	if _, err := executeConfig("set", "defaultbranch", "trunk"); err != nil {
		t.Fatalf("config set defaultbranch error = %v", err)
	}
	loaded, err := config.Load(config.CurrentPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.DefaultBranch != "trunk" {
		t.Fatalf("saved defaultbranch = %q, want trunk", loaded.DefaultBranch)
	}

	out, err := executeConfig("get", "defaultbranch")
	if err != nil {
		t.Fatalf("config get defaultbranch error = %v", err)
	}
	if strings.TrimSpace(out) != "trunk" {
		t.Fatalf("config get defaultbranch output = %q, want trunk", out)
	}

	rejected := [][]string{
		{"set", "defaultbranch", ""},
		{"set", "defaultbranch", "  "},
		{"set", "defaultbranch"},
		{"set", "defaultbranch", "a", "b"},
		{"get", "defaultbranch", "extra"},
		{"remove", "defaultbranch"},
	}
	for _, args := range rejected {
		if _, err := executeConfig(args...); err == nil {
			t.Fatalf("config %q error = nil, want error", args)
		}
	}
	if config.Current.DefaultBranch != "trunk" {
		t.Fatalf("defaultbranch after rejected commands = %q, want trunk", config.Current.DefaultBranch)
	}
}

func TestConfigAllOptionalProvider(t *testing.T) {
	config.Current = &config.Config{
		DefaultBranch: "main",
		Providers: []config.Provider{
			{Name: "azure", Token: "azure-token", BaseUrl: "https://dev.azure.com", Options: map[string]string{"OrgName": "contoso"}},
			{Name: "github", Token: "github-token", BaseUrl: "https://github.com", Options: map[string]string{}},
		},
	}
	config.CurrentPath = filepath.Join(t.TempDir(), "ginit.yaml")

	out, err := executeConfig("all")
	if err != nil {
		t.Fatalf("config all error = %v", err)
	}
	if !strings.Contains(out, "defaultbranch: main") || !strings.Contains(out, "name: azure") || !strings.Contains(out, "name: github") {
		t.Fatalf("config all output missing full config fields:\n%s", out)
	}

	out, err = executeConfig("all", "azure")
	if err != nil {
		t.Fatalf("config all azure error = %v", err)
	}
	if !strings.Contains(out, "name: azure") || !strings.Contains(out, "OrgName: contoso") || strings.Contains(out, "name: github") || strings.Contains(out, "defaultbranch:") {
		t.Fatalf("config all azure output did not isolate provider:\n%s", out)
	}

	if _, err := executeConfig("all", "unknown"); err == nil {
		t.Fatal("config all unknown error = nil, want error")
	}
	if _, err := executeConfig("all", "azure", "extra"); err == nil {
		t.Fatal("config all azure extra error = nil, want error")
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
