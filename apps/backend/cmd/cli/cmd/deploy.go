package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Manage deploy targets and pipelines",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		LoadConfig()
		RequireProject()
	},
}

var deployListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List deploy targets",
	Run: func(cmd *cobra.Command, args []string) {
		res, err := GetWithProject(Cfg.URL + "/v1/deploy/targets")
		Must(err)
		PrintList(res, "targets")
	},
}

var deployCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a deploy target",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		typ, _ := cmd.Flags().GetString("type")
		runtime, _ := cmd.Flags().GetString("runtime")
		entrypoint, _ := cmd.Flags().GetString("entrypoint")
		cron, _ := cmd.Flags().GetString("cron")
		envPairs, _ := cmd.Flags().GetStringArray("env")

		envVars, err := parseEnvPairs(envPairs)
		Must(err)

		body := map[string]interface{}{
			"name":    args[0],
			"envVars": envVars,
		}
		if typ != "" {
			body["type"] = typ
		}
		if runtime != "" {
			body["runtime"] = runtime
		}
		if entrypoint != "" {
			body["entrypoint"] = entrypoint
		}
		if cron != "" {
			body["cron"] = cron
		}

		res, err := PostWithProject(Cfg.URL+"/v1/deploy/targets", body)
		Must(err)
		PrettyPrint(res)
	},
}

var deployTriggerCmd = &cobra.Command{
	Use:   "trigger <targetId>",
	Short: "Trigger a serverless deploy target execution",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		data, _ := cmd.Flags().GetString("data")
		body := map[string]interface{}{"trigger": "cli"}
		if data != "" {
			body["request"] = data
		}
		res, err := PostWithProject(Cfg.URL+"/v1/deploy/targets/"+args[0]+"/executions", body)
		Must(err)
		PrettyPrint(res)
	},
}

var deployStatusCmd = &cobra.Command{
	Use:   "status <targetId>",
	Short: "Get deploy target status",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		res, err := GetWithProject(Cfg.URL + "/v1/deploy/targets/" + args[0])
		Must(err)
		PrettyPrint(res)
	},
}

var deployDeleteCmd = &cobra.Command{
	Use:   "delete <targetId>",
	Short: "Delete a deploy target",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		Must(DelWithProject(Cfg.URL + "/v1/deploy/targets/" + args[0]))
		fmt.Println("Deleted.")
	},
}

func init() {
	deployCreateCmd.Flags().String("type", "web", "Target type, e.g. web, serverless")
	deployCreateCmd.Flags().String("runtime", "", "Runtime for the target")
	deployCreateCmd.Flags().String("entrypoint", "", "Entrypoint for the target")
	deployCreateCmd.Flags().String("cron", "", "Cron schedule for the target")
	deployCreateCmd.Flags().StringArray("env", nil, "Environment variable as KEY=VALUE (repeatable)")

	deployTriggerCmd.Flags().String("data", "", "Request payload passed to a serverless target")

	deployCmd.AddCommand(deployListCmd)
	deployCmd.AddCommand(deployCreateCmd)
	deployCmd.AddCommand(deployTriggerCmd)
	deployCmd.AddCommand(deployStatusCmd)
	deployCmd.AddCommand(deployDeleteCmd)
}
