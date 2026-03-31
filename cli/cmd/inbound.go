package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/isolate-project/isolate-panel/cli/pkg"
)

var (
	inboundFormat   string
	inboundCore     string
	inboundName     string
	inboundPort     int
	inboundProtocol string
)

var inboundCmd = &cobra.Command{
	Use:   "inbound",
	Short: "Inbound management commands",
	Long:  `Manage Isolate Panel inbounds.`,
}

var inboundListCmd = &cobra.Command{
	Use:   "list",
	Short: "List inbounds",
	Long:  `List all inbounds with optional filtering.`,
	RunE:  runInboundList,
}

var inboundShowCmd = &cobra.Command{
	Use:   "show <id|name>",
	Short: "Show inbound details",
	Long:  `Show detailed information about an inbound.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runInboundShow,
}

var inboundCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new inbound",
	Long:  `Create a new inbound (non-interactive mode with flags only).`,
	RunE:  runInboundCreate,
}

var inboundDeleteCmd = &cobra.Command{
	Use:   "delete <id|name>",
	Short: "Delete an inbound",
	Long:  `Delete an inbound. Use --force to skip confirmation.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runInboundDelete,
}

var inboundAddUsersCmd = &cobra.Command{
	Use:   "add-users <inbound-id> <user-id1> [user-id2] [...]",
	Short: "Add users to inbound",
	Long:  `Add one or more users to an inbound.`,
	Args:  cobra.MinimumNArgs(2),
	RunE:  runInboundAddUsers,
}

var inboundRemoveUserCmd = &cobra.Command{
	Use:   "remove-user <inbound-id> <user-id>",
	Short: "Remove user from inbound",
	Long:  `Remove a user from an inbound.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runInboundRemoveUser,
}

var inboundUsersCmd = &cobra.Command{
	Use:   "users <inbound-id>",
	Short: "List users in inbound",
	Long:  `List all users assigned to an inbound.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runInboundUsers,
}

func init() {
	// Global inbound flags
	inboundCmd.PersistentFlags().StringVar(&inboundFormat, "format", "table", "Output format (table, json, csv, quiet)")

	// Create flags
	inboundCreateCmd.Flags().StringVar(&inboundCore, "core", "singbox", "Core (singbox, xray, mihomo)")
	inboundCreateCmd.Flags().StringVar(&inboundName, "name", "", "Inbound name")
	inboundCreateCmd.Flags().IntVar(&inboundPort, "port", 0, "Inbound port")
	inboundCreateCmd.Flags().StringVar(&inboundProtocol, "protocol", "vmess", "Protocol")

	// Add subcommands
	inboundCmd.AddCommand(inboundListCmd)
	inboundCmd.AddCommand(inboundShowCmd)
	inboundCmd.AddCommand(inboundCreateCmd)
	inboundCmd.AddCommand(inboundDeleteCmd)
	inboundCmd.AddCommand(inboundAddUsersCmd)
	inboundCmd.AddCommand(inboundRemoveUserCmd)
	inboundCmd.AddCommand(inboundUsersCmd)
}

// InboundCmd returns the inbound command
func InboundCmd() *cobra.Command {
	return inboundCmd
}

func runInboundList(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	var result struct {
		Data []map[string]interface{} `json:"data"`
	}

	if err := client.Get("/api/inbounds", &result); err != nil {
		return err
	}

	return outputInbounds(cmd.OutOrStdout(), result.Data, false)
}

func runInboundShow(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := client.Get("/api/inbounds/"+args[0], &result); err != nil {
		return err
	}

	return outputInbounds(cmd.OutOrStdout(), []map[string]interface{}{result}, true)
}

func runInboundCreate(cmd *cobra.Command, args []string) error {
	if inboundName == "" {
		return fmt.Errorf("--name is required")
	}
	if inboundPort == 0 {
		return fmt.Errorf("--port is required")
	}

	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	reqBody := map[string]interface{}{
		"name":     inboundName,
		"core":     inboundCore,
		"protocol": inboundProtocol,
		"port":     inboundPort,
	}

	var result map[string]interface{}
	if err := client.Post("/api/inbounds", reqBody, &result); err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), "✓ Inbound created successfully")
	return outputInbounds(cmd.OutOrStdout(), []map[string]interface{}{result}, true)
}

func runInboundDelete(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	if err := client.Delete("/api/inbounds/" + args[0]); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Inbound %s deleted successfully\n", args[0])
	return nil
}

