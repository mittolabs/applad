package cmd

import (
	"github.com/spf13/cobra"
)

const version = "1.0.0"

// RootCmd is the top-level CLI command.
var RootCmd = &cobra.Command{
	Use:     "applad",
	Short:   "CLI for the Applad BaaS platform",
	Long:    "Manage projects, databases, storage, auth, functions, workflows, and more from the command line.",
	Version: version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		LoadConfig()
	},
}

func init() {
	RootCmd.AddCommand(loginCmd)
	RootCmd.AddCommand(logoutCmd)
	RootCmd.AddCommand(whoamiCmd)
	RootCmd.AddCommand(projectsCmd)
	RootCmd.AddCommand(databasesCmd)
	RootCmd.AddCommand(storageCmd)
	RootCmd.AddCommand(authCmd)
	RootCmd.AddCommand(functionsCmd)
	RootCmd.AddCommand(deployCmd)
	RootCmd.AddCommand(workflowsCmd)
	RootCmd.AddCommand(logsCmd)
	RootCmd.AddCommand(schemaCmd)
	RootCmd.AddCommand(envCmd)
	RootCmd.AddCommand(analyticsCmd)
	RootCmd.AddCommand(searchCmd)
	RootCmd.AddCommand(vectorsCmd)
	RootCmd.AddCommand(flagsCmd)
	RootCmd.AddCommand(initCmd)
}
