package prompts

import (
	"reflect"
	"testing"
)

func TestVisibilityChoices(t *testing.T) {
	tests := map[string][]string{
		"github":    {"private", "public"},
		"azure":     {"private", "public"},
		"bitbucket": {"private", "public"},
		"gitlab":    {"private", "internal", "public"},
		"gitea":     {"private", "limited", "public"},
		"forgejo":   {"private", "limited", "public"},
	}
	for provider, want := range tests {
		if got := VisibilityChoices(provider); !reflect.DeepEqual(got, want) {
			t.Errorf("VisibilityChoices(%q) = %#v, want %#v", provider, got, want)
		}
	}
}
