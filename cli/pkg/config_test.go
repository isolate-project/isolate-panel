package pkg_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vovk4morkovk4/isolate-panel/cli/pkg"
)

func TestConfig(t *testing.T) {
	t.Run("creates default config", func(t *testing.T) {
		config := pkg.NewConfig()
		assert.NotNil(t, config)
		assert.Equal(t, "table", config.OutputFormat)
		assert.False(t, config.NoColor)
	})

	t.Run("loads config from file", func(t *testing.T) {
		// Create temp config file
		tmpFile := "/tmp/test-config.json"
		defer os.Remove(tmpFile)

		configData := `{
			"current_profile": "test",
			"profiles": {
				"test": {
					"url": "http://localhost:8080",
					"token": "test-token"
				}
			}
		}`

		err := os.WriteFile(tmpFile, []byte(configData), 0644)
		assert.NoError(t, err)

		config, err := pkg.LoadConfig(tmpFile)
		assert.NoError(t, err)
		assert.Equal(t, "test", config.CurrentProfile)
		assert.NotNil(t, config.Profiles["test"])
	})

	t.Run("saves config to file", func(t *testing.T) {
		tmpFile := "/tmp/test-save-config.json"
		defer os.Remove(tmpFile)

		config := pkg.NewConfig()
		config.CurrentProfile = "test"

		err := config.Save(tmpFile)
		assert.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(tmpFile)
		assert.NoError(t, err)
	})
}

func TestFormatter(t *testing.T) {
	t.Run("formats as table", func(t *testing.T) {
		formatter := pkg.NewTableFormatter()
		assert.NotNil(t, formatter)
	})

	t.Run("formats as JSON", func(t *testing.T) {
		formatter := pkg.NewJSONFormatter()
		assert.NotNil(t, formatter)
	})

	t.Run("formats as CSV", func(t *testing.T) {
		formatter := pkg.NewCSVFormatter()
		assert.NotNil(t, formatter)
	})
}

func TestExitCodes(t *testing.T) {
	t.Run("exit success is 0", func(t *testing.T) {
		assert.Equal(t, 0, pkg.ExitSuccess)
	})

	t.Run("exit general error is 1", func(t *testing.T) {
		assert.Equal(t, 1, pkg.ExitGeneralError)
	})

	t.Run("exit auth error is 2", func(t *testing.T) {
		assert.Equal(t, 2, pkg.ExitAuthError)
	})

	t.Run("exit not found is 3", func(t *testing.T) {
		assert.Equal(t, 3, pkg.ExitNotFound)
	})
}
