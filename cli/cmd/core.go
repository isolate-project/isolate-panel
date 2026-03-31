package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/isolate-project/isolate-panel/cli/pkg"
)

var (
	coreFormat string
	coreTail   int
	coreFollow bool
)

var coreCmd = &cobra.Command{
	Use:   "core",
	Short: "Core management commands",
	Long:  `Manage Isolate Panel proxy cores (singbox, xray, mihomo).`,
}

var coreListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cores",
	Long:  `List all cores and their status.`,
	RunE:  runCoreList,
}

var coreStatusCmd = &cobra.Command{
	Use:   "status <core>",
	Short: "Show core status",
	Long:  `Show status of a specific core.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCoreStatus,
}

var coreStartCmd = &cobra.Command{
	Use:   "start <singbox|xray|mihomo>",
	Short: "Start a core",
	Long:  `Start a proxy core.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCoreStart,
}

var coreStopCmd = &cobra.Command{
	Use:   "stop <singbox|xray|mihomo>",
	Short: "Stop a core",
	Long:  `Stop a proxy core.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCoreStop,
}

var coreRestartCmd = &cobra.Command{
	Use:   "restart <singbox|xray|mihomo>",
	Short: "Restart a core",
	Long:  `Restart a proxy core.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCoreRestart,
}

var coreLogsCmd = &cobra.Command{
	Use:   "logs <singbox|xray|mihomo>",
	Short: "Show core logs",
	Long:  `Show logs for a proxy core.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCoreLogs,
}

func init() {
	// Global core flags
	coreCmd.PersistentFlags().StringVar(&coreFormat, "format", "table", "Output format (table, json, csv, quiet)")

	// Logs flags
	coreLogsCmd.Flags().IntVar(&coreTail, "tail", 100, "Number of lines to show")
	coreLogsCmd.Flags().BoolVar(&coreFollow, "follow", false, "Follow log output")

	// Add subcommands
	coreCmd.AddCommand(coreListCmd)
	coreCmd.AddCommand(coreStatusCmd)
	coreCmd.AddCommand(coreStartCmd)
	coreCmd.AddCommand(coreStopCmd)
	coreCmd.AddCommand(coreRestartCmd)
	coreCmd.AddCommand(coreLogsCmd)
}

// CoreCmd returns the core command
func CoreCmd() *cobra.Command {
	return coreCmd
}

func runCoreList(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	var result struct {
		Data []map[string]interface{} `json:"data"`
	}

	if err := client.Get("/api/cores", &result); err != nil {
		return err
	}

	return outputCores(cmd.OutOrStdout(), result.Data, false)
}

func runCoreStatus(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := client.Get("/api/cores/"+args[0]+"/status", &result); err != nil {
		return err
	}

	return outputCores(cmd.OutOrStdout(), []map[string]interface{}{result}, true)
}

func runCoreStart(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := client.Post("/api/cores/"+args[0]+"/start", nil, &result); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Core %s started successfully.\n", args[0])
	return nil
}

func runCoreStop(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := client.Post("/api/cores/"+args[0]+"/stop", nil, &result); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Core %s stopped successfully.\n", args[0])
	return nil
}

func runCoreRestart(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := client.Post("/api/cores/"+args[0]+"/restart", nil, &result); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Core %s restarted successfully.\n", args[0])
	return nil
}

func runCoreLogs(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	return client.StreamCoreLogs(context.Background(), args[0], coreTail, coreFollow, cmd.OutOrStdout())
}

// outputCores handles formatting output for core(s)
func outputCores(out io.Writer, cores []map[string]interface{}, detailed bool) error {
	format := pkg.ParseFormat(coreFormat)

	switch format {
	case pkg.FormatJSON:
		if detailed && len(cores) == 1 {
			return pkg.WriteJSON(out, cores[0])
		}
		return pkg.WriteJSON(out, cores)
	case pkg.FormatCSV:
		return outputCoresCSV(out, cores)
	case pkg.FormatQuiet:
		return outputCoresQuiet(out, cores)
	default:
		return outputCoresTable(out, cores, detailed)
	}
}

func outputCoresTable(out io.Writer, cores []map[string]interface{}, detailed bool) error {
	tw := pkg.NewTableWriter(out)

	if detailed && len(cores) == 1 {
		tw.AddRow("PROPERTY", "VALUE")
		for k, v := range cores[0] {
			tw.AddRow(k, fmt.Sprintf("%v", v))
		}
	} else {
		tw.AddRow("NAME", "STATUS", "UPTIME", "VERSION")
		for _, c := range cores {
			name := ""
			if n, ok := c["name"].(string); ok {
				name = n
			}
			status := "stopped"
			if s, ok := c["status"].(string); ok {
				status = s
			}
			uptime := "0"
			if u, ok := c["uptime"].(string); ok {
				uptime = u
			}
			version := ""
			if v, ok := c["version"].(string); ok {
				version = v
			}

			tw.AddRow(name, status, uptime, version)
		}
	}

	return tw.Render()
}

func outputCoresCSV(out io.Writer, cores []map[string]interface{}) error {
	headers := []string{"name", "status", "uptime", "version"}
	rows := make([][]string, len(cores))

	for i, c := range cores {
		name := ""
		if n, ok := c["name"].(string); ok {
			name = n
		}
		status := ""
		if s, ok := c["status"].(string); ok {
			status = s
		}
		uptime := ""
		if u, ok := c["uptime"].(string); ok {
			uptime = u
		}
		version := ""
		if v, ok := c["version"].(string); ok {
			version = v
		}

		rows[i] = []string{name, status, uptime, version}
	}

	return pkg.WriteCSV(out, headers, rows)
}

func outputCoresQuiet(out io.Writer, cores []map[string]interface{}) error {
	values := make([]string, len(cores))
	for i, c := range cores {
		if n, ok := c["name"].(string); ok {
			values[i] = n
		}
	}
	return pkg.WriteQuiet(out, values)
}
