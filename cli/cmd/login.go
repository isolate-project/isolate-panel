package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/vovk4morkovk4/isolate-panel/cli/pkg"
)

var (
	loginURL      string
	loginUsername string
	loginPassword string
)

// LoginCmd returns the login command
func LoginCmd() *cobra.Command {
	return loginCmd
}

var loginCmd = &cobra.Command{
	Use:   "login [profile-name]",
	Short: "Login to Isolate Panel",
	Long:  `Login to an Isolate Panel instance. If profile name is not provided, uses "default" profile.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runLogin,
}

func init() {
	loginCmd.Flags().StringVarP(&loginURL, "url", "u", "http://localhost:8080", "Panel URL")
	loginCmd.Flags().StringVarP(&loginUsername, "username", "n", "", "Username (non-interactive mode)")
	loginCmd.Flags().StringVarP(&loginPassword, "password", "p", "", "Password (non-interactive mode)")
}

func runLogin(cmd *cobra.Command, args []string) error {
	profileName := "default"
	if len(args) > 0 {
		profileName = args[0]
	}

	// Load existing config
	config, err := pkg.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get credentials
	username := loginUsername
	password := loginPassword

	// Interactive mode if credentials not provided
	if username == "" || password == "" {
		reader := bufio.NewReader(os.Stdin)

		if username == "" {
			fmt.Print("Username: ")
			username, _ = reader.ReadString('\n')
			username = strings.TrimSpace(username)
		}

		if password == "" {
			fmt.Print("Password: ")
			passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}
			fmt.Println() // New line after password input
			password = string(passwordBytes)
		}
	}

	if username == "" || password == "" {
		return fmt.Errorf("username and password are required")
	}

	// Perform login
	baseURL := strings.TrimSuffix(loginURL, "/")
	loginURL := baseURL + "/api/auth/login"

	loginReq := map[string]string{
		"username": username,
		"password": password,
	}

	jsonData, err := json.Marshal(loginReq)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(loginURL, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		var errorResp map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return fmt.Errorf("login failed with status %d", resp.StatusCode)
		}
		return fmt.Errorf("login failed: %s", errorResp["error"])
	}

	var loginResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	// Save profile
	profile := pkg.Profile{
		PanelURL:       baseURL,
		Username:       username,
		AccessToken:    loginResp.AccessToken,
		RefreshToken:   loginResp.RefreshToken,
		TokenExpiresAt: time.Now().Add(time.Duration(loginResp.ExpiresIn) * time.Second).Format(time.RFC3339),
	}

	config.SetProfile(profileName, profile)
	config.SetCurrentProfile(profileName)

	if err := config.SaveConfig(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Logged in successfully to %s as %s\n", baseURL, username)
	fmt.Printf("  Profile: %s\n", profileName)

	return nil
}
