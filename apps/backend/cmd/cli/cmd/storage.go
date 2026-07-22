package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var storageCmd = &cobra.Command{
	Use:   "storage",
	Short: "Manage buckets and files",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		LoadConfig()
		RequireProject()
	},
}

var storageListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all buckets",
	Run: func(cmd *cobra.Command, args []string) {
		res, err := GetWithProject(Cfg.URL + "/v1/storage/buckets")
		Must(err)
		PrintList(res, "buckets")
	},
}

var storageFilesCmd = &cobra.Command{
	Use:   "files <bucketId>",
	Short: "List files in a bucket",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		res, err := GetWithProject(Cfg.URL + "/v1/storage/buckets/" + args[0] + "/files")
		Must(err)
		PrintList(res, "files")
	},
}

var storageUploadCmd = &cobra.Command{
	Use:   "upload <bucketId> <filePath>",
	Short: "Upload a file to a bucket",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Uploading %s to bucket %s...\n", args[1], args[0])
		res, err := UploadFile(Cfg.URL+"/v1/storage/buckets/"+args[0]+"/files", args[1])
		Must(err)
		PrettyPrint(res)
	},
}

var storageDeleteCmd = &cobra.Command{
	Use:   "delete <bucketId> <fileId>",
	Short: "Delete a file from a bucket",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		Must(DelWithProject(fmt.Sprintf("%s/v1/storage/buckets/%s/files/%s", Cfg.URL, args[0], args[1])))
		fmt.Println("Deleted.")
	},
}

func init() {
	storageCmd.AddCommand(storageListCmd)
	storageCmd.AddCommand(storageFilesCmd)
	storageCmd.AddCommand(storageUploadCmd)
	storageCmd.AddCommand(storageDeleteCmd)
}
