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
	outboundFormat string
)

var outboundCmd = &cobra.Command{
	Use:   "outbound",
	Short: "Outbound management commands",
	Long:  `Manage Isolate Panel outbounds.`,
}

var outboundListCmd = &cobra.Command{
	Use:   "list",
	Short: "List outbounds",
	Long:  `List all outbounds.`,
	RunE:  runOutboundList,
}

var outboundShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show outbound details",
	Long:  `Show detailed information about an outbound.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runOutboundShow,
}

var outboundCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new outbound",
	Long:  `Create a new outbound.`,
	RunE:  runOutboundCreate,
}

var outboundDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an outbound",
	Long:  `Delete an outbound.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runOutboundDelete,
}

func init() {
	// Global outbound flags
	outboundCmd.PersistentFlags().StringVar(&outboundFormat, "format", "table", "Output format (table, json, csv, quiet)")

	// Add subcommands
	outboundCmd.AddCommand(outboundListCmd)
	outboundCmd.AddCommand(outboundShowCmd)
	outboundCmd.AddCommand(outboundCreateCmd)
	outboundCmd.AddCommand(outboundDeleteCmd)
}

// OutboundCmd returns the outbound command
func OutboundCmd() *cobra.Command {
	return outboundCmd
}

func runOutboundList(cmd *cobra.Command, args []string) error {
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
	req, err := http.NewRequest("GET", profile.PanelURL+"/api/outbounds", nil)
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
	format := pkg.ParseFormat(outboundFormat)

	switch format {
	case pkg.FormatJSON:
		return pkg.WriteJSON(cmd.OutOrStdout(), result.Data)
	case pkg.FormatCSV:
		return outputOutboundsCSV(cmd.OutOrStdout(), result.Data)
	case pkg.FormatQuiet:
		return outputOutboundsQuiet(cmd.OutOrStdout(), result.Data)
	default:
		return outputOutboundsTable(cmd.OutOrStdout(), result.Data)
	}
}

func outputOutboundsTable(out io.Writer, outbounds []map[string]interface{}) error {
	tw := pkg.NewTableWriter(out)
	tw.AddRow("ID", "NAME", "TYPE", "CORE", "ENABLED")

	for _, o := range outbounds {
		id := fmt.Sprintf("%.0f", o["id"].(float64))
		name := ""
		if n, ok := o["name"].(string); ok {
			name = n
		}
		otype := ""
		if t, ok := o["type"].(string); ok {
			otype = t
		}
		core := ""
		if c, ok := o["core"].(string); ok {
			core = c
		}
		enabled := "true"
		if e, ok := o["enabled"].(bool); ok && !e {
			enabled = "false"
		}

		tw.AddRow(id, name, otype, core, enabled)
	}

	return tw.Render()
}

func outputOutboundsCSV(out io.Writer, outbounds []map[string]interface{}) error {
	headers := []string{"id", "name", "type", "core", "enabled"}
	rows := make([][]string, len(outbounds))

	for i, o := range outbounds {
		id := fmt.Sprintf("%.0f", o["id"].(float64))
		name := ""
		if n, ok := o["name"].(string); ok {
			name = n
		}
		otype := ""
		if t, ok := o["type"].(string); ok {
			otype = t
		}
		core := ""
		if c, ok := o["core"].(string); ok {
			core = c
		}
		enabled := "true"
		if e, ok := o["enabled"].(bool); ok && !e {
			enabled = "false"
		}

		rows[i] = []string{id, name, otype, core, enabled}
	}

	return pkg.WriteCSV(out, headers, rows)
}

func outputOutboundsQuiet(out io.Writer, outbounds []map[string]interface{}) error {
	values := make([]string, len(outbounds))
	for i, o := range outbounds {
		if n, ok := o["name"].(string); ok {
			values[i] = n
		}
	}
	return pkg.WriteQuiet(out, values)
}

func runOutboundShow(cmd *cobra.Command, args []string) error {
	fmt.Println("Outbound show command - to be implemented")
	return nil
}

func runOutboundCreate(cmd *cobra.Command, args []string) error {
	fmt.Println("Outbound create command - to be implemented")
	return nil
}

func runOutboundDelete(cmd *cobra.Command, args []string) error {
	fmt.Printf("Deleting outbound: %s\n", args[0])
	fmt.Println("API integration - to be implemented")
	return nil
}
