package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/vovk4morkovk4/isolate-panel/cli/pkg"
)

var (
	backupFormat       string
	backupOutputDir    string
	backupNoEncryption bool
	backupNoCores      bool
	backupNoCerts      bool
	backupNoWARP       bool
	backupForce        bool
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup management commands",
	Long:  `Commands for creating, restoring, and managing backups.`,
}

var backupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new backup",
	Long:  `Create a new backup of the database and configurations.`,
	RunE:  runBackupCreate,
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all backups",
	Long:  `List all available backups.`,
	RunE:  runBackupList,
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore <backup-id>",
	Short: "Restore from a backup",
	Long:  `Restore the system from a backup. Use --force to skip confirmation.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runBackupRestore,
}

var backupDeleteCmd = &cobra.Command{
	Use:   "delete <backup-id>",
	Short: "Delete a backup",
	Long:  `Delete a backup.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runBackupDelete,
}

var backupDownloadCmd = &cobra.Command{
	Use:   "download <backup-id>",
	Short: "Download a backup",
	Long:  `Download a backup file to the current directory.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runBackupDownload,
}

var backupScheduleCmd = &cobra.Command{
	Use:   "schedule [cron-expression]",
	Short: "Set or get backup schedule",
	Long:  `Set backup schedule using cron expression or get current schedule.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runBackupSchedule,
}

func init() {
	// Global backup flags
	backupCmd.PersistentFlags().StringVar(&backupFormat, "format", "table", "Output format (table, json, csv, quiet)")

	// Create flags
	backupCreateCmd.Flags().BoolVar(&backupNoEncryption, "no-encryption", false, "Disable backup encryption")
	backupCreateCmd.Flags().BoolVar(&backupNoCores, "no-cores", false, "Exclude core configurations")
	backupCreateCmd.Flags().BoolVar(&backupNoCerts, "no-certs", false, "Exclude certificates")
	backupCreateCmd.Flags().BoolVar(&backupNoWARP, "no-warp", false, "Exclude WARP keys")

	// Download flags
	backupDownloadCmd.Flags().StringVarP(&backupOutputDir, "output", "o", ".", "Output directory for downloaded backup")

	// Restore flags
	backupRestoreCmd.Flags().BoolVar(&backupForce, "force", false, "Skip confirmation prompt")

	// Add subcommands
	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	backupCmd.AddCommand(backupDeleteCmd)
	backupCmd.AddCommand(backupDownloadCmd)
	backupCmd.AddCommand(backupScheduleCmd)
}

// BackupCmd returns the backup command
func BackupCmd() *cobra.Command {
	return backupCmd
}

func getAuthHeader() (string, error) {
	config, err := pkg.LoadConfig()
	if err != nil {
		return "", err
	}

	profile, err := config.GetCurrentProfile()
	if err != nil {
		return "", fmt.Errorf("no profile selected. Use 'isolate-panel login' first")
	}

	return "Bearer " + profile.AccessToken, nil
}

func getBaseURL() (string, error) {
	config, err := pkg.LoadConfig()
	if err != nil {
		return "", err
	}

	profile, err := config.GetCurrentProfile()
	if err != nil {
		return "", fmt.Errorf("no profile selected")
	}

	return profile.PanelURL, nil
}

func runBackupCreate(cmd *cobra.Command, args []string) error {
	baseURL, err := getBaseURL()
	if err != nil {
		return err
	}

	authHeader, err := getAuthHeader()
	if err != nil {
		return err
	}

	reqBody := map[string]interface{}{
		"type":               "manual",
		"encryption_enabled": !backupNoEncryption,
		"include_cores":      !backupNoCores,
		"include_certs":      !backupNoCerts,
		"include_warp":       !backupNoWARP,
		"include_geo":        false,
	}

	jsonData, _ := json.Marshal(reqBody)

	client := &http.Client{Timeout: 300 * time.Second}
	httpReq, err := http.NewRequest("POST", baseURL+"/api/backups/create", bytes.NewReader(jsonData))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Authorization", authHeader)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error: %s", resp.Status)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	fmt.Println("✓ Backup created successfully!")
	if data, ok := result["data"].(map[string]interface{}); ok {
		fmt.Printf("  ID: %.0f\n", data["id"])
		fmt.Printf("  Filename: %s\n", data["filename"])
		fmt.Printf("  Status: %s\n", data["status"])
	}
	if msg, ok := result["message"].(string); ok {
		fmt.Printf("  %s\n", msg)
	}

	return nil
}

func runBackupList(cmd *cobra.Command, args []string) error {
	baseURL, err := getBaseURL()
	if err != nil {
		return err
	}

	authHeader, err := getAuthHeader()
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", baseURL+"/api/backups", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", authHeader)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("API error: %s", resp.Status)
	}

	var result struct {
		Data []map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	// Output based on format
	format := pkg.ParseFormat(backupFormat)

	switch format {
	case pkg.FormatJSON:
		return pkg.WriteJSON(cmd.OutOrStdout(), result.Data)
	case pkg.FormatCSV:
		return outputBackupsCSV(cmd.OutOrStdout(), result.Data)
	case pkg.FormatQuiet:
		return outputBackupsQuiet(cmd.OutOrStdout(), result.Data)
	default:
		return outputBackupsTable(cmd.OutOrStdout(), result.Data)
	}
}

