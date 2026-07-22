package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate and store credentials",
	Run: func(cmd *cobra.Command, args []string) {
		url, _ := cmd.Flags().GetString("url")
		email, _ := cmd.Flags().GetString("email")
		password, _ := cmd.Flags().GetString("password")

		if email == "" {
			email = Prompt("Email: ")
		}
		if password == "" {
			password = Prompt("Password: ")
		}

		body := map[string]string{"email": email, "password": password}
		res, err := Post(url+"/v1/console/login", "", body)
		Must(err)

		token, _ := res["token"].(string)
		if token == "" {
			fmt.Println("Login failed.")
			return
		}
		Cfg.URL = url
		Cfg.ConsoleToken = token
		SaveConfig()
		fmt.Println("Logged in successfully.")

		// Prompt to select a project
		fmt.Println("\nProjects:")
		projects, _ := APIList(url+"/v1/projects", token)
		for i, p := range projects {
			fmt.Printf("  [%d] %s (%s)\n", i+1, p["name"], p["$id"])
		}
		if len(projects) > 0 {
			choice := Prompt("Select project number (or press Enter to skip): ")
			if n, err := strconv.Atoi(strings.TrimSpace(choice)); err == nil && n >= 1 && n <= len(projects) {
				Cfg.ProjectID, _ = projects[n-1]["$id"].(string)
				fmt.Printf("Project set to: %s\n", Cfg.ProjectID)
			}
		}
		SaveConfig()
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored credentials",
	Run: func(cmd *cobra.Command, args []string) {
		Cfg = Config{}
		SaveConfig()
		fmt.Println("Logged out.")
	},
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current user",
	Run: func(cmd *cobra.Command, args []string) {
		if Cfg.ConsoleToken == "" {
			fmt.Println("Not logged in. Run 'applad login'.")
			return
		}
		res, err := Get(Cfg.URL+"/v1/console/me", Cfg.ConsoleToken)
		Must(err)
		PrettyPrint(res)
	},
}

func init() {
	loginCmd.Flags().String("url", Getenv("APPLAD_URL", "http://localhost:80"), "API base URL")
	loginCmd.Flags().String("email", "", "Account email")
	loginCmd.Flags().String("password", "", "Account password")
}
