package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config holds CLI authentication state persisted to ~/.applad/config.json.
type Config struct {
	URL          string `json:"url"`
	ProjectID    string `json:"projectId"`
	APIKey       string `json:"apiKey"`
	ConsoleToken string `json:"consoleToken"`
}

var (
	Cfg        Config
	HTTPClient = &http.Client{Timeout: 30 * time.Second}
)

// LoadConfig reads the CLI config from disk and env vars.
func LoadConfig() {
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return
	}
	json.Unmarshal(data, &Cfg) //nolint:errcheck
	if Cfg.URL == "" {
		Cfg.URL = Getenv("APPLAD_URL", "http://localhost:80")
	}
	if Cfg.ProjectID == "" {
		Cfg.ProjectID = os.Getenv("APPLAD_PROJECT")
	}
	if Cfg.APIKey == "" {
		Cfg.APIKey = os.Getenv("APPLAD_KEY")
	}
}

// SaveConfig persists the CLI config to disk.
func SaveConfig() {
	dir := filepath.Dir(ConfigPath())
	os.MkdirAll(dir, 0700) //nolint:errcheck
	data, _ := json.MarshalIndent(Cfg, "", "  ")
	os.WriteFile(ConfigPath(), data, 0600) //nolint:errcheck
}

// ConfigPath returns ~/.applad/config.json.
func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".applad", "config.json")
}

// RequireProject exits if no project is set.
func RequireProject() {
	if Cfg.ProjectID == "" {
		fmt.Fprintln(os.Stderr, "error: no project selected. Run 'applad projects use <id>'")
		os.Exit(1)
	}
}

// ── HTTP helpers ──────────────────────────────────────────────────────────────

func Get(url, token string) (map[string]interface{}, error) {
	return DoRequest(http.MethodGet, url, token, "", nil)
}

func GetWithProject(url string) (map[string]interface{}, error) {
	return DoRequest(http.MethodGet, url, Cfg.ConsoleToken, Cfg.ProjectID, nil)
}

func Post(url, token string, body interface{}) (map[string]interface{}, error) {
	return DoRequest(http.MethodPost, url, token, "", body)
}

func PostWithProject(url string, body interface{}) (map[string]interface{}, error) {
	return DoRequest(http.MethodPost, url, Cfg.ConsoleToken, Cfg.ProjectID, body)
}

func PatchWithProject(url string, body interface{}) (map[string]interface{}, error) {
	return DoRequest(http.MethodPatch, url, Cfg.ConsoleToken, Cfg.ProjectID, body)
}

func PutWithProject(url string, body interface{}) (map[string]interface{}, error) {
	return DoRequest(http.MethodPut, url, Cfg.ConsoleToken, Cfg.ProjectID, body)
}

func Del(url, token string) error {
	_, err := DoRequest(http.MethodDelete, url, token, "", nil)
	return err
}

func DelWithProject(url string) error {
	_, err := DoRequest(http.MethodDelete, url, Cfg.ConsoleToken, Cfg.ProjectID, nil)
	return err
}

func DoRequest(method, url, token, projectID string, body interface{}) (map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if projectID != "" {
		req.Header.Set("X-Applad-Project", projectID)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return map[string]interface{}{}, nil
	}
	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if resp.StatusCode >= 400 {
		msg, _ := out["message"].(string)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, msg)
	}
	return out, nil
}

func APIList(url, token string) ([]map[string]interface{}, error) {
	res, err := Get(url, token)
	if err != nil {
		return nil, err
	}
	for _, key := range []string{"projects", "users", "data"} {
		if v, ok := res[key]; ok {
			if arr, ok := v.([]interface{}); ok {
				out := make([]map[string]interface{}, 0, len(arr))
				for _, item := range arr {
					if m, ok := item.(map[string]interface{}); ok {
						out = append(out, m)
					}
				}
				return out, nil
			}
		}
	}
	return nil, nil
}

func UploadFile(url, filePath string) (map[string]interface{}, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, err
	}
	mw.Close()

	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+Cfg.ConsoleToken)
	if Cfg.ProjectID != "" {
		req.Header.Set("X-Applad-Project", Cfg.ProjectID)
	}
	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		msg, _ := out["message"].(string)
		return nil, fmt.Errorf("upload error %d: %s", resp.StatusCode, msg)
	}
	return out, nil
}

// ── Print helpers ─────────────────────────────────────────────────────────────

func PrettyPrint(v interface{}) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}

func PrintList(res map[string]interface{}, key string) {
	if items, ok := res[key].([]interface{}); ok {
		fmt.Printf("Total: %v\n", res["total"])
		for _, item := range items {
			if m, ok := item.(map[string]interface{}); ok {
				id, _ := m["$id"].(string)
				name, _ := m["name"].(string)
				status, _ := m["status"].(string)
				line := id
				if name != "" {
					line += " | " + name
				}
				if status != "" {
					line += " | " + status
				}
				fmt.Println(" -", line)
			}
		}
	} else {
		PrettyPrint(res)
	}
}

func FindByName(res map[string]interface{}, key, name string) map[string]interface{} {
	if res == nil {
		return nil
	}
	items, ok := res[key].([]interface{})
	if !ok {
		return nil
	}
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if n, _ := m["name"].(string); n == name {
			return m
		}
	}
	return nil
}

func Getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Prompt(label string) string {
	fmt.Print(label)
	var s string
	fmt.Scanln(&s) //nolint:errcheck
	return strings.TrimSpace(s)
}

func Must(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
