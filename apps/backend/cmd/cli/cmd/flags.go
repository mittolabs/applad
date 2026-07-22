package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var flagsCmd = &cobra.Command{
	Use:   "flags",
	Short: "Manage feature flags",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		LoadConfig()
		RequireProject()
	},
}

var flagsListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all flags",
	Run: func(cmd *cobra.Command, args []string) {
		res, err := GetWithProject(Cfg.URL + "/v1/flags")
		Must(err)
		PrintList(res, "flags")
	},
}

var flagsGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a flag by key",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		res, err := GetWithProject(Cfg.URL + "/v1/flags/" + args[0])
		Must(err)
		PrettyPrint(res)
	},
}

var flagsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a feature flag",
	Run: func(cmd *cobra.Command, args []string) {
		key, _ := cmd.Flags().GetString("key")
		name, _ := cmd.Flags().GetString("name")
		if key == "" {
			key = Prompt("Flag key: ")
		}
		if name == "" {
			name = key
		}
		res, err := PostWithProject(Cfg.URL+"/v1/flags", map[string]string{
			"key":  key,
			"name": name,
		})
		Must(err)
		PrettyPrint(res)
	},
}

var flagsToggleCmd = &cobra.Command{
	Use:   "toggle <key>",
	Short: "Toggle a flag on or off",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		enabled, _ := cmd.Flags().GetBool("enabled")
		res, err := PatchWithProject(Cfg.URL+"/v1/flags/"+args[0]+"/toggle", map[string]interface{}{
			"enabled": enabled,
		})
		Must(err)
		PrettyPrint(res)
	},
}

var flagsDeleteCmd = &cobra.Command{
	Use:   "delete <key>",
	Short: "Delete a flag",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		Must(DelWithProject(Cfg.URL + "/v1/flags/" + args[0]))
		fmt.Println("Deleted.")
	},
}

var flagsEvalCmd = &cobra.Command{
	Use:   "eval <key>",
	Short: "Evaluate a flag",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		res, err := GetWithProject(Cfg.URL + "/v1/flags/evaluate/" + args[0])
		Must(err)
		PrettyPrint(res)
	},
}

func init() {
	flagsCreateCmd.Flags().String("key", "", "Flag key")
	flagsCreateCmd.Flags().String("name", "", "Flag display name")
	flagsToggleCmd.Flags().Bool("enabled", true, "Enable or disable the flag")

	flagsCmd.AddCommand(flagsListCmd)
	flagsCmd.AddCommand(flagsGetCmd)
	flagsCmd.AddCommand(flagsCreateCmd)
	flagsCmd.AddCommand(flagsToggleCmd)
	flagsCmd.AddCommand(flagsDeleteCmd)
	flagsCmd.AddCommand(flagsEvalCmd)
}
