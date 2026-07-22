package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage users and sessions",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		LoadConfig()
		RequireProject()
	},
}

var authUsersCmd = &cobra.Command{
	Use:     "users",
	Aliases: []string{"list"},
	Short:   "List users",
	Run: func(cmd *cobra.Command, args []string) {
		res, err := GetWithProject(Cfg.URL + "/v1/users")
		Must(err)
		PrintList(res, "users")
	},
}

var authGetCmd = &cobra.Command{
	Use:   "get <userId>",
	Short: "Get a user by ID",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		res, err := GetWithProject(Cfg.URL + "/v1/users/" + args[0])
		Must(err)
		PrettyPrint(res)
	},
}

var authCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a user",
	Run: func(cmd *cobra.Command, args []string) {
		email, _ := cmd.Flags().GetString("email")
		if email == "" {
			email = Prompt("Email: ")
		}
		res, err := PostWithProject(Cfg.URL+"/v1/users", map[string]string{"email": email})
		Must(err)
		PrettyPrint(res)
	},
}

var authDeleteCmd = &cobra.Command{
	Use:   "delete <userId>",
	Short: "Delete a user",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		Must(DelWithProject(Cfg.URL + "/v1/users/" + args[0]))
		fmt.Println("Deleted.")
	},
}

func init() {
	authCreateCmd.Flags().String("email", "", "User email")

	authCmd.AddCommand(authUsersCmd)
	authCmd.AddCommand(authGetCmd)
	authCmd.AddCommand(authCreateCmd)
	authCmd.AddCommand(authDeleteCmd)
}
