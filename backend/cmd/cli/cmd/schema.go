package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type schemaFile struct {
	Databases []schemaDatabase `json:"databases"`
}
type schemaDatabase struct {
	Name   string        `json:"name"`
	Tables []schemaTable `json:"tables"`
}
type schemaTable struct {
	Name    string         `json:"name"`
	Columns []schemaColumn `json:"columns"`
}

type schemaColumn struct {
	Key      string `json:"key"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Push schema migrations",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		LoadConfig()
		RequireProject()
	},
}

var schemaPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push schema from a JSON file",
	Run: func(cmd *cobra.Command, args []string) {
		file, _ := cmd.Flags().GetString("file")
		data, err := os.ReadFile(file)
		Must(err)
		var schema schemaFile
		if err := json.Unmarshal(data, &schema); err != nil {
			fmt.Fprintf(os.Stderr, "error: invalid JSON in %s: %v\n", file, err)
			os.Exit(1)
		}
		fmt.Printf("Pushing schema from %s...\n", file)
		base := Cfg.URL + "/v1/databases"
		for _, sdb := range schema.Databases {
			dbRes, err := PostWithProject(base, map[string]string{"name": sdb.Name})
			if err != nil {
				listRes, lerr := GetWithProject(base)
				Must(lerr)
				dbRes = FindByName(listRes, "databases", sdb.Name)
				if dbRes == nil {
					fmt.Fprintf(os.Stderr, "error: could not create or find database %q\n", sdb.Name)
					os.Exit(1)
				}
			}
			dbID, _ := dbRes["$id"].(string)
			fmt.Printf("  database %q (%s)\n", sdb.Name, dbID)

			for _, st := range sdb.Tables {
				tableBase := base + "/" + dbID + "/tables"
				tRes, terr := PostWithProject(tableBase, map[string]string{"name": st.Name})
				if terr != nil {
					listRes2, _ := GetWithProject(tableBase)
					tRes = FindByName(listRes2, "tables", st.Name)
				}
				tableID, _ := tRes["$id"].(string)
				fmt.Printf("    table %q (%s)\n", st.Name, tableID)

				columnBase := tableBase + "/" + tableID + "/columns"
				for _, column := range st.Columns {
					_, columnErr := PostWithProject(columnBase+"/"+column.Type, map[string]interface{}{
						"key":      column.Key,
						"required": column.Required,
					})
					if columnErr != nil {
						fmt.Printf("      column %q: %v (may already exist)\n", column.Key, columnErr)
					} else {
						fmt.Printf("      column %q (%s)\n", column.Key, column.Type)
					}
				}
			}
		}
		fmt.Println("Schema push complete.")
	},
}

var schemaDumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Dump current schema to stdout",
	Run: func(cmd *cobra.Command, args []string) {
		res, err := GetWithProject(Cfg.URL + "/v1/databases")
		Must(err)
		PrettyPrint(res)
	},
}

func init() {
	schemaPushCmd.Flags().String("file", "schema.json", "Path to schema JSON file")

	schemaCmd.AddCommand(schemaPushCmd)
	schemaCmd.AddCommand(schemaDumpCmd)
}
