package initcmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stefanjarina/ginit/api"
	c "github.com/stefanjarina/ginit/config"
)

var githubCmd = &cobra.Command{
	Use:   "github [repo_name] [description]",
	Short: "Initialize repo for GitHub",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		var config c.Config
		err := viper.Unmarshal(&config)
		if err != nil {
			log.Fatalf("unable to decode config file, %v", err)
		}

		var token string
		for _, provider := range config.Providers {
			if provider.Name == "github" {
				token = provider.Token
			}
		}

		err, token, name, description, visibility, localFiles, gitignore := api.GetAnswers("github", token)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Name: %s\n", name)
		fmt.Printf("Name: %s\n", description)
		fmt.Printf("Name: %s\n", visibility)
		fmt.Printf("Name: %v\n", localFiles)
		fmt.Printf("Name: %v\n", gitignore)
	},
}

func init() {
}
