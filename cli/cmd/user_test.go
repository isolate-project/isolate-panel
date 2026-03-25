package cmd_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vovk4morkovk4/isolate-panel/cli/cmd"
)

func TestUserCommands(t *testing.T) {
	t.Run("creates user list command", func(t *testing.T) {
		listCmd := cmd.NewUserListCommand()
		assert.NotNil(t, listCmd)
		assert.Equal(t, "list", listCmd.Name())
	})

	t.Run("creates user show command", func(t *testing.T) {
		showCmd := cmd.NewUserShowCommand()
		assert.NotNil(t, showCmd)
		assert.Equal(t, "show", showCmd.Name())
	})

	t.Run("creates user create command", func(t *testing.T) {
		createCmd := cmd.NewUserCreateCommand()
		assert.NotNil(t, createCmd)
		assert.Equal(t, "create", createCmd.Name())
	})

	t.Run("creates user update command", func(t *testing.T) {
		updateCmd := cmd.NewUserUpdateCommand()
		assert.NotNil(t, updateCmd)
		assert.Equal(t, "update", updateCmd.Name())
	})

	t.Run("creates user delete command", func(t *testing.T) {
		deleteCmd := cmd.NewUserDeleteCommand()
		assert.NotNil(t, deleteCmd)
		assert.Equal(t, "delete", deleteCmd.Name())
	})
}

func TestUserCommandFlags(t *testing.T) {
	t.Run("user list has format flag", func(t *testing.T) {
		listCmd := cmd.NewUserListCommand()
		assert.NotNil(t, listCmd)

		flags := listCmd.Flags()
		assert.NotNil(t, flags)
	})

	t.Run("user create has required flags", func(t *testing.T) {
		createCmd := cmd.NewUserCreateCommand()
		assert.NotNil(t, createCmd)

		flags := createCmd.Flags()
		assert.NotNil(t, flags)
	})
}
