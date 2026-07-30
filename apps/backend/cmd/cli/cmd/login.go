package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// errMFARequired signals that the password was accepted but the account has
// MFA enrolled and a second-factor code is still needed.
var errMFARequired = errors.New("console_mfa_required")

// consoleLogin posts credentials to /v1/console/login. Unlike the generic Post
// helper it inspects the error "type" so the caller can tell an MFA challenge
// (console_mfa_required) apart from a genuine failure and re-submit with a code.
func consoleLogin(url string, body map[string]string) (map[string]interface{}, error) {
	b, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, url+"/v1/console/login", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&out) //nolint:errcheck
	if resp.StatusCode >= 400 {
		if t, _ := out["type"].(string); t == "console_mfa_required" {
			return nil, errMFARequired
		}
		msg, _ := out["message"].(string)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, msg)
	}
	return out, nil
}

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
		res, err := consoleLogin(url, body)
		if errors.Is(err, errMFARequired) {
			// Password was correct but the account has MFA enrolled. Ask for the
			// code and re-submit, matching what the console UI does.
			body["code"] = Prompt("Authentication code: ")
			res, err = consoleLogin(url, body)
		}
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
