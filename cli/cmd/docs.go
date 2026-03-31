package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var docsCmd = &cobra.Command{
	Use:    "docs [output-dir]",
	Short:  "Generate markdown documentation",
	Long:   `Generate markdown documentation for all cli commands.`,
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "./docs"
		if len(args) > 0 {
			dir = args[0]
		}

		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create docs directory: %w", err)
		}

		// rootCmd usually needs to be accessible, we'll assume this command will be added to rootCmd in root.go
		if cmd.Root() != nil {
			err := doc.GenMarkdownTree(cmd.Root(), dir)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Documentation generated successfully in %s\n", dir)
			return nil
		}
		
		return fmt.Errorf("root command not found")
	},
}

// DocsCmd returns the docs command
func DocsCmd() *cobra.Command {
	return docsCmd
}
