package cmd

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stefanjarina/ginit/cmd/configcmd"
	"github.com/stefanjarina/ginit/cmd/initcmd"
	"github.com/stefanjarina/ginit/config"
	"github.com/stefanjarina/ginit/globals"
	"gopkg.in/yaml.v2"
)

var cfgFile string
var accessibility bool

var rootCmd = &cobra.Command{
	Use:     "ginit",
	Version: "0.0.1",
	Short:   "Custom GIT repository initializer",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// You can bind cobra and viper in a few locations, but PersistencePreRunE on the root command works well
		return initConfig(cmd)
	},
	Long: ``,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func addSubCommands() {
	rootCmd.AddCommand(initcmd.InitCmd)
	rootCmd.AddCommand(configcmd.ConfigCmd)
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/ginit/ginit.yaml)")
	rootCmd.PersistentFlags().BoolVar(&accessibility, "accessibility", false, "Enable accessibility features")
	_ = viper.BindPFlag("accessibility", rootCmd.PersistentFlags().Lookup("accessibility"))

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	addSubCommands()
}

// initConfig reads in config file and ENV variables if set.
func initConfig(cmd *cobra.Command) error {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".ginit" (without extension).
		var cfgPath = path.Join(home, ".config", "ginit")
		viper.AddConfigPath(cfgPath)
		viper.SetConfigType("yaml")
		viper.SetConfigName("ginit")

		filePath := path.Join(cfgPath, "ginit.yaml")

		// Create default config file if it does not exist
		if _, err := os.Stat(filepath.Dir(filePath)); os.IsNotExist(err) {
			createDefaultConfigFile(filePath)
		}
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal(err)
	}

	_, err := fmt.Fprintln(os.Stdout, "Using config file:", viper.ConfigFileUsed())
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func createDefaultConfigFile(filePath string) {
	var providers []config.Provider

	for _, p := range globals.SupportedRepos {
		provider := config.Provider{
			Name:    p,
			BaseUrl: "",
			Token:   "",
			Options: make(map[string]string),
		}
		providers = append(providers, provider)
	}

	configuration := config.Config{
		DefaultBranch: "main",
		Providers:     providers,
	}

	yamlData, err := yaml.Marshal(&configuration)
	if err != nil {
		log.Fatal("Error while creating default configuration", err)
	}

	err = os.MkdirAll(filepath.Dir(filePath), 0644)
	if err != nil {
		log.Fatal("Error creating config directory ", err)
	}

	err = os.WriteFile(filePath, yamlData, 0644)
	if err != nil {
		log.Fatal("Error while saving default config file:", err)
	}
}
