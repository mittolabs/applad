package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage project environment variables",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		LoadConfig()
		RequireProject()
	},
}

var envListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List environment variables",
	Run: func(cmd *cobra.Command, args []string) {
		res, err := GetWithProject(Cfg.URL + "/v1/credentials")
		Must(err)
		PrintList(res, "credentials")
	},
}

var envSetCmd = &cobra.Command{
	Use:   "set <name>",
	Short: "Set an environment variable",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		value, _ := cmd.Flags().GetString("value")
		if value == "" {
			value = Prompt("Value: ")
		}
		res, err := PostWithProject(Cfg.URL+"/v1/credentials", map[string]string{"name": args[0], "value": value})
		Must(err)
		PrettyPrint(res)
	},
}

var envDeleteCmd = &cobra.Command{
	Use:     "delete <credentialId>",
	Aliases: []string{"rm"},
	Short:   "Delete an environment variable",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		Must(DelWithProject(Cfg.URL + "/v1/credentials/" + args[0]))
		fmt.Println("Deleted.")
	},
}

func init() {
	envSetCmd.Flags().String("value", "", "Variable value")

	envCmd.AddCommand(envListCmd)
	envCmd.AddCommand(envSetCmd)
	envCmd.AddCommand(envDeleteCmd)
}
