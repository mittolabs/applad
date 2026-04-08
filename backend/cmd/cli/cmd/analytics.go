package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var analyticsCmd = &cobra.Command{
	Use:   "analytics",
	Short: "Manage analytics events and queries",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		LoadConfig()
		RequireProject()
	},
}

var analyticsEventsCmd = &cobra.Command{
	Use:     "events",
	Aliases: []string{"list", "ls"},
	Short:   "List recent analytics events",
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetString("limit")
		res, err := GetWithProject(fmt.Sprintf("%s/v1/analytics/events?limit=%s", Cfg.URL, limit))
		Must(err)
		PrintList(res, "events")
	},
}

var analyticsCountCmd = &cobra.Command{
	Use:   "count <eventName>",
	Short: "Get event count",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		res, err := GetWithProject(fmt.Sprintf("%s/v1/analytics/events/%s/count", Cfg.URL, args[0]))
		Must(err)
		PrettyPrint(res)
	},
}

var analyticsDAUCmd = &cobra.Command{
	Use:   "dau",
	Short: "Get daily active users",
	Run: func(cmd *cobra.Command, args []string) {
		days, _ := cmd.Flags().GetString("days")
		res, err := GetWithProject(fmt.Sprintf("%s/v1/analytics/dau?days=%s", Cfg.URL, days))
		Must(err)
		PrettyPrint(res)
	},
}

var analyticsFunnelCmd = &cobra.Command{
	Use:   "funnel",
	Short: "Run a funnel analysis",
	Run: func(cmd *cobra.Command, args []string) {
		steps, _ := cmd.Flags().GetStringSlice("steps")
		res, err := PostWithProject(Cfg.URL+"/v1/analytics/funnels", map[string]interface{}{
			"steps": steps,
		})
		Must(err)
		PrettyPrint(res)
	},
}

func init() {
	analyticsEventsCmd.Flags().String("limit", "25", "Number of events to return")
	analyticsDAUCmd.Flags().String("days", "30", "Number of days")
	analyticsFunnelCmd.Flags().StringSlice("steps", nil, "Funnel step event names")

	analyticsCmd.AddCommand(analyticsEventsCmd)
	analyticsCmd.AddCommand(analyticsCountCmd)
	analyticsCmd.AddCommand(analyticsDAUCmd)
	analyticsCmd.AddCommand(analyticsFunnelCmd)
}
