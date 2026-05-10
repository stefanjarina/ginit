package model

type ProjectInfo struct {
	Name               string
	Description        string
	Visibility         string
	RemoteUrl          string
	GitIgnoreConfigs   []string
	ExcludedLocalFiles []string
}
