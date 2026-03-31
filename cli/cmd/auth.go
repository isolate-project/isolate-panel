package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/isolate-project/isolate-panel/cli/pkg"
)

var logoutCmd = &cobra.Command{
	Use:   "logout [profile-name]",
	Short: "Logout from Isolate Panel",
	Long:  `Logout from an Isolate Panel instance. Removes the profile or clears tokens.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runLogout,
}

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Profile management commands",
	Long:  `Manage Isolate Panel profiles.`,
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	Long:  `List all configured profiles.`,
	RunE:  runProfileList,
}

var profileSwitchCmd = &cobra.Command{
	Use:   "switch <profile-name>",
	Short: "Switch to a profile",
	Long:  `Switch to a different profile.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runProfileSwitch,
}

var profileCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show current profile",
	Long:  `Show the currently active profile.`,
	RunE:  runProfileCurrent,
}

var profileDeleteCmd = &cobra.Command{
	Use:   "delete <profile-name>",
	Short: "Delete a profile",
	Long:  `Delete a profile.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runProfileDelete,
}

func init() {
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileSwitchCmd)
	profileCmd.AddCommand(profileCurrentCmd)
	profileCmd.AddCommand(profileDeleteCmd)
}

// LogoutCmd returns the logout command
func LogoutCmd() *cobra.Command {
	return logoutCmd
}

// ProfileCmd returns the profile command
func ProfileCmd() *cobra.Command {
	return profileCmd
}

func runLogout(cmd *cobra.Command, args []string) error {
	profileName := "default"
	if len(args) > 0 {
		profileName = args[0]
	}

	config, err := pkg.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if _, ok := config.Profiles[profileName]; !ok {
		return fmt.Errorf("profile '%s' not found", profileName)
	}

	config.DeleteProfile(profileName)
	if err := config.SaveConfig(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Logged out from profile '%s'\n", profileName)
	return nil
}

func runProfileList(cmd *cobra.Command, args []string) error {
	config, err := pkg.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	profiles := config.ListProfiles()
	if len(profiles) == 0 {
		fmt.Println("No profiles configured. Use 'isolate-panel login' to create one.")
		return nil
	}

	fmt.Printf("%-20s %-40s %s\n", "PROFILE", "URL", "USERNAME")
	fmt.Println("-------------------------------------------------------------")

	for _, name := range profiles {
		profile := config.Profiles[name]
		current := ""
		if name == config.CurrentProfile {
			current = " (current)"
		}
		fmt.Printf("%-20s %-40s %s%s\n", name, profile.PanelURL, profile.Username, current)
	}

	return nil
}

func runProfileSwitch(cmd *cobra.Command, args []string) error {
	profileName := args[0]

	config, err := pkg.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if _, ok := config.Profiles[profileName]; !ok {
		return fmt.Errorf("profile '%s' not found", profileName)
	}

	config.SetCurrentProfile(profileName)
	if err := config.SaveConfig(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Switched to profile '%s'\n", profileName)
	return nil
}

func runProfileCurrent(cmd *cobra.Command, args []string) error {
	config, err := pkg.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	profile, err := config.GetCurrentProfile()
	if err != nil {
		return fmt.Errorf("no current profile set")
	}

	fmt.Printf("Current profile: %s\n", config.CurrentProfile)
	fmt.Printf("URL: %s\n", profile.PanelURL)
	fmt.Printf("Username: %s\n", profile.Username)
	return nil
}

func runProfileDelete(cmd *cobra.Command, args []string) error {
	profileName := args[0]

	config, err := pkg.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if _, ok := config.Profiles[profileName]; !ok {
		return fmt.Errorf("profile '%s' not found", profileName)
	}

	config.DeleteProfile(profileName)
	if err := config.SaveConfig(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Profile '%s' deleted\n", profileName)
	return nil
}