func outputBackupsTable(out io.Writer, backups []map[string]interface{}) error {
	tw := pkg.NewTableWriter(out)
	tw.AddRow("ID", "FILENAME", "SIZE (MB)", "STATUS", "CREATED")

	for _, b := range backups {
		id := fmt.Sprintf("%.0f", b["id"].(float64))
		filename := b["filename"].(string)
		size := ""
		if s, ok := b["file_size_bytes"].(float64); ok {
			size = fmt.Sprintf("%.2f", s/1024/1024)
		}
		status := b["status"].(string)
		created := ""
		if c, ok := b["created_at"].(string); ok {
			created = c[:19]
		}

		tw.AddRow(id, filename, size, status, created)
	}

	return tw.Render()
}

func outputBackupsCSV(out io.Writer, backups []map[string]interface{}) error {
	headers := []string{"id", "filename", "file_size_bytes", "status", "created_at"}
	rows := make([][]string, len(backups))

	for i, b := range backups {
		id := fmt.Sprintf("%.0f", b["id"].(float64))
		filename := b["filename"].(string)
		size := ""
		if s, ok := b["file_size_bytes"].(float64); ok {
			size = fmt.Sprintf("%.0f", s)
		}
		status := b["status"].(string)
		created := ""
		if c, ok := b["created_at"].(string); ok {
			created = c
		}

		rows[i] = []string{id, filename, size, status, created}
	}

	return pkg.WriteCSV(out, headers, rows)
}

func outputBackupsQuiet(out io.Writer, backups []map[string]interface{}) error {
	values := make([]string, len(backups))
	for i, b := range backups {
		values[i] = b["filename"].(string)
	}
	return pkg.WriteQuiet(out, values)
}

func runBackupRestore(cmd *cobra.Command, args []string) error {
	if !backupForce {
		fmt.Print("⚠️  WARNING: Restore will overwrite all data. Continue? (yes/no): ")
		var answer string
		fmt.Scanln(&answer)
		if answer != "yes" {
			fmt.Println("Restore cancelled")
			return nil
		}
	}

	backupID := args[0]
	baseURL, err := getBaseURL()
	if err != nil {
		return err
	}

	authHeader, err := getAuthHeader()
	if err != nil {
		return err
	}

	reqBody := map[string]bool{"force": true}
	jsonData, _ := json.Marshal(reqBody)

	client := &http.Client{Timeout: 300 * time.Second}
	httpReq, err := http.NewRequest("POST", baseURL+"/api/backups/"+backupID+"/restore", bytes.NewReader(jsonData))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Authorization", authHeader)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error: %s", resp.Status)
	}

	fmt.Println("✓ Restore operation started!")
	fmt.Println("  Note: Restore runs in background. Check backup status for progress.")

	return nil
}

func runBackupDelete(cmd *cobra.Command, args []string) error {
	backupID := args[0]
	baseURL, err := getBaseURL()
	if err != nil {
		return err
	}

	authHeader, err := getAuthHeader()
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("DELETE", baseURL+"/api/backups/"+backupID, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", authHeader)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error: %s", resp.Status)
	}

	fmt.Printf("✓ Backup %s deleted\n", backupID)
	return nil
}

func runBackupDownload(cmd *cobra.Command, args []string) error {
	backupID := args[0]
	baseURL, err := getBaseURL()
	if err != nil {
		return err
	}

	authHeader, err := getAuthHeader()
	if err != nil {
		return err
	}

	// First get backup info
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", baseURL+"/api/backups/"+backupID, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", authHeader)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	filename := result.Data["filename"].(string)
	outputPath := filepath.Join(backupOutputDir, filename)

	// Download file
	req, err = http.NewRequest("GET", baseURL+"/api/backups/"+backupID+"/download", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", authHeader)

	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := os.WriteFile(outputPath, data, 0600); err != nil {
		return err
	}

	fmt.Printf("✓ Backup downloaded: %s\n", outputPath)
	return nil
}

func runBackupSchedule(cmd *cobra.Command, args []string) error {
	baseURL, err := getBaseURL()
	if err != nil {
		return err
	}

	authHeader, err := getAuthHeader()
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 30 * time.Second}

	if len(args) == 0 {
		// Get current schedule
		req, err := http.NewRequest("GET", baseURL+"/api/backups/schedule", nil)
		if err != nil {
			return err
		}

		req.Header.Set("Authorization", authHeader)

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		var result struct {
			Data map[string]interface{} `json:"data"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}

		schedule := ""
		if s, ok := result.Data["schedule"].(string); ok {
			schedule = s
		}

		nextRun := ""
		if n, ok := result.Data["next_run"].(string); ok && n != "" {
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
	reqBody := map[string]string{"cron": cronExpr}
	jsonData, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", baseURL+"/api/backups/schedule", bytes.NewReader(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error: %s", resp.Status)
	}

	fmt.Printf("✓ Backup schedule updated: %s\n", cronExpr)
	return nil
}
