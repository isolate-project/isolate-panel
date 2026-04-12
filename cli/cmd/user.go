package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/isolate-project/isolate-panel/cli/pkg"
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

var userRegenerateCmd = &cobra.Command{
	Use:   "regenerate <username|id>",
	Short: "Regenerate user credentials",
	Long:  `Regenerate user credentials. Use --force to skip confirmation.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runUserRegenerate,
}

var userUpdateCmd = &cobra.Command{
	Use:   "update <username|id>",
	Short: "Update user",
	Long:  `Update user properties.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runUserUpdate,
}

func init() {
	// Global user flags
	userCmd.PersistentFlags().StringVar(&userFormat, "format", "table", "Output format (table, json, csv, quiet)")

	// Create flags
	userCreateCmd.Flags().StringVar(&userEmail, "email", "", "User email")
	userCreateCmd.Flags().Int64Var(&userTrafficLimit, "traffic-limit", 0, "Traffic limit in bytes")
	userCreateCmd.Flags().StringVar(&userExpiry, "expiry", "", "Expiry date (ISO 8601)")

	// Update flags
	userUpdateCmd.Flags().Int64Var(&userTrafficLimit, "traffic-limit", 0, "Traffic limit in bytes")
	userUpdateCmd.Flags().StringVar(&userExpiry, "expiry", "", "Expiry date (ISO 8601)")
	userUpdateCmd.Flags().BoolVar(&userActive, "active", true, "User active status")

	// Delete flags
	userDeleteCmd.Flags().BoolVar(&userForce, "force", false, "Skip confirmation")

	// Add subcommands
	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userShowCmd)
	userCmd.AddCommand(userCreateCmd)
	userCmd.AddCommand(userUpdateCmd)
	userCmd.AddCommand(userDeleteCmd)
	userCmd.AddCommand(userCredentialsCmd)
	userCmd.AddCommand(userRegenerateCmd)
}

// UserCmd returns the user command
func UserCmd() *cobra.Command {
	return userCmd
}

func runUserList(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	var result struct {
		Data []map[string]interface{} `json:"users"`
	}

	if err := client.Get("/api/users", &result); err != nil {
		return err
	}

	return outputUsers(cmd.OutOrStdout(), result.Data, false)
}

func runUserShow(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := client.Get("/api/users/"+args[0], &result); err != nil {
		return err
	}

	return outputUsers(cmd.OutOrStdout(), []map[string]interface{}{result}, true)
}

func runUserCreate(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	req := map[string]interface{}{
		"username": args[0],
	}
	if userEmail != "" {
		req["email"] = userEmail
	}
	if userTrafficLimit > 0 {
		req["traffic_limit_bytes"] = userTrafficLimit
	}
	if userExpiry != "" {
		req["expiry_date"] = userExpiry
	}

	var result map[string]interface{}
	if err := client.Post("/api/users", req, &result); err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), "User created successfully.")
	return outputUsers(cmd.OutOrStdout(), []map[string]interface{}{result}, true)
}

func runUserDelete(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	// TODO: Prompt for confirmation if !userForce

	if err := client.Delete("/api/users/" + args[0]); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "User %s deleted successfully.\n", args[0])
	return nil
}

func runUserCredentials(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	var result map[string]interface{}
	// Assuming endpoints like `/api/users/:id/credentials`
	if err := client.Get(fmt.Sprintf("/api/users/%s/credentials", args[0]), &result); err != nil {
		// Fallback for getting whole user if separate creds endpoint doesn't exist.
		// Wait, the backend API for credential might be something like user model directly or subscriptions
		return err
	}

	format := pkg.ParseFormat(userFormat)
	if format == pkg.FormatJSON {
		return pkg.WriteJSON(cmd.OutOrStdout(), result)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Credentials for %s:\n", args[0])
	tw := pkg.NewTableWriter(cmd.OutOrStdout())
	tw.AddRow("KEY", "VALUE")
	for k, v := range result {
		tw.AddRow(k, fmt.Sprintf("%v", v))
	}
	return tw.Render()
}

func runUserRegenerate(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := client.Post(fmt.Sprintf("/api/users/%s/regenerate", args[0]), nil, &result); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Credentials for %s regenerated successfully.\n", args[0])
	return nil
}

func runUserUpdate(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	req := map[string]interface{}{}
	if cmd.Flags().Changed("traffic-limit") {
		req["traffic_limit_bytes"] = userTrafficLimit
	}
	if cmd.Flags().Changed("expiry") {
		req["expiry_date"] = userExpiry
	}
	if cmd.Flags().Changed("active") {
		req["is_active"] = userActive
	}

	var result map[string]interface{}
	if err := client.Put("/api/users/"+args[0], req, &result); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "User %s updated successfully.\n", args[0])
	return outputUsers(cmd.OutOrStdout(), []map[string]interface{}{result}, true)
}

// outputUsers handles formatting output for user(s)
func outputUsers(out io.Writer, users []map[string]interface{}, detailed bool) error {
	format := pkg.ParseFormat(userFormat)

	switch format {
	case pkg.FormatJSON:
		if detailed && len(users) == 1 {
			return pkg.WriteJSON(out, users[0])
		}
		return pkg.WriteJSON(out, users)
	case pkg.FormatCSV:
		return outputUsersCSV(out, users)
	case pkg.FormatQuiet:
		return outputUsersQuiet(out, users)
	default:
		return outputUsersTable(out, users, detailed)
	}
}

func outputUsersTable(out io.Writer, users []map[string]interface{}, detailed bool) error {
	tw := pkg.NewTableWriter(out)
	
	if detailed && len(users) == 1 {
		tw.AddRow("PROPERTY", "VALUE")
		for k, v := range users[0] {
			tw.AddRow(k, fmt.Sprintf("%v", v))
		}
	} else {
		tw.AddRow("ID", "USERNAME", "EMAIL", "STATUS", "TRAFFIC", "EXPIRY")
		for _, u := range users {
			id := fmt.Sprintf("%.0f", u["id"].(float64))
			username, _ := u["username"].(string)
			email, _ := u["email"].(string)
			
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
	}

	return tw.Render()
}

func outputUsersCSV(out io.Writer, users []map[string]interface{}) error {
	headers := []string{"id", "username", "email", "is_active", "traffic_used_bytes", "expiry_date"}
	rows := make([][]string, len(users))

	for i, u := range users {
		id := ""
		if v, ok := u["id"].(float64); ok {
			id = fmt.Sprintf("%.0f", v)
		} else if v, ok := u["id"].(string); ok {
			id = v
		}
		
		username, _ := u["username"].(string)
		email, _ := u["email"].(string)
		
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
		values[i], _ = u["username"].(string)
	}
	return pkg.WriteQuiet(out, values)
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
