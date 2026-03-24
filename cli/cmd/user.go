package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/vovk4morkovk4/isolate-panel/cli/pkg"
)

var (
	userFormat       string
	userEmail        string
	userTrafficLimit int64
	userExpiry       string
	userActive       bool
	userForce        bool
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "User management commands",
	Long:  `Manage Isolate Panel users.`,
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List users",
	Long:  `List all users with optional filtering.`,
	RunE:  runUserList,
}

var userShowCmd = &cobra.Command{
	Use:   "show <username|id>",
	Short: "Show user details",
	Long:  `Show detailed information about a user.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runUserShow,
}

var userCreateCmd = &cobra.Command{
	Use:   "create <username>",
	Short: "Create a new user",
	Long:  `Create a new user with specified parameters.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runUserCreate,
}

var userDeleteCmd = &cobra.Command{
	Use:   "delete <username|id>",
	Short: "Delete a user",
	Long:  `Delete a user. Use --force to skip confirmation.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runUserDelete,
}

var userCredentialsCmd = &cobra.Command{
	Use:   "credentials <username|id>",
	Short: "Show user credentials",
	Long:  `Show user credentials (admin only).`,
	Args:  cobra.ExactArgs(1),
	RunE:  runUserCredentials,
}

func init() {
	// Global user flags
	userCmd.PersistentFlags().StringVar(&userFormat, "format", "table", "Output format (table, json, csv, quiet)")

	// Create flags
	userCreateCmd.Flags().StringVar(&userEmail, "email", "", "User email")
	userCreateCmd.Flags().Int64Var(&userTrafficLimit, "traffic-limit", 0, "Traffic limit in bytes")
	userCreateCmd.Flags().StringVar(&userExpiry, "expiry", "", "Expiry date (ISO 8601)")

	// Delete flags
	userDeleteCmd.Flags().BoolVar(&userForce, "force", false, "Skip confirmation")

	// Add subcommands
	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userShowCmd)
	userCmd.AddCommand(userCreateCmd)
	userCmd.AddCommand(userDeleteCmd)
	userCmd.AddCommand(userCredentialsCmd)
}

// UserCmd returns the user command
func UserCmd() *cobra.Command {
	return userCmd
}

func runUserList(cmd *cobra.Command, args []string) error {
	config, err := pkg.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	profile, err := config.GetCurrentProfile()
	if err != nil {
		return fmt.Errorf("no profile selected. Use 'isolate-panel login' first")
	}

	// Make API request
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", profile.PanelURL+"/api/users", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+profile.AccessToken)

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
	format := pkg.ParseFormat(userFormat)

	switch format {
	case pkg.FormatJSON:
		return pkg.WriteJSON(cmd.OutOrStdout(), result.Data)
	case pkg.FormatCSV:
		return outputUsersCSV(cmd.OutOrStdout(), result.Data)
	case pkg.FormatQuiet:
		return outputUsersQuiet(cmd.OutOrStdout(), result.Data)
	default:
		return outputUsersTable(cmd.OutOrStdout(), result.Data)
	}
}

func outputUsersTable(out io.Writer, users []map[string]interface{}) error {
	tw := pkg.NewTableWriter(out)
	tw.AddRow("ID", "USERNAME", "EMAIL", "STATUS", "TRAFFIC", "EXPIRY")

	for _, u := range users {
		id := fmt.Sprintf("%.0f", u["id"].(float64))
		username := u["username"].(string)
		email := ""
		if e, ok := u["email"].(string); ok {
			email = e
		}
		isActive := "Active"
		if active, ok := u["is_active"].(bool); ok && !active {
			isActive = "Inactive"
		}
		traffic := "0 B"
		if t, ok := u["traffic_used_bytes"].(float64); ok {
			traffic = formatBytes(int64(t))
		}
		expiry := "Never"
		if e, ok := u["expiry_date"].(string); ok && e != "" {
			expiry = e[:10]
		}

		tw.AddRow(id, username, email, isActive, traffic, expiry)
	}

	return tw.Render()
}

func outputUsersCSV(out io.Writer, users []map[string]interface{}) error {
	headers := []string{"id", "username", "email", "is_active", "traffic_used_bytes", "expiry_date"}
	rows := make([][]string, len(users))

	for i, u := range users {
		id := fmt.Sprintf("%.0f", u["id"].(float64))
		username := u["username"].(string)
		email := ""
		if e, ok := u["email"].(string); ok {
			email = e
		}
		isActive := "true"
		if active, ok := u["is_active"].(bool); ok && !active {
			isActive = "false"
		}
		traffic := "0"
		if t, ok := u["traffic_used_bytes"].(float64); ok {
			traffic = fmt.Sprintf("%.0f", t)
		}
		expiry := ""
		if e, ok := u["expiry_date"].(string); ok && e != "" {
			expiry = e
		}

		rows[i] = []string{id, username, email, isActive, traffic, expiry}
	}

	return pkg.WriteCSV(out, headers, rows)
}

func outputUsersQuiet(out io.Writer, users []map[string]interface{}) error {
	values := make([]string, len(users))
	for i, u := range users {
		values[i] = u["username"].(string)
	}
	return pkg.WriteQuiet(out, values)
}

func runUserShow(cmd *cobra.Command, args []string) error {
	// TODO: Implement user show
	fmt.Println("User show command - to be implemented")
	return nil
}

func runUserCreate(cmd *cobra.Command, args []string) error {
	// TODO: Implement user create
	fmt.Println("User create command - to be implemented")
	return nil
}

func runUserDelete(cmd *cobra.Command, args []string) error {
	// TODO: Implement user delete
	fmt.Println("User delete command - to be implemented")
	return nil
}

func runUserCredentials(cmd *cobra.Command, args []string) error {
	// TODO: Implement user credentials
	fmt.Println("User credentials command - to be implemented")
	return nil
}

func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
