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
}

// InboundCmd returns the inbound command
func InboundCmd() *cobra.Command {
	return inboundCmd
}

func runInboundList(cmd *cobra.Command, args []string) error {
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
	req, err := http.NewRequest("GET", profile.PanelURL+"/api/inbounds", nil)
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
	format := pkg.ParseFormat(inboundFormat)

	switch format {
	case pkg.FormatJSON:
		return pkg.WriteJSON(cmd.OutOrStdout(), result.Data)
	case pkg.FormatCSV:
		return outputInboundsCSV(cmd.OutOrStdout(), result.Data)
	case pkg.FormatQuiet:
		return outputInboundsQuiet(cmd.OutOrStdout(), result.Data)
	default:
		return outputInboundsTable(cmd.OutOrStdout(), result.Data)
	}
}

func outputInboundsTable(out io.Writer, inbounds []map[string]interface{}) error {
	tw := pkg.NewTableWriter(out)
	tw.AddRow("ID", "NAME", "PROTOCOL", "PORT", "CORE", "USERS")

	for _, i := range inbounds {
		id := fmt.Sprintf("%.0f", i["id"].(float64))
		name := i["name"].(string)
		protocol := ""
		if p, ok := i["protocol"].(string); ok {
			protocol = p
		}
		port := ""
		if p, ok := i["port"].(float64); ok {
			port = fmt.Sprintf("%.0f", p)
		}
		core := ""
		if c, ok := i["core"].(string); ok {
			core = c
		}
		users := "0"
		if u, ok := i["user_count"].(float64); ok {
			users = fmt.Sprintf("%.0f", u)
		}

		tw.AddRow(id, name, protocol, port, core, users)
	}

	return tw.Render()
}

func outputInboundsCSV(out io.Writer, inbounds []map[string]interface{}) error {
	headers := []string{"id", "name", "protocol", "port", "core", "user_count"}
	rows := make([][]string, len(inbounds))

	for i, ib := range inbounds {
		id := fmt.Sprintf("%.0f", ib["id"].(float64))
		name := ib["name"].(string)
		protocol := ""
		if p, ok := ib["protocol"].(string); ok {
			protocol = p
		}
		port := ""
		if p, ok := ib["port"].(float64); ok {
			port = fmt.Sprintf("%.0f", p)
		}
		core := ""
		if c, ok := ib["core"].(string); ok {
			core = c
		}
		users := "0"
		if u, ok := ib["user_count"].(float64); ok {
			users = fmt.Sprintf("%.0f", u)
		}

		rows[i] = []string{id, name, protocol, port, core, users}
	}

	return pkg.WriteCSV(out, headers, rows)
}

func outputInboundsQuiet(out io.Writer, inbounds []map[string]interface{}) error {
	values := make([]string, len(inbounds))
	for i, ib := range inbounds {
		values[i] = ib["name"].(string)
	}
	return pkg.WriteQuiet(out, values)
}

func runInboundShow(cmd *cobra.Command, args []string) error {
	fmt.Println("Inbound show command - to be implemented")
	return nil
}

func runInboundCreate(cmd *cobra.Command, args []string) error {
	if inboundName == "" {
		return fmt.Errorf("--name is required")
	}
	if inboundPort == 0 {
		return fmt.Errorf("--port is required")
	}

	fmt.Printf("Creating inbound: %s (%s/%d)\n", inboundName, inboundProtocol, inboundPort)
	fmt.Println("API integration - to be implemented")
	return nil
}

func runInboundDelete(cmd *cobra.Command, args []string) error {
	fmt.Printf("Deleting inbound: %s\n", args[0])
	fmt.Println("API integration - to be implemented")
	return nil
}
