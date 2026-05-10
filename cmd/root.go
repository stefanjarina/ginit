package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/stefanjarina/ginit/cmd/configcmd"
	"github.com/stefanjarina/ginit/cmd/gitignorecmd"
	"github.com/stefanjarina/ginit/cmd/initcmd"
	"github.com/stefanjarina/ginit/config"
	"github.com/stefanjarina/ginit/console"
	"github.com/stefanjarina/ginit/globals"
)

var (
	cfgFile       string
	accessibility bool
	verbose       bool
	noColor       bool
)

var GitTag string = "0.0.1"

var rootCmd = &cobra.Command{
	Use:     "ginit",
	Version: GitTag,
	Short:   "Custom GIT repository initializer",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		console.Verbose = verbose
		console.NoColor = noColor
		console.Accessible = accessibility
		return initConfig()
	},
	Long: ``,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// GetRootCommand returns the root command for documentation generation
func GetRootCommand() *cobra.Command {
	return rootCmd
}

func addSubCommands() {
	rootCmd.AddCommand(initcmd.InitCmd)
	rootCmd.AddCommand(configcmd.ConfigCmd)
	rootCmd.AddCommand(gitignorecmd.GitignoreCmd)
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/ginit/ginit.yaml)")
	rootCmd.PersistentFlags().BoolVar(&accessibility, "accessibility", false, "Enable accessibility features")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output (full error chains)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")

	addSubCommands()
}

// initConfig resolves the config path, creates a default config if missing,
// loads it into config.Current, and records the path in config.CurrentPath.
func initConfig() error {
	path := cfgFile
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		path = filepath.Join(home, ".config", "ginit", "ginit.yaml")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := config.CreateDefault(path, globals.SupportedRepos); err != nil {
			return err
		}
		console.Info("Created default config at " + path)
	}

	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	config.Current = cfg
	config.CurrentPath = path
	return nil
}

// Accessibility returns the value of the --accessibility persistent flag.
// Used by command bodies that build huh forms.
func Accessibility() bool { return accessibility }
