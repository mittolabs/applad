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

var deployTriggerCmd = &cobra.Command{
	Use:   "trigger <targetId>",
	Short: "Trigger a deployment",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		res, err := PostWithProject(Cfg.URL+"/v1/deploy/targets/"+args[0]+"/trigger", nil)
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
	deployCmd.AddCommand(deployListCmd)
	deployCmd.AddCommand(deployTriggerCmd)
	deployCmd.AddCommand(deployStatusCmd)
	deployCmd.AddCommand(deployDeleteCmd)
}
