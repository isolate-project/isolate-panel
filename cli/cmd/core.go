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
	Use:   "status [core]",
	Short: "Show core status",
	Long:  `Show status of a specific core or all cores.`,
	Args:  cobra.MaximumNArgs(1),
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
	req, err := http.NewRequest("GET", profile.PanelURL+"/api/cores", nil)
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
	format := pkg.ParseFormat(coreFormat)

	switch format {
	case pkg.FormatJSON:
		return pkg.WriteJSON(cmd.OutOrStdout(), result.Data)
	case pkg.FormatCSV:
		return outputCoresCSV(cmd.OutOrStdout(), result.Data)
	case pkg.FormatQuiet:
		return outputCoresQuiet(cmd.OutOrStdout(), result.Data)
	default:
		return outputCoresTable(cmd.OutOrStdout(), result.Data)
	}
}

func outputCoresTable(out io.Writer, cores []map[string]interface{}) error {
	tw := pkg.NewTableWriter(out)
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
		uptime := ""
		if u, ok := c["uptime"].(string); ok {
			uptime = u
		}
		version := ""
		if v, ok := c["version"].(string); ok {
			version = v
		}

		tw.AddRow(name, status, uptime, version)
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

func runCoreStatus(cmd *cobra.Command, args []string) error {
	fmt.Println("Core status command - to be implemented")
	return nil
}

func runCoreStart(cmd *cobra.Command, args []string) error {
	coreName := args[0]
	fmt.Printf("Starting core: %s\n", coreName)
	fmt.Println("API integration - to be implemented")
	return nil
}

func runCoreStop(cmd *cobra.Command, args []string) error {
	coreName := args[0]
	fmt.Printf("Stopping core: %s\n", coreName)
	fmt.Println("API integration - to be implemented")
	return nil
}

func runCoreRestart(cmd *cobra.Command, args []string) error {
	coreName := args[0]
	fmt.Printf("Restarting core: %s\n", coreName)
	fmt.Println("API integration - to be implemented")
	return nil
}

func runCoreLogs(cmd *cobra.Command, args []string) error {
	coreName := args[0]
	fmt.Printf("Showing logs for core: %s (tail=%d, follow=%v)\n", coreName, coreTail, coreFollow)
	fmt.Println("API integration - to be implemented")
	return nil
}
