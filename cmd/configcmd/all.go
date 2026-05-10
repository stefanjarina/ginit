package configcmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stefanjarina/ginit/config"
	"gopkg.in/yaml.v2"
)

var allCmd = &cobra.Command{
	Use:   "all [provider]",
	Short: "Print configuration",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var value any = config.Current
		if len(args) == 1 {
			p := config.Current.GetProvider(args[0])
			if p == nil {
				return fmt.Errorf("unknown provider: %s", args[0])
			}
			value = p
		}
		data, err := yaml.Marshal(value)
		if err != nil {
			return fmt.Errorf("marshal config: %w", err)
		}
		fmt.Print(string(data))
		return nil
	},
}