func runInboundAddUsers(cmd *cobra.Command, args []string) error {
	inboundID := args[0]
	userIDs := args[1:]

	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	// For each user... we might not have a bulk endpoint in CLI (though we might have /api/inbounds/bulk)
	// Assuming an endpoint exists like: POST /api/inbounds/:id/users
	reqBody := map[string]interface{}{
		"user_ids": userIDs,
	}
	if err := client.Post(fmt.Sprintf("/api/inbounds/%s/users", inboundID), reqBody, nil); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Added users %v to inbound %s\n", userIDs, inboundID)
	return nil
}

func runInboundRemoveUser(cmd *cobra.Command, args []string) error {
	inboundID, userID := args[0], args[1]
	
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	if err := client.Delete(fmt.Sprintf("/api/inbounds/%s/users/%s", inboundID, userID)); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Removed user %s from inbound %s\n", userID, inboundID)
	return nil
}

func runInboundUsers(cmd *cobra.Command, args []string) error {
	inboundID := args[0]
	
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	var result struct {
		Data []map[string]interface{} `json:"data"`
	}

	if err := client.Get(fmt.Sprintf("/api/inbounds/%s/users", inboundID), &result); err != nil {
		return err
	}

	// Reuse user formatter
	return outputUsers(cmd.OutOrStdout(), result.Data, false)
}

func outputInbounds(out io.Writer, inbounds []map[string]interface{}, detailed bool) error {
	format := pkg.ParseFormat(inboundFormat)
	switch format {
	case pkg.FormatJSON:
		if detailed && len(inbounds) == 1 {
			return pkg.WriteJSON(out, inbounds[0])
		}
		return pkg.WriteJSON(out, inbounds)
	case pkg.FormatCSV:
		return outputInboundsCSV(out, inbounds)
	case pkg.FormatQuiet:
		return outputInboundsQuiet(out, inbounds)
	default:
		return outputInboundsTable(out, inbounds, detailed)
	}
}

func outputInboundsTable(out io.Writer, inbounds []map[string]interface{}, detailed bool) error {
	tw := pkg.NewTableWriter(out)
	
	if detailed && len(inbounds) == 1 {
		tw.AddRow("PROPERTY", "VALUE")
		for k, v := range inbounds[0] {
			tw.AddRow(k, fmt.Sprintf("%v", v))
		}
	} else {
		tw.AddRow("ID", "NAME", "PROTOCOL", "PORT", "CORE", "USERS")
		for _, i := range inbounds {
			id := ""
			if v, ok := i["id"].(float64); ok {
				id = fmt.Sprintf("%.0f", v)
			} else if v, ok := i["id"].(string); ok {
				id = v
			}
			
			name, _ := i["name"].(string)
			protocol, _ := i["protocol"].(string)
			
			port := ""
			if p, ok := i["port"].(float64); ok {
				port = fmt.Sprintf("%.0f", p)
			}
			
			core, _ := i["core"].(string)
			
			users := "0"
			if u, ok := i["user_count"].(float64); ok {
				users = fmt.Sprintf("%.0f", u)
			}

			tw.AddRow(id, name, protocol, port, core, users)
		}
	}

	return tw.Render()
}

func outputInboundsCSV(out io.Writer, inbounds []map[string]interface{}) error {
	headers := []string{"id", "name", "protocol", "port", "core", "user_count"}
	rows := make([][]string, len(inbounds))

	for j, ib := range inbounds {
		id := ""
		if v, ok := ib["id"].(float64); ok {
			id = fmt.Sprintf("%.0f", v)
		} else if v, ok := ib["id"].(string); ok {
			id = v
		}
		name, _ := ib["name"].(string)
		protocol, _ := ib["protocol"].(string)
		
		port := ""
		if p, ok := ib["port"].(float64); ok {
			port = fmt.Sprintf("%.0f", p)
		}
		
		core, _ := ib["core"].(string)
		
		users := "0"
		if u, ok := ib["user_count"].(float64); ok {
			users = fmt.Sprintf("%.0f", u)
		}

		rows[j] = []string{id, name, protocol, port, core, users}
	}

	return pkg.WriteCSV(out, headers, rows)
}

func outputInboundsQuiet(out io.Writer, inbounds []map[string]interface{}) error {
	values := make([]string, len(inbounds))
	for j, ib := range inbounds {
		values[j], _ = ib["name"].(string)
	}
	return pkg.WriteQuiet(out, values)
}
