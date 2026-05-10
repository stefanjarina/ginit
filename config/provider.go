package config

type Provider struct {
	Name    string            `yaml:"name"`
	Token   string            `yaml:"token"`
	BaseUrl string            `yaml:"baseurl"`
	Options map[string]string `yaml:"options"`
}
