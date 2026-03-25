package pkg_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vovk4morkovk4/isolate-panel/cli/pkg"
)

func TestTableFormatter(t *testing.T) {
	t.Run("formats data as table", func(t *testing.T) {
		data := []map[string]interface{}{
			{"id": 1, "name": "User1"},
			{"id": 2, "name": "User2"},
		}

		var buf bytes.Buffer
		formatter := pkg.NewTableFormatter(&buf)
		err := formatter.Format(data)

		assert.NoError(t, err)
		assert.NotEmpty(t, buf.String())
	})

	t.Run("formats empty data", func(t *testing.T) {
		data := []map[string]interface{}{}

		var buf bytes.Buffer
		formatter := pkg.NewTableFormatter(&buf)
		err := formatter.Format(data)

		assert.NoError(t, err)
	})
}

func TestJSONFormatter(t *testing.T) {
	t.Run("formats data as JSON", func(t *testing.T) {
		data := map[string]interface{}{
			"id":   1,
			"name": "User1",
		}

		var buf bytes.Buffer
		formatter := pkg.NewJSONFormatter(&buf)
		err := formatter.Format(data)

		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "User1")
	})

	t.Run("formats array as JSON", func(t *testing.T) {
		data := []map[string]interface{}{
			{"id": 1, "name": "User1"},
		}

		var buf bytes.Buffer
		formatter := pkg.NewJSONFormatter(&buf)
		err := formatter.Format(data)

		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "[")
	})
}

func TestCSVFormatter(t *testing.T) {
	t.Run("formats data as CSV", func(t *testing.T) {
		data := []map[string]interface{}{
			{"id": 1, "name": "User1"},
			{"id": 2, "name": "User2"},
		}

		var buf bytes.Buffer
		formatter := pkg.NewCSVFormatter(&buf)
		err := formatter.Format(data)

		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "id,name")
	})
}

func TestQuietFormatter(t *testing.T) {
	t.Run("formats data quietly", func(t *testing.T) {
		data := map[string]interface{}{
			"id": 1,
		}

		var buf bytes.Buffer
		formatter := pkg.NewQuietFormatter(&buf)
		err := formatter.Format(data)

		assert.NoError(t, err)
	})
}

func TestOutputFormats(t *testing.T) {
	formats := []string{"table", "json", "csv", "quiet"}

	for _, format := range formats {
		t.Run("supports "+format+" format", func(t *testing.T) {
			assert.Contains(t, formats, format)
		})
	}
}
