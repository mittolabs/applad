package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var workflowsCmd = &cobra.Command{
	Use:     "workflows",
	Aliases: []string{"wf"},
	Short:   "Manage and execute workflows",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		LoadConfig()
		RequireProject()
	},
}

var wfListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all workflows",
	Run: func(cmd *cobra.Command, args []string) {
		res, err := GetWithProject(Cfg.URL + "/v1/workflows")
		Must(err)
		PrintList(res, "workflows")
	},
}

var wfExecuteCmd = &cobra.Command{
	Use:     "execute <workflowId>",
	Aliases: []string{"run"},
	Short:   "Execute a workflow",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		data, _ := cmd.Flags().GetString("data")
		body := map[string]interface{}{}
		if data != "" {
			json.Unmarshal([]byte(data), &body) //nolint:errcheck
		}
		res, err := PostWithProject(Cfg.URL+"/v1/workflows/"+args[0]+"/executions", body)
		Must(err)
		PrettyPrint(res)
	},
}

var wfLogsCmd = &cobra.Command{
	Use:   "logs <workflowId>",
	Short: "List workflow executions",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		res, err := GetWithProject(Cfg.URL + "/v1/workflows/" + args[0] + "/executions")
		Must(err)
		PrintList(res, "executions")
	},
}

var wfGetCmd = &cobra.Command{
	Use:   "get <workflowId>",
	Short: "Get a workflow",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		res, err := GetWithProject(Cfg.URL + "/v1/workflows/" + args[0])
		Must(err)
		PrettyPrint(res)
	},
}

var wfDeleteCmd = &cobra.Command{
	Use:   "delete <workflowId>",
	Short: "Delete a workflow",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		Must(DelWithProject(Cfg.URL + "/v1/workflows/" + args[0]))
		fmt.Println("Deleted.")
	},
}

func init() {
	wfExecuteCmd.Flags().String("data", "", "JSON trigger data")

	workflowsCmd.AddCommand(wfListCmd)
	workflowsCmd.AddCommand(wfExecuteCmd)
	workflowsCmd.AddCommand(wfLogsCmd)
	workflowsCmd.AddCommand(wfGetCmd)
	workflowsCmd.AddCommand(wfDeleteCmd)
}
