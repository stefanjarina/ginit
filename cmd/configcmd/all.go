package configcmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stefanjarina/ginit/config"
	"gopkg.in/yaml.v2"
)

// redactedToken replaces token values in `config all` output. Use
// `ginit config get <provider> token` for an explicit read.
const redactedToken = "<redacted>"

var allCmd = &cobra.Command{
	Use:   "all [provider]",
	Short: "Print configuration",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var value any
		if len(args) == 1 {
			p := config.Current.GetProvider(args[0])
			if p == nil {
				return fmt.Errorf("unknown provider: %s", args[0])
			}
			value = redactProvider(*p)
		} else {
			value = redactConfig(*config.Current)
		}
		data, err := yaml.Marshal(value)
		if err != nil {
			return fmt.Errorf("marshal config: %w", err)
		}
		fmt.Print(string(data))
		return nil
	},
}

// redactConfig returns a copy of c with every provider token redacted.
// config.Current is left untouched so a later Save cannot persist the marker.
func redactConfig(c config.Config) config.Config {
	providers := make([]config.Provider, len(c.Providers))
	for i, p := range c.Providers {
		providers[i] = redactProvider(p)
	}
	c.Providers = providers
	return c
}

// redactProvider replaces a set token with the marker. An unset token stays
// empty so the output still shows which providers lack one.
func redactProvider(p config.Provider) config.Provider {
	if p.Token != "" {
		p.Token = redactedToken
	}
	return p
}
