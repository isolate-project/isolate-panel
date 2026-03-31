package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/isolate-project/isolate-panel/cli/pkg"
)

var (
	certFormat   string
	certDomain   string
	certEmail    string
	certWildcard bool
)

var certCmd = &cobra.Command{
	Use:   "cert",
	Short: "Certificate management commands",
	Long:  `Manage TLS certificates.`,
}

var certListCmd = &cobra.Command{
	Use:   "list",
	Short: "List certificates",
	Long:  `List all certificates.`,
	RunE:  runCertList,
}

var certRequestCmd = &cobra.Command{
	Use:   "request <domain>",
	Short: "Request a certificate",
	Long:  `Request a new TLS certificate via ACME.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCertRequest,
}

var certShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show certificate details",
	Long:  `Show detailed information about a certificate.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCertShow,
}

var certRenewCmd = &cobra.Command{
	Use:   "renew <id>",
	Short: "Renew a certificate",
	Long:  `Renew an existing certificate.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCertRenew,
}

var certDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a certificate",
	Long:  `Delete a certificate.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCertDelete,
}

func init() {
	// Global cert flags
	certCmd.PersistentFlags().StringVar(&certFormat, "format", "table", "Output format (table, json, csv, quiet)")

	// Request flags
	certRequestCmd.Flags().StringVar(&certEmail, "email", "", "Email for ACME")
	certRequestCmd.Flags().BoolVar(&certWildcard, "wildcard", false, "Request wildcard certificate")

	// Add subcommands
	certCmd.AddCommand(certListCmd)
	certCmd.AddCommand(certRequestCmd)
	certCmd.AddCommand(certShowCmd)
	certCmd.AddCommand(certRenewCmd)
	certCmd.AddCommand(certDeleteCmd)
}

// CertCmd returns the cert command
func CertCmd() *cobra.Command {
	return certCmd
}

func runCertList(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	var result struct {
		Data []map[string]interface{} `json:"data"`
	}

	if err := client.Get("/api/certificates", &result); err != nil {
		return err
	}

	return outputCerts(cmd.OutOrStdout(), result.Data, false)
}

func runCertRequest(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	domain := args[0]
	reqBody := map[string]interface{}{
		"domain":   domain,
		"provider": "acme",
	}
	if certEmail != "" {
		// Just in case API takes email directly, otherwise it might come from settings
		reqBody["email"] = certEmail
	}

	var result map[string]interface{}
	if err := client.Post("/api/certificates/request", reqBody, &result); err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), "✓ Certificate requested successfully")
	return outputCerts(cmd.OutOrStdout(), []map[string]interface{}{result}, true)
}

func runCertShow(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := client.Get("/api/certificates/"+args[0], &result); err != nil {
		return err
	}

	return outputCerts(cmd.OutOrStdout(), []map[string]interface{}{result}, true)
}

func runCertRenew(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := client.Post("/api/certificates/"+args[0]+"/renew", nil, &result); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Certificate %s renewed successfully\n", args[0])
	return outputCerts(cmd.OutOrStdout(), []map[string]interface{}{result}, true)
}

func runCertDelete(cmd *cobra.Command, args []string) error {
	client, err := pkg.GetClient()
	if err != nil {
		return err
	}

	if err := client.Delete("/api/certificates/" + args[0]); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Certificate %s deleted successfully\n", args[0])
	return nil
}

// outputCerts handles formatting output for certificate(s)
func outputCerts(out io.Writer, certs []map[string]interface{}, detailed bool) error {
	format := pkg.ParseFormat(certFormat)

	switch format {
	case pkg.FormatJSON:
		if detailed && len(certs) == 1 {
			return pkg.WriteJSON(out, certs[0])
		}
		return pkg.WriteJSON(out, certs)
	case pkg.FormatCSV:
		headers := []string{"id", "domain", "status", "expires_at", "provider"}
		rows := make([][]string, len(certs))
		for i, c := range certs {
			id := ""
			if v, ok := c["id"].(float64); ok {
				id = fmt.Sprintf("%.0f", v)
			} else if v, ok := c["id"].(string); ok {
				id = v
			}
			domain, _ := c["domain"].(string)
			status, _ := c["status"].(string)
			expires, _ := c["expires_at"].(string)
			provider, _ := c["provider"].(string)

			rows[i] = []string{id, domain, status, expires, provider}
		}
		return pkg.WriteCSV(out, headers, rows)
	case pkg.FormatQuiet:
		values := make([]string, len(certs))
		for i, c := range certs {
			values[i], _ = c["domain"].(string)
		}
		return pkg.WriteQuiet(out, values)
	default:
		tw := pkg.NewTableWriter(out)
		if detailed && len(certs) == 1 {
			tw.AddRow("PROPERTY", "VALUE")
			for k, v := range certs[0] {
				tw.AddRow(k, fmt.Sprintf("%v", v))
			}
		} else {
			tw.AddRow("ID", "DOMAIN", "STATUS", "EXPIRES", "PROVIDER")
			for _, c := range certs {
				id := ""
				if v, ok := c["id"].(float64); ok {
					id = fmt.Sprintf("%.0f", v)
				} else if v, ok := c["id"].(string); ok {
					id = v
				}
				domain, _ := c["domain"].(string)
				status, _ := c["status"].(string)
				
				expires := "Never"
				if e, ok := c["expires_at"].(string); ok && e != "" {
					if len(e) >= 10 {
						expires = e[:10]
					} else {
						expires = e
					}
				}
				provider, _ := c["provider"].(string)

				tw.AddRow(id, domain, status, expires, provider)
			}
		}
		return tw.Render()
	}
}
