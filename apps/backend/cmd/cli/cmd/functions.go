package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var functionsCmd = &cobra.Command{
	Use:     "functions",
	Aliases: []string{"fn"},
	Short:   "Manage and invoke functions",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		LoadConfig()
		RequireProject()
	},
}

var fnListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all functions",
	Run: func(cmd *cobra.Command, args []string) {
		res, err := GetWithProject(Cfg.URL + "/v1/functions")
		Must(err)
		PrintList(res, "functions")
	},
}

var fnInvokeCmd = &cobra.Command{
	Use:     "invoke <functionId>",
	Aliases: []string{"exec"},
	Short:   "Invoke a function",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		data, _ := cmd.Flags().GetString("data")
		body := map[string]interface{}{}
		if data != "" {
			json.Unmarshal([]byte(data), &body) //nolint:errcheck
		}
		res, err := PostWithProject(Cfg.URL+"/v1/functions/"+args[0]+"/executions", body)
		Must(err)
		PrettyPrint(res)
	},
}

var fnLogsCmd = &cobra.Command{
	Use:   "logs <functionId>",
	Short: "List function executions",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		res, err := GetWithProject(Cfg.URL + "/v1/functions/" + args[0] + "/executions")
		Must(err)
		PrintList(res, "executions")
	},
}

var fnPushCmd = &cobra.Command{
	Use:   "push <name>",
	Short: "Create or update a function from a local source file or directory",
	Long: "Reads inline source from a local file (or the entrypoint file inside a\n" +
		"directory), then creates the function if it does not exist or updates it\n" +
		"if a function with the same name already exists.",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		runtime, _ := cmd.Flags().GetString("runtime")
		source, _ := cmd.Flags().GetString("source")
		entrypoint, _ := cmd.Flags().GetString("entrypoint")
		timeout, _ := cmd.Flags().GetInt("timeout")
		envPairs, _ := cmd.Flags().GetStringArray("env")

		if runtime == "" {
			fmt.Fprintln(os.Stderr, "error: --runtime is required (see 'applad functions runtimes')")
			os.Exit(1)
		}
		if source == "" {
			fmt.Fprintln(os.Stderr, "error: --source is required (path to a file or directory)")
			os.Exit(1)
		}

		code, err := readFunctionSource(source, runtime)
		Must(err)

		envVars, err := parseEnvPairs(envPairs)
		Must(err)

		body := map[string]interface{}{
			"name":       name,
			"runtime":    runtime,
			"sourceType": "inline",
			"source":     code,
			"envVars":    envVars,
		}
		if entrypoint != "" {
			body["entrypoint"] = entrypoint
		}
		if timeout > 0 {
			body["timeout"] = timeout
		}

		// Create-or-update: look for an existing function with the same name.
		listRes, err := GetWithProject(Cfg.URL + "/v1/functions")
		Must(err)
		existing := FindByName(listRes, "functions", name)

		var res map[string]interface{}
		if existing != nil {
			id, _ := existing["$id"].(string)
			fmt.Printf("Updating function %q (%s)...\n", name, id)
			res, err = PutWithProject(Cfg.URL+"/v1/functions/"+id, body)
		} else {
			fmt.Printf("Creating function %q...\n", name)
			res, err = PostWithProject(Cfg.URL+"/v1/functions", body)
		}
		Must(err)
		PrettyPrint(res)
	},
}

var fnRuntimesCmd = &cobra.Command{
	Use:   "runtimes",
	Short: "List supported function runtimes",
	Run: func(cmd *cobra.Command, args []string) {
		res, err := GetWithProject(Cfg.URL + "/v1/functions/runtimes")
		Must(err)
		PrettyPrint(res)
	},
}

// readFunctionSource returns the inline source for a function. A file path is
// read directly; a directory is resolved to the single source file the runtime
// expects (index.js, main.py, ...), falling back to the only regular file it
// contains.
func readFunctionSource(path, runtime string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	// Directory: prefer the conventional entrypoint filename for the runtime.
	if fname := sourceFilenameForRuntime(runtime); fname != "" {
		candidate := filepath.Join(path, fname)
		if _, err := os.Stat(candidate); err == nil {
			data, err := os.ReadFile(candidate)
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
	}

	// Otherwise, if the directory holds exactly one regular file, use it.
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e.Name())
		}
	}
	if len(files) == 1 {
		data, err := os.ReadFile(filepath.Join(path, files[0]))
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	return "", fmt.Errorf("directory %q holds multiple files; point --source at a single source file", path)
}

// sourceFilenameForRuntime mirrors the backend runtime's inline source filename
// so that pushing a directory selects the file the build will expect.
func sourceFilenameForRuntime(runtime string) string {
	switch {
	case strings.HasPrefix(runtime, "node"):
		return "index.js"
	case strings.HasPrefix(runtime, "python"):
		return "main.py"
	case strings.HasPrefix(runtime, "go"):
		return "main.go"
	case strings.HasPrefix(runtime, "dart"):
		return "main.dart"
	case strings.HasPrefix(runtime, "bun"):
		return "index.ts"
	default:
		return ""
	}
}

// parseEnvPairs turns repeated KEY=VALUE flags into a map.
func parseEnvPairs(pairs []string) (map[string]string, error) {
	out := map[string]string{}
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("invalid --env %q, expected KEY=VALUE", p)
		}
		out[k] = v
	}
	return out, nil
}

var fnDeleteCmd = &cobra.Command{
	Use:   "delete <functionId>",
	Short: "Delete a function",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		Must(DelWithProject(Cfg.URL + "/v1/functions/" + args[0]))
		fmt.Println("Deleted.")
	},
}

func init() {
	fnInvokeCmd.Flags().String("data", "", "JSON data to pass to the function")

	fnPushCmd.Flags().String("runtime", "", "Runtime to build with, e.g. node20, python311 (required)")
	fnPushCmd.Flags().String("source", "", "Path to a source file or directory (required)")
	fnPushCmd.Flags().String("entrypoint", "", "Handler entrypoint (defaults to the server value)")
	fnPushCmd.Flags().Int("timeout", 0, "Execution timeout in seconds")
	fnPushCmd.Flags().StringArray("env", nil, "Environment variable as KEY=VALUE (repeatable)")

	functionsCmd.AddCommand(fnListCmd)
	functionsCmd.AddCommand(fnInvokeCmd)
	functionsCmd.AddCommand(fnPushCmd)
	functionsCmd.AddCommand(fnRuntimesCmd)
	functionsCmd.AddCommand(fnLogsCmd)
	functionsCmd.AddCommand(fnDeleteCmd)
}
