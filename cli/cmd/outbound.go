package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/isolate-project/isolate-panel/cli/pkg"
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
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	var result struct {
		Data []map[string]interface{} `json:"data"`
	}

	if err := client.Get("/api/outbounds", &result); err != nil {
		return err
	}

	return outputOutbounds(cmd.OutOrStdout(), result.Data, false)
}

func runOutboundShow(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := client.Get("/api/outbounds/"+args[0], &result); err != nil {
		return err
	}

	return outputOutbounds(cmd.OutOrStdout(), []map[string]interface{}{result}, true)
}

func runOutboundCreate(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	// Just an example stub, as we didn't add flags for this command yet.
	// Normally we would parse args mapping it to the body request
	reqBody := map[string]interface{}{}

	var result map[string]interface{}
	if err := client.Post("/api/outbounds", reqBody, &result); err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), "✓ Outbound created successfully")
	return outputOutbounds(cmd.OutOrStdout(), []map[string]interface{}{result}, true)
}

func runOutboundDelete(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	if err := client.Delete("/api/outbounds/" + args[0]); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Outbound %s deleted successfully\n", args[0])
	return nil
}

func outputOutbounds(out io.Writer, outbounds []map[string]interface{}, detailed bool) error {
	format := pkg.ParseFormat(outboundFormat)

	switch format {
	case pkg.FormatJSON:
		if detailed && len(outbounds) == 1 {
			return pkg.WriteJSON(out, outbounds[0])
		}
		return pkg.WriteJSON(out, outbounds)
	case pkg.FormatCSV:
		return outputOutboundsCSV(out, outbounds)
	case pkg.FormatQuiet:
		return outputOutboundsQuiet(out, outbounds)
	default:
		return outputOutboundsTable(out, outbounds, detailed)
	}
}

func outputOutboundsTable(out io.Writer, outbounds []map[string]interface{}, detailed bool) error {
	tw := pkg.NewTableWriter(out)
	
	if detailed && len(outbounds) == 1 {
		tw.AddRow("PROPERTY", "VALUE")
		for k, v := range outbounds[0] {
			tw.AddRow(k, fmt.Sprintf("%v", v))
		}
	} else {
		tw.AddRow("ID", "NAME", "TYPE", "CORE", "ENABLED")
		for _, o := range outbounds {
			id := ""
			if v, ok := o["id"].(float64); ok {
				id = fmt.Sprintf("%.0f", v)
			} else if v, ok := o["id"].(string); ok {
				id = v
			}
			name, _ := o["name"].(string)
			otype, _ := o["type"].(string)
			core, _ := o["core"].(string)
			
			enabled := "true"
			if e, ok := o["enabled"].(bool); ok && !e {
				enabled = "false"
			}

			tw.AddRow(id, name, otype, core, enabled)
		}
	}

	return tw.Render()
}

func outputOutboundsCSV(out io.Writer, outbounds []map[string]interface{}) error {
	headers := []string{"id", "name", "type", "core", "enabled"}
	rows := make([][]string, len(outbounds))

	for i, o := range outbounds {
		id := ""
		if v, ok := o["id"].(float64); ok {
			id = fmt.Sprintf("%.0f", v)
		} else if v, ok := o["id"].(string); ok {
			id = v
		}
		name, _ := o["name"].(string)
		otype, _ := o["type"].(string)
		core, _ := o["core"].(string)
		
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
		values[i], _ = o["name"].(string)
	}
	return pkg.WriteQuiet(out, values)
}
