package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Manage search indexes and documents",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		LoadConfig()
		RequireProject()
	},
}

var searchIndexesCmd = &cobra.Command{
	Use:     "indexes",
	Aliases: []string{"list", "ls"},
	Short:   "List search indexes",
	Run: func(cmd *cobra.Command, args []string) {
		res, err := GetWithProject(Cfg.URL + "/v1/search/indexes")
		Must(err)
		PrintList(res, "indexes")
	},
}

var searchCreateIndexCmd = &cobra.Command{
	Use:   "create-index",
	Short: "Create a search index",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			name = Prompt("Index name: ")
		}
		res, err := PostWithProject(Cfg.URL+"/v1/search/indexes", map[string]string{"name": name})
		Must(err)
		PrettyPrint(res)
	},
}

var searchQueryCmd = &cobra.Command{
	Use:   "query <indexId> <query>",
	Short: "Search an index",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetString("limit")
		res, err := GetWithProject(fmt.Sprintf("%s/v1/search/indexes/%s/search?q=%s&limit=%s", Cfg.URL, args[0], args[1], limit))
		Must(err)
		PrettyPrint(res)
	},
}

var searchDeleteIndexCmd = &cobra.Command{
	Use:   "delete <indexId>",
	Short: "Delete a search index",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		Must(DelWithProject(Cfg.URL + "/v1/search/indexes/" + args[0]))
		fmt.Println("Deleted.")
	},
}

func init() {
	searchCreateIndexCmd.Flags().String("name", "", "Index name")
	searchQueryCmd.Flags().String("limit", "25", "Maximum results")

	searchCmd.AddCommand(searchIndexesCmd)
	searchCmd.AddCommand(searchCreateIndexCmd)
	searchCmd.AddCommand(searchQueryCmd)
	searchCmd.AddCommand(searchDeleteIndexCmd)
}
