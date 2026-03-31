package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/isolate-project/isolate-panel/cli/pkg"
)

var (
	statsFormat string
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show statistics",
	Long:  `Show dashboard statistics.`,
	RunE:  runStats,
}

var connectionsCmd = &cobra.Command{
	Use:   "connections",
	Short: "Show active connections",
	Long:  `Show active proxy connections.`,
	RunE:  runConnections,
}

func init() {
	// Stats flags
	statsCmd.Flags().StringVar(&statsFormat, "format", "table", "Output format (table, json, csv, quiet)")
	connectionsCmd.Flags().StringVar(&statsFormat, "format", "table", "Output format (table, json, csv, quiet)")
}

// StatsCmd returns the stats command
func StatsCmd() *cobra.Command {
	return statsCmd
}

// ConnectionsCmd returns the connections command
func ConnectionsCmd() *cobra.Command {
	return connectionsCmd
}

func runStats(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	var result struct {
		Data map[string]interface{} `json:"data"`
	}

	if err := client.Get("/api/traffic/overview", &result); err != nil {
		return err
	}

	format := pkg.ParseFormat(statsFormat)
	if format == pkg.FormatJSON {
		return pkg.WriteJSON(cmd.OutOrStdout(), result.Data)
	}

	tw := pkg.NewTableWriter(cmd.OutOrStdout())
	tw.AddRow("METRIC", "VALUE")
	for k, v := range result.Data {
		tw.AddRow(k, fmt.Sprintf("%v", v))
	}
	return tw.Render()
}

func runConnections(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	var result struct {
		Data []map[string]interface{} `json:"data"`
	}

	if err := client.Get("/api/connections", &result); err != nil {
		return err
	}

	format := pkg.ParseFormat(statsFormat)
	if format == pkg.FormatJSON {
		return pkg.WriteJSON(cmd.OutOrStdout(), result.Data)
	}

	tw := pkg.NewTableWriter(cmd.OutOrStdout())
	tw.AddRow("USER_ID", "IP", "REMOTE_ADDR")
	for _, c := range result.Data {
		userID := fmt.Sprintf("%v", c["user_id"])
		ip := fmt.Sprintf("%v", c["ip"])
		remote := fmt.Sprintf("%v", c["remote_addr"])
		tw.AddRow(userID, ip, remote)
	}
	return tw.Render()
}
