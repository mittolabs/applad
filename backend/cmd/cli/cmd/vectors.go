package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var vectorsCmd = &cobra.Command{
	Use:   "vectors",
	Short: "Manage vector indexes and embeddings",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		LoadConfig()
		RequireProject()
	},
}

var vectorsListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List vector indexes",
	Run: func(cmd *cobra.Command, args []string) {
		res, err := GetWithProject(Cfg.URL + "/v1/vectors/indexes")
		Must(err)
		PrintList(res, "indexes")
	},
}

var vectorsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a vector index",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		dimensions, _ := cmd.Flags().GetInt("dimensions")
		metric, _ := cmd.Flags().GetString("metric")
		if name == "" {
			name = Prompt("Index name: ")
		}
		body := map[string]interface{}{
			"name":       name,
			"dimensions": dimensions,
			"metric":     metric,
		}
		res, err := PostWithProject(Cfg.URL+"/v1/vectors/indexes", body)
		Must(err)
		PrettyPrint(res)
	},
}

var vectorsQueryCmd = &cobra.Command{
	Use:   "query <indexId>",
	Short: "Query similar vectors",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		vectorStr, _ := cmd.Flags().GetString("vector")
		topK, _ := cmd.Flags().GetInt("top-k")
		var vector []float64
		json.Unmarshal([]byte(vectorStr), &vector) //nolint:errcheck
		body := map[string]interface{}{
			"vector": vector,
			"topK":   topK,
		}
		res, err := PostWithProject(Cfg.URL+"/v1/vectors/indexes/"+args[0]+"/query", body)
		Must(err)
		PrettyPrint(res)
	},
}

var vectorsDeleteCmd = &cobra.Command{
	Use:   "delete <indexId>",
	Short: "Delete a vector index",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		Must(DelWithProject(Cfg.URL + "/v1/vectors/indexes/" + args[0]))
		fmt.Println("Deleted.")
	},
}

func init() {
	vectorsCreateCmd.Flags().String("name", "", "Index name")
	vectorsCreateCmd.Flags().Int("dimensions", 1536, "Vector dimensions")
	vectorsCreateCmd.Flags().String("metric", "cosine", "Distance metric (cosine, euclidean, dot)")
	vectorsQueryCmd.Flags().String("vector", "[]", "Query vector as JSON array")
	vectorsQueryCmd.Flags().Int("top-k", 10, "Number of results")

	vectorsCmd.AddCommand(vectorsListCmd)
	vectorsCmd.AddCommand(vectorsCreateCmd)
	vectorsCmd.AddCommand(vectorsQueryCmd)
	vectorsCmd.AddCommand(vectorsDeleteCmd)
}
