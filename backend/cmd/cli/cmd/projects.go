package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:     "projects",
	Aliases: []string{"project"},
	Short:   "Manage projects",
}

var projectsListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all projects",
	Run: func(cmd *cobra.Command, args []string) {
		res, err := Get(Cfg.URL+"/v1/projects", Cfg.ConsoleToken)
		Must(err)
		PrintList(res, "projects")
	},
}

var projectsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new project",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			name = Prompt("Project name: ")
		}
		res, err := Post(Cfg.URL+"/v1/projects", Cfg.ConsoleToken, map[string]string{"name": name})
		Must(err)
		PrettyPrint(res)
		if id, ok := res["$id"].(string); ok {
			Cfg.ProjectID = id
			SaveConfig()
			fmt.Printf("Active project set to: %s\n", id)
		}
	},
}

var projectsUseCmd = &cobra.Command{
	Use:   "use <projectId>",
	Short: "Set the active project",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		Cfg.ProjectID = args[0]
		SaveConfig()
		fmt.Printf("Active project: %s\n", Cfg.ProjectID)
	},
}

var projectsDeleteCmd = &cobra.Command{
	Use:   "delete <projectId>",
	Short: "Delete a project",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		Must(Del(Cfg.URL+"/v1/projects/"+args[0], Cfg.ConsoleToken))
		fmt.Println("Deleted.")
	},
}

func init() {
	projectsCreateCmd.Flags().String("name", "", "Project name")

	projectsCmd.AddCommand(projectsListCmd)
	projectsCmd.AddCommand(projectsCreateCmd)
	projectsCmd.AddCommand(projectsUseCmd)
	projectsCmd.AddCommand(projectsDeleteCmd)
}
