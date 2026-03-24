package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/vovk4morkovk4/isolate-panel/cli/cmd"
)

var (
	version = "0.3.0"
	rootCmd = &cobra.Command{
		Use:     "isolate-panel",
		Short:   "Isolate Panel CLI",
		Long:    `Isolate Panel - Lightweight proxy core management panel for Xray, Sing-box, and Mihomo`,
		Version: version,
	}
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Add global flags
	rootCmd.PersistentFlags().StringP("config", "c", "", "config file path")
	rootCmd.PersistentFlags().StringP("url", "u", "http://localhost:8080", "panel URL")
	rootCmd.PersistentFlags().StringP("token", "t", "", "API token")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(cmd.LoginCmd())
	rootCmd.AddCommand(cmd.LogoutCmd())
	rootCmd.AddCommand(cmd.ProfileCmd())
	// rootCmd.AddCommand(cmd.BackupCmd()) // Temporarily disabled
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Isolate Panel CLI version %s\n", version)
	},
}
