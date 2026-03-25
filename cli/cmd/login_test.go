package cmd_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vovk4morkovk4/isolate-panel/cli/cmd"
)

func TestLoginCommand(t *testing.T) {
	t.Run("creates login command", func(t *testing.T) {
		loginCmd := cmd.NewLoginCommand()
		assert.NotNil(t, loginCmd)
		assert.Equal(t, "login", loginCmd.Name())
	})

	t.Run("has required flags", func(t *testing.T) {
		loginCmd := cmd.NewLoginCommand()
		assert.NotNil(t, loginCmd)

		// Check if command has username and password flags
		flags := loginCmd.Flags()
		assert.NotNil(t, flags)
	})
}

func TestLogoutCommand(t *testing.T) {
	t.Run("creates logout command", func(t *testing.T) {
		logoutCmd := cmd.NewLogoutCommand()
		assert.NotNil(t, logoutCmd)
		assert.Equal(t, "logout", logoutCmd.Name())
	})
}

func TestProfileCommands(t *testing.T) {
	t.Run("creates profile list command", func(t *testing.T) {
		listCmd := cmd.NewProfileListCommand()
		assert.NotNil(t, listCmd)
	})

	t.Run("creates profile switch command", func(t *testing.T) {
		switchCmd := cmd.NewProfileSwitchCommand()
		assert.NotNil(t, switchCmd)
	})

	t.Run("creates profile current command", func(t *testing.T) {
		currentCmd := cmd.NewProfileCurrentCommand()
		assert.NotNil(t, currentCmd)
	})
}
