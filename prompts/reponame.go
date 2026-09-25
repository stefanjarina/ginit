package prompts

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/stefanjarina/ginit/api"
)

// ValidateRepoName checks name against the rules of provider, so a bad name
// is rejected in the prompt instead of failing later as an API error.
func ValidateRepoName(provider, name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("repository name is required")
	}
	if strings.HasSuffix(strings.ToLower(name), ".git") {
		return errors.New("repository name must not end with .git")
	}

	switch provider {
	case "github":
		return validateSimpleName(name, 100)
	case "gitea", "forgejo":
		if err := validateSimpleName(name, 100); err != nil {
			return err
		}
		for _, suffix := range []string{".wiki", ".rss", ".atom"} {
			if strings.HasSuffix(strings.ToLower(name), suffix) {
				return fmt.Errorf("repository name must not end with %s", suffix)
			}
		}
		return nil
	case "gitlab":
		return validateGitlabName(name)
	case "azure":
		return validateAzureName(name)
	case "bitbucket":
		return validateBitbucketName(name)
	default:
		return nil
	}
}

// validateSimpleName enforces the GitHub-style rule set: ASCII letters,
// digits, '.', '-' and '_' only.
func validateSimpleName(name string, maxLen int) error {
	if len(name) > maxLen {
		return fmt.Errorf("repository name must be at most %d characters", maxLen)
	}
	if name == "." || name == ".." {
		return fmt.Errorf("repository name must not be %q", name)
	}
	for _, r := range name {
		if !isSimpleNameRune(r) {
			return fmt.Errorf("repository name contains %s; use only letters, digits, '.', '-' and '_'", describeRune(r))
		}
	}
	return nil
}

func isSimpleNameRune(r rune) bool {
	return r < utf8.RuneSelf && (unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-' || r == '_')
}

// validateGitlabName follows GitLab's project name rules: letters, digits,
// emoji, '_', '.', '+', '-' and spaces, starting with a letter, digit, emoji
// or '_'.
func validateGitlabName(name string) error {
	if utf8.RuneCountInString(name) > 255 {
		return errors.New("repository name must be at most 255 characters")
	}
	first, _ := utf8.DecodeRuneInString(name)
	if !(unicode.IsLetter(first) || unicode.IsDigit(first) || first == '_' || unicode.Is(unicode.So, first)) {
		return errors.New("repository name must start with a letter, digit, emoji or '_'")
	}
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.So, r) || strings.ContainsRune("_.+- ", r) {
			continue
		}
		return fmt.Errorf("repository name contains %s; use only letters, digits, emoji, spaces, '_', '.', '+' and '-'", describeRune(r))
	}
	return nil
}

// validateAzureName follows the Azure DevOps naming restrictions for Git
// repositories.
func validateAzureName(name string) error {
	if utf8.RuneCountInString(name) > 64 {
		return errors.New("repository name must be at most 64 characters")
	}
	if strings.HasPrefix(name, "_") || strings.HasPrefix(name, ".") {
		return errors.New("repository name must not start with '_' or '.'")
	}
	if strings.HasSuffix(name, ".") {
		return errors.New("repository name must not end with '.'")
	}
	for _, r := range name {
		if unicode.IsControl(r) || strings.ContainsRune(`\/:*?"<>|;#${},+=[]`, r) {
			return fmt.Errorf("repository name contains %s, which Azure DevOps does not allow", describeRune(r))
		}
	}
	return nil
}

// validateBitbucketName accepts any display name that produces a usable
// slug; the slug is what goes into the URL.
func validateBitbucketName(name string) error {
	slug := api.BitbucketSlug(name)
	if slug == "" {
		return errors.New("repository name needs at least one letter or digit")
	}
	if len(slug) > 62 {
		return errors.New("repository name must be at most 62 characters")
	}
	if strings.HasSuffix(slug, ".git") {
		return errors.New("repository name must not end with .git")
	}
	return nil
}

func describeRune(r rune) string {
	if r == ' ' {
		return "a space"
	}
	if unicode.IsPrint(r) {
		return fmt.Sprintf("%q", r)
	}
	return fmt.Sprintf("%U", r)
}
