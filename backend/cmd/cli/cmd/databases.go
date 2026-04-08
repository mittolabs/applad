package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var databasesCmd = &cobra.Command{
	Use:     "databases",
	Aliases: []string{"db"},
	Short:   "Manage databases and tables",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		LoadConfig()
		RequireProject()
	},
}

var dbListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all databases",
	Run: func(cmd *cobra.Command, args []string) {
		res, err := GetWithProject(Cfg.URL + "/v1/databases")
		Must(err)
		PrintList(res, "databases")
	},
}

var dbCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a database",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			name = Prompt("Database name: ")
		}
		res, err := PostWithProject(Cfg.URL+"/v1/databases", map[string]string{"name": name})
		Must(err)
		PrettyPrint(res)
	},
}

var dbTablesCmd = &cobra.Command{
	Use:   "tables <databaseId>",
	Short: "List tables in a database",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		res, err := GetWithProject(Cfg.URL + "/v1/databases/" + args[0] + "/collections")
		Must(err)
		PrintList(res, "collections")
	},
}

var dbCreateTableCmd = &cobra.Command{
	Use:   "create-table <databaseId>",
	Short: "Create a table in a database",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			name = Prompt("Table name: ")
		}
		res, err := PostWithProject(Cfg.URL+"/v1/databases/"+args[0]+"/collections", map[string]string{"name": name})
		Must(err)
		PrettyPrint(res)
	},
}

var dbQueryCmd = &cobra.Command{
	Use:   "query <databaseId> <tableId>",
	Short: "Query rows in a table",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetString("limit")
		url := fmt.Sprintf("%s/v1/databases/%s/collections/%s/documents?limit=%s", Cfg.URL, args[0], args[1], limit)
		res, err := GetWithProject(url)
		Must(err)
		PrintList(res, "documents")
	},
}

var dbDeleteCmd = &cobra.Command{
	Use:   "delete <databaseId>",
	Short: "Delete a database",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		Must(DelWithProject(Cfg.URL + "/v1/databases/" + args[0]))
		fmt.Println("Deleted.")
	},
}

func init() {
	dbCreateCmd.Flags().String("name", "", "Database name")
	dbCreateTableCmd.Flags().String("name", "", "Table name")
	dbQueryCmd.Flags().String("limit", "25", "Number of rows to return")

	databasesCmd.AddCommand(dbListCmd)
	databasesCmd.AddCommand(dbCreateCmd)
	databasesCmd.AddCommand(dbTablesCmd)
	databasesCmd.AddCommand(dbCreateTableCmd)
	databasesCmd.AddCommand(dbQueryCmd)
	databasesCmd.AddCommand(dbDeleteCmd)
}
