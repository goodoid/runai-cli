package template

import (
	"fmt"
	"os"

	"github.com/run-ai/runai-cli/pkg/client"
	"github.com/run-ai/runai-cli/pkg/templates"
	"github.com/spf13/cobra"
)

func NewTemplateGetCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:   "get TEMPLATE_NAME",
		Short: "Get information about one of the templates in the cluster.",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(0)
			}

			kubeClient, err := client.GetClient()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			clientset := kubeClient.GetClientset()

			templates := templates.NewTemplates(clientset)
			configName := args[0]
			config, err := templates.GetTemplate(configName)

			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			if config == nil {
				fmt.Printf("Template '%s' not found\n", configName)
				os.Exit(1)
			}

			fmt.Printf("Name: %s\n", configName)
			fmt.Printf("Description: %s\n\n", config.Description)
			fmt.Println("Values:")
			fmt.Println("---------------------------")
			fmt.Println(config.Values)
		},
	}

	return command
}
