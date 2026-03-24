package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
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
	fmt.Println("Stats command - to be implemented")
	return nil
}

func runConnections(cmd *cobra.Command, args []string) error {
	fmt.Println("Connections command - to be implemented")
	return nil
}
