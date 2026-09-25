package config

type Provider struct {
	Name    string            `yaml:"name"`
	Token   string            `yaml:"token"`
	BaseUrl string            `yaml:"base_url"`
	Options map[string]string `yaml:"options"`
}
