package config

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"gopkg.in/yaml.v2"
)

// Provider keys handled by the config command and init.
const (
	TokenKey   = "token"
	BaseUrlKey = "base_url"
	// OrgNameKey is the Azure DevOps organization.
	OrgNameKey = "org_name"
	// UserKey is the optional Bitbucket Data Center / Server user.
	UserKey = "user"
)

// knownKeys are the keys ginit reads. A key that folds to one of them takes
// its spelling, so "orgname" becomes "org_name" rather than staying "orgname".
var knownKeys = []string{TokenKey, BaseUrlKey, OrgNameKey, UserKey, DefaultBranchKey, ProtocolKey}

// foldKey is the form keys are compared in: lowercase, without "_" and "-".
func foldKey(key string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(key) {
		if r == '_' || r == '-' {
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// CanonicalKey returns the snake_case form a key is stored under. Keys that
// differ only in case, "_" or "-" are the same key: "OrgName", "orgname",
// "org-name" and "org_name" all return "org_name". Unknown keys are converted
// to snake_case, so "customKey" is stored as "custom_key".
func CanonicalKey(key string) string {
	fold := foldKey(key)
	for _, k := range knownKeys {
		if foldKey(k) == fold {
			return k
		}
	}
	return toSnakeCase(strings.TrimSpace(key))
}

func toSnakeCase(s string) string {
	runes := []rune(s)
	var b strings.Builder
	for i, r := range runes {
		switch {
		case r == '-' || r == ' ':
			r = '_'
		case unicode.IsUpper(r):
			// Start a new word at "aB" and at the "C" of "ABCd".
			if i > 0 && runes[i-1] != '_' && runes[i-1] != '-' &&
				(unicode.IsLower(runes[i-1]) || unicode.IsDigit(runes[i-1]) ||
					(i+1 < len(runes) && unicode.IsLower(runes[i+1]) && unicode.IsUpper(runes[i-1]))) {
				b.WriteRune('_')
			}
			r = unicode.ToLower(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}

// canonicalOptions rewrites option keys to their canonical form. When two keys
// collapse to one, the one already canonical wins, otherwise the first in
// sorted order, so the result does not depend on map iteration.
func canonicalOptions(opts map[string]string) map[string]string {
	out := make(map[string]string, len(opts))
	keys := make([]string, 0, len(opts))
	for k := range opts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		ck := CanonicalKey(k)
		if ck == "" {
			continue
		}
		if _, taken := out[ck]; taken && k != ck {
			continue
		}
		out[ck] = opts[k]
	}
	return out
}

// legacyKeys maps keys used before the switch to snake_case to their
// replacements. yaml.v2 ignores unknown keys, so without this check an old
// file would load with the values silently dropped.
var legacyKeys = map[string]string{
	"defaultbranch":  DefaultBranchKey,
	"secretpatterns": "secret_patterns",
	"baseurl":        BaseUrlKey,
}

func checkLegacyKeys(data []byte) error {
	var raw struct {
		Top       map[string]any   `yaml:",inline"`
		Providers []map[string]any `yaml:"providers"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return err
	}
	var found []string
	for old, repl := range legacyKeys {
		if _, ok := raw.Top[old]; ok {
			found = append(found, fmt.Sprintf("%s -> %s", old, repl))
		}
	}
	for _, p := range raw.Providers {
		for old, repl := range legacyKeys {
			if _, ok := p[old]; ok {
				found = append(found, fmt.Sprintf("providers[%v].%s -> %s", p["name"], old, repl))
			}
		}
	}
	if len(found) == 0 {
		return nil
	}
	sort.Strings(found)
	return fmt.Errorf("outdated keys: %s", strings.Join(found, ", "))
}
