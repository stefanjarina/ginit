package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalKey(t *testing.T) {
	tests := map[string]string{
		"OrgName":       "org_name",
		"orgname":       "org_name",
		"org-name":      "org_name",
		"ORG_NAME":      "org_name",
		"User":          "user",
		"baseurl":       "base_url",
		"BaseUrl":       "base_url",
		"TOKEN":         "token",
		"defaultbranch": "default_branch",
		"DefaultBranch": "default_branch",
		"customKey":     "custom_key",
		"custom-key":    "custom_key",
		"HTTPTimeout":   "http_timeout",
		"already_snake": "already_snake",
		"  spaced  ":    "spaced",
	}
	for in, want := range tests {
		if got := CanonicalKey(in); got != want {
			t.Errorf("CanonicalKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestOptionKeysIgnoreCase(t *testing.T) {
	cfg := &Config{Providers: []Provider{{Name: "azure", Options: map[string]string{}}}}
	if err := cfg.SetValue("azure", "orgname", "contoso"); err != nil {
		t.Fatalf("SetValue(orgname) error = %v", err)
	}
	if got := cfg.GetValue("azure", "OrgName"); got != "contoso" {
		t.Errorf("GetValue(OrgName) = %q, want contoso", got)
	}
	if err := cfg.SetValue("azure", "OrgName", "fabrikam"); err != nil {
		t.Fatalf("SetValue(OrgName) error = %v", err)
	}
	opts := cfg.GetProvider("azure").Options
	if len(opts) != 1 || opts[OrgNameKey] != "fabrikam" {
		t.Errorf("options = %v, want only %s: fabrikam", opts, OrgNameKey)
	}
	if err := cfg.RemoveValue("azure", "ORGNAME"); err != nil {
		t.Fatalf("RemoveValue(ORGNAME) error = %v", err)
	}
	if len(opts) != 0 {
		t.Errorf("options after remove = %v, want empty", opts)
	}
}

func TestSetValueUnknownProvider(t *testing.T) {
	cfg := &Config{Providers: []Provider{{Name: "github", Options: map[string]string{}}}}
	for _, key := range []string{"token", "base_url", "org_name"} {
		err := cfg.SetValue("azure", key, "value")
		if err == nil || !strings.Contains(err.Error(), "unknown provider") {
			t.Errorf("SetValue(azure, %s) error = %v, want unknown provider", key, err)
		}
	}
	if err := cfg.RemoveValue("azure", "token"); err == nil {
		t.Error("RemoveValue(azure) error = nil, want unknown provider")
	}
}

func TestSetValueEmptyKey(t *testing.T) {
	cfg := &Config{Providers: []Provider{{Name: "github", Options: map[string]string{}}}}
	if err := cfg.SetValue("github", "  ", "value"); err == nil {
		t.Error("SetValue with empty key error = nil, want error")
	}
}

func TestLoadCanonicalizesOptionKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ginit.yaml")
	data := "default_branch: main\nproviders:\n- name: azure\n  options:\n    OrgName: contoso\n    customKey: x\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	opts := cfg.GetProvider("azure").Options
	if opts["org_name"] != "contoso" || opts["custom_key"] != "x" || len(opts) != 2 {
		t.Errorf("options = %v, want org_name and custom_key", opts)
	}
}

func TestLoadCollidingOptionKeysPrefersCanonical(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ginit.yaml")
	data := "providers:\n- name: azure\n  options:\n    OrgName: old\n    org_name: new\n    orgname: other\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if opts := cfg.GetProvider("azure").Options; len(opts) != 1 || opts["org_name"] != "new" {
		t.Errorf("options = %v, want only org_name: new", opts)
	}
}

func TestLoadRejectsLegacyKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ginit.yaml")
	data := "defaultbranch: main\nproviders:\n- name: github\n  baseurl: https://github.com\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want outdated keys error")
	}
	for _, want := range []string{"defaultbranch -> default_branch", "providers[github].baseurl -> base_url"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Load() error = %q, want it to contain %q", err, want)
		}
	}
}
