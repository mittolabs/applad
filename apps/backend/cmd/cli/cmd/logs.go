package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Tail audit logs",
	PreRun: func(cmd *cobra.Command, args []string) {
		RequireProject()
	},
	Run: func(cmd *cobra.Command, args []string) {
		resource, _ := cmd.Flags().GetString("resource")
		fmt.Printf("Tailing %s logs (Ctrl+C to stop)...\n", resource)

		seen := map[string]bool{}
		for {
			res, err := GetWithProject(Cfg.URL + "/v1/audit?limit=20")
			if err == nil {
				if logs, ok := res["logs"].([]interface{}); ok {
					for _, l := range logs {
						entry, ok := l.(map[string]interface{})
						if !ok {
							continue
						}
						id, _ := entry["$id"].(string)
						if seen[id] {
							continue
						}
						seen[id] = true
						ts, _ := entry["$createdAt"].(string)
						action, _ := entry["action"].(string)
						path, _ := entry["path"].(string)
						status, _ := entry["statusCode"].(float64)
						fmt.Printf("[%s] %s %s → %d\n", ts, action, path, int(status))
					}
				}
			}
			time.Sleep(2 * time.Second)
		}
	},
}

func init() {
	logsCmd.Flags().String("resource", "audit", "Log resource type")
}
