package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var (
	backupCmd = &cobra.Command{
		Use:   "backup",
		Short: "Backup management commands",
		Long:  `Commands for creating, restoring, and managing backups`,
	}

	backupCreateCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a new backup",
		Long:  `Create a new backup of the database and configurations`,
		RunE:  runBackupCreate,
	}

	backupListCmd = &cobra.Command{
		Use:   "list",
		Short: "List all backups",
		Long:  `List all available backups`,
		RunE:  runBackupList,
	}

	backupRestoreCmd = &cobra.Command{
		Use:   "restore <backup-id>",
		Short: "Restore from a backup",
		Long:  `Restore the system from a backup. Use --force to skip confirmation`,
		Args:  cobra.ExactArgs(1),
		RunE:  runBackupRestore,
	}

	backupDownloadCmd = &cobra.Command{
		Use:   "download <backup-id>",
		Short: "Download a backup",
		Long:  `Download a backup file to the current directory`,
		Args:  cobra.ExactArgs(1),
		RunE:  runBackupDownload,
	}

	backupScheduleCmd = &cobra.Command{
		Use:   "schedule [cron-expression]",
		Short: "Set or get backup schedule",
		Long:  `Set backup schedule using cron expression or get current schedule. Example: "0 3 * * *" for daily at 3 AM`,
		RunE:  runBackupSchedule,
	}

	// Flags
	backupForce        bool
	backupOutputDir    string
	backupNoEncryption bool
	backupNoCores      bool
	backupNoCerts      bool
	backupNoWARP       bool
)

type BackupRequest struct {
	Type              string `json:"type"`
	EncryptionEnabled bool   `json:"encryption_enabled"`
	IncludeCores      bool   `json:"include_cores"`
	IncludeCerts      bool   `json:"include_certs"`
	IncludeWARP       bool   `json:"include_warp"`
	IncludeGeo        bool   `json:"include_geo"`
}

type Backup struct {
	ID            uint       `json:"id"`
	Filename      string     `json:"filename"`
	FilePath      string     `json:"file_path"`
	FileSizeBytes int64      `json:"file_size_bytes"`
	Status        string     `json:"status"`
	BackupType    string     `json:"backup_type"`
	CreatedAt     time.Time  `json:"created_at"`
	CompletedAt   *time.Time `json:"completed_at"`
}

type APIResponse struct {
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
	Error   string      `json:"error"`
}

func init() {
	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	backupCmd.AddCommand(backupDownloadCmd)
	backupCmd.AddCommand(backupScheduleCmd)

	// Flags
	backupRestoreCmd.Flags().BoolVar(&backupForce, "force", false, "Skip confirmation prompt")
	backupDownloadCmd.Flags().StringVarP(&backupOutputDir, "output", "o", ".", "Output directory for downloaded backup")
	backupCreateCmd.Flags().BoolVar(&backupNoEncryption, "no-encryption", false, "Disable backup encryption")
	backupCreateCmd.Flags().BoolVar(&backupNoCores, "no-cores", false, "Exclude core configurations")
	backupCreateCmd.Flags().BoolVar(&backupNoCerts, "no-certs", false, "Exclude certificates")
	backupCreateCmd.Flags().BoolVar(&backupNoWARP, "no-warp", false, "Exclude WARP keys")
}

func getBaseURL() string {
	url, _ := rootCmd.PersistentFlags().GetString("url")
	return url
}

func getToken() string {
	token, _ := rootCmd.PersistentFlags().GetString("token")
	return token
}

func makeRequest(method, path string, body interface{}) ([]byte, error) {
	url := getBaseURL() + "/api" + path
	token := getToken()

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")

	// Skip TLS verification for self-signed certs
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 300 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		var apiResp APIResponse
		if err := json.Unmarshal(respBody, &apiResp); err != nil {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, apiResp.Error)
	}

	return respBody, nil
}

func runBackupCreate(cmd *cobra.Command, args []string) error {
	req := BackupRequest{
		Type:              "manual",
		EncryptionEnabled: !backupNoEncryption,
		IncludeCores:      !backupNoCores,
		IncludeCerts:      !backupNoCerts,
		IncludeWARP:       !backupNoWARP,
		IncludeGeo:        false,
	}

	fmt.Println("Creating backup...")
	respBody, err := makeRequest("POST", "/backups/create", req)
	if err != nil {
		return err
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return err
	}

	backupJSON, _ := json.MarshalIndent(apiResp.Data, "", "  ")
	fmt.Printf("Backup created successfully!\n%s\n", string(backupJSON))
	fmt.Printf("Message: %s\n", apiResp.Message)

	return nil
}

