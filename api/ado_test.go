package api

import "testing"

func TestBuildAdoOrgURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		org     string
		want    string
	}{
		{name: "default cloud", baseURL: "https://dev.azure.com", org: "contoso", want: "https://dev.azure.com/contoso"},
		{name: "trailing slash", baseURL: "https://dev.azure.com/", org: "/contoso/", want: "https://dev.azure.com/contoso"},
		{name: "custom server", baseURL: "https://ado.example.com/tfs", org: "DefaultCollection", want: "https://ado.example.com/tfs/DefaultCollection"},
		{name: "scheme added", baseURL: "ado.example.com/tfs", org: "DefaultCollection", want: "https://ado.example.com/tfs/DefaultCollection"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildAdoOrgURL(tt.baseURL, tt.org); got != tt.want {
				t.Fatalf("BuildAdoOrgURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
