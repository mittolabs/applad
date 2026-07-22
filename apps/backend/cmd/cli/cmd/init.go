package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Applad project configuration",
	Long:  "Creates an .applad.yaml config file in the current directory.",
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Stat(".applad.yaml"); err == nil {
			fmt.Println(".applad.yaml already exists.")
			return
		}

		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			name = Prompt("Project name: ")
		}

		url, _ := cmd.Flags().GetString("url")

		config := map[string]interface{}{
			"name":     name,
			"endpoint": url,
		}

		// If logged in, create the project on the server
		if Cfg.ConsoleToken != "" {
			fmt.Println("Creating project on server...")
			res, err := Post(Cfg.URL+"/v1/projects", Cfg.ConsoleToken, map[string]string{"name": name})
			if err == nil {
				if id, ok := res["$id"].(string); ok {
					config["projectId"] = id
					Cfg.ProjectID = id
					SaveConfig()
					fmt.Printf("Project created: %s\n", id)

					// Fetch the API key
					keys, kerr := Get(Cfg.URL+"/v1/projects/"+id+"/keys", Cfg.ConsoleToken)
					if kerr == nil {
						if keyList, ok := keys["keys"].([]interface{}); ok && len(keyList) > 0 {
							if k, ok := keyList[0].(map[string]interface{}); ok {
								config["apiKey"], _ = k["secret"].(string)
							}
						}
					}
				}
			} else {
				fmt.Printf("Warning: could not create project: %v\n", err)
			}
		}

		data, _ := json.MarshalIndent(config, "", "  ")
		if err := os.WriteFile(".applad.yaml", data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Created .applad.yaml")
	},
}

func init() {
	initCmd.Flags().String("name", "", "Project name")
	initCmd.Flags().String("url", Getenv("APPLAD_URL", "http://localhost:80"), "API endpoint URL")
}