func runBackupList(cmd *cobra.Command, args []string) error {
	fmt.Println("Fetching backups...")
	respBody, err := makeRequest("GET", "/backups", nil)
	if err != nil {
		return err
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return err
	}

	backups, ok := apiResp.Data.([]interface{})
	if !ok {
		return fmt.Errorf("invalid response format")
	}

	if len(backups) == 0 {
		fmt.Println("No backups found")
		return nil
	}

	fmt.Printf("\n%-6s %-40s %-12s %-10s %-25s\n", "ID", "Filename", "Size (MB)", "Status", "Created At")
	fmt.Println("---------------------------------------------------------------------------------------------")

	for _, b := range backups {
		backup := b.(map[string]interface{})
		id := uint(backup["id"].(float64))
		filename := backup["filename"].(string)
		sizeBytes := int64(backup["file_size_bytes"].(float64))
		status := backup["status"].(string)
		createdAt := backup["created_at"].(string)

		sizeMB := float64(sizeBytes) / 1024 / 1024
		fmt.Printf("%-6d %-40s %-12.2f %-10s %-25s\n", id, filename, sizeMB, status, createdAt)
	}

	return nil
}

func runBackupRestore(cmd *cobra.Command, args []string) error {
	backupID := args[0]

	if !backupForce {
		fmt.Print("⚠️  WARNING: Restore will overwrite all data. Continue? (yes/no): ")
		var answer string
		fmt.Scanln(&answer)
		if answer != "yes" {
			fmt.Println("Restore cancelled")
			return nil
		}
	}

	fmt.Printf("Restoring from backup %s...\n", backupID)
	req := map[string]bool{"force": true}
	respBody, err := makeRequest("POST", "/backups/"+backupID+"/restore", req)
	if err != nil {
		return err
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return err
	}

	fmt.Printf("Restore operation started!\nMessage: %s\n", apiResp.Message)
	fmt.Println("Note: Restore runs in background. Check backup status for progress.")

	return nil
}

func runBackupDownload(cmd *cobra.Command, args []string) error {
	backupID := args[0]

	fmt.Printf("Downloading backup %s...\n", backupID)

	// First get backup info to get filename
	respBody, err := makeRequest("GET", "/backups/"+backupID, nil)
	if err != nil {
		return err
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return err
	}

	backupData, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid response format")
	}

	filename := backupData["filename"].(string)
	outputPath := filepath.Join(backupOutputDir, filename)

	// Download file
	respBody, err = makeRequest("GET", "/backups/"+backupID+"/download", nil)
	if err != nil {
		return err
	}

	if err := os.WriteFile(outputPath, respBody, 0600); err != nil {
		return err
	}

	fmt.Printf("Backup downloaded successfully: %s\n", outputPath)
	return nil
}

func runBackupSchedule(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		// Get current schedule
		respBody, err := makeRequest("GET", "/backups/schedule", nil)
		if err != nil {
			return err
		}

		var apiResp APIResponse
		if err := json.Unmarshal(respBody, &apiResp); err != nil {
			return err
		}

		data, ok := apiResp.Data.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid response format")
		}

		schedule := ""
		if s, ok := data["schedule"].(string); ok {
			schedule = s
		}

		nextRun := ""
		if n, ok := data["next_run"].(string); ok && n != "" {
			nextRun = n
		}

		if schedule == "" {
			fmt.Println("No backup schedule configured")
		} else {
			fmt.Printf("Current schedule: %s\n", schedule)
			if nextRun != "" {
				fmt.Printf("Next run: %s\n", nextRun)
			}
		}
		return nil
	}

	// Set schedule
	cronExpr := args[0]
	req := map[string]string{"cron": cronExpr}

	respBody, err := makeRequest("POST", "/backups/schedule", req)
	if err != nil {
		return err
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return err
	}

	fmt.Printf("Backup schedule updated: %s\n", cronExpr)
	fmt.Printf("Message: %s\n", apiResp.Message)

	return nil
}
