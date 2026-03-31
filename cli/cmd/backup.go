package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/isolate-project/isolate-panel/cli/pkg"
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

func runBackupCreate(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
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

	var result struct {
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}

	if err := client.Post("/api/backups/create", reqBody, &result); err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), "✓ Backup created successfully!")
	if result.Data != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "  ID: %.0f\n", result.Data["id"])
		fmt.Fprintf(cmd.OutOrStdout(), "  Filename: %s\n", result.Data["filename"])
		fmt.Fprintf(cmd.OutOrStdout(), "  Status: %s\n", result.Data["status"])
	}
	if result.Message != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", result.Message)
	}

	return nil
}

func runBackupList(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	var result struct {
		Data []map[string]interface{} `json:"data"`
	}

	if err := client.Get("/api/backups", &result); err != nil {
		return err
	}

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
		id := ""
		if v, ok := b["id"].(float64); ok {
			id = fmt.Sprintf("%.0f", v)
		} else if v, ok := b["id"].(string); ok {
			id = v
		}

		filename := ""
		if f, ok := b["filename"].(string); ok {
			filename = f
		}
		
		size := ""
		if s, ok := b["file_size_bytes"].(float64); ok {
			size = fmt.Sprintf("%.2f", s/1024/1024)
		}
		
		status := ""
		if s, ok := b["status"].(string); ok {
			status = s
		}
		
		created := ""
		if c, ok := b["created_at"].(string); ok {
			if len(c) >= 19 {
				created = c[:19]
			} else {
				created = c
			}
		}

		tw.AddRow(id, filename, size, status, created)
	}

	return tw.Render()
}

func outputBackupsCSV(out io.Writer, backups []map[string]interface{}) error {
	headers := []string{"id", "filename", "file_size_bytes", "status", "created_at"}
	rows := make([][]string, len(backups))

	for i, b := range backups {
		id := ""
		if v, ok := b["id"].(float64); ok {
			id = fmt.Sprintf("%.0f", v)
		} else if v, ok := b["id"].(string); ok {
			id = v
		}
		
		filename := ""
		if f, ok := b["filename"].(string); ok {
			filename = f
		}

		size := ""
		if s, ok := b["file_size_bytes"].(float64); ok {
			size = fmt.Sprintf("%.0f", s)
		}
		status := ""
		if s, ok := b["status"].(string); ok {
			status = s
		}
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
		if f, ok := b["filename"].(string); ok {
			values[i] = f
		}
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
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	reqBody := map[string]bool{"force": true}
	var result map[string]interface{}

	if err := client.Post("/api/backups/"+backupID+"/restore", reqBody, &result); err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), "✓ Restore operation started!")
	fmt.Fprintln(cmd.OutOrStdout(), "  Note: Restore runs in background. Check backup status for progress.")

	return nil
}

func runBackupDelete(cmd *cobra.Command, args []string) error {
	backupID := args[0]
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	if err := client.Delete("/api/backups/" + backupID); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Backup %s deleted\n", backupID)
	return nil
}

func runBackupDownload(cmd *cobra.Command, args []string) error {
	backupID := args[0]
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	// First get backup info
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := client.Get("/api/backups/"+backupID, &result); err != nil {
		return err
	}

	filename, ok := result.Data["filename"].(string)
	if !ok || filename == "" {
		return fmt.Errorf("could not determine filename from backup metadata")
	}
	outputPath := filepath.Join(backupOutputDir, filename)

	f, err := os.OpenFile(outputPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}
	defer f.Close()

	if err := client.Download("/api/backups/"+backupID+"/download", f); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Backup downloaded: %s\n", outputPath)
	return nil
}

func runBackupSchedule(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		// Get current schedule
		var result struct {
			Data map[string]interface{} `json:"data"`
		}

		if err := client.Get("/api/backups/schedule", &result); err != nil {
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
			fmt.Fprintln(cmd.OutOrStdout(), "No backup schedule configured")
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Current schedule: %s\n", schedule)
			if nextRun != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Next run: %s\n", nextRun)
			}
		}
		return nil
	}

	// Set schedule
	cronExpr := args[0]
	reqBody := map[string]string{"cron": cronExpr}
	var result map[string]interface{}

	if err := client.Post("/api/backups/schedule", reqBody, &result); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Backup schedule updated: %s\n", cronExpr)
	return nil
}
