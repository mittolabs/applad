package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var functionsCmd = &cobra.Command{
	Use:     "functions",
	Aliases: []string{"fn"},
	Short:   "Manage and invoke functions",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		LoadConfig()
		RequireProject()
	},
}

var fnListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all functions",
	Run: func(cmd *cobra.Command, args []string) {
		res, err := GetWithProject(Cfg.URL + "/v1/functions")
		Must(err)
		PrintList(res, "functions")
	},
}

var fnInvokeCmd = &cobra.Command{
	Use:     "invoke <functionId>",
	Aliases: []string{"exec"},
	Short:   "Invoke a function",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		data, _ := cmd.Flags().GetString("data")
		body := map[string]interface{}{}
		if data != "" {
			json.Unmarshal([]byte(data), &body) //nolint:errcheck
		}
		res, err := PostWithProject(Cfg.URL+"/v1/functions/"+args[0]+"/executions", body)
		Must(err)
		PrettyPrint(res)
	},
}

var fnLogsCmd = &cobra.Command{
	Use:   "logs <functionId>",
	Short: "List function executions",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		res, err := GetWithProject(Cfg.URL + "/v1/functions/" + args[0] + "/executions")
		Must(err)
		PrintList(res, "executions")
	},
}

var fnDeleteCmd = &cobra.Command{
	Use:   "delete <functionId>",
	Short: "Delete a function",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		Must(DelWithProject(Cfg.URL + "/v1/functions/" + args[0]))
		fmt.Println("Deleted.")
	},
}

func init() {
	fnInvokeCmd.Flags().String("data", "", "JSON data to pass to the function")

	functionsCmd.AddCommand(fnListCmd)
	functionsCmd.AddCommand(fnInvokeCmd)
	functionsCmd.AddCommand(fnLogsCmd)
	functionsCmd.AddCommand(fnDeleteCmd)
}
