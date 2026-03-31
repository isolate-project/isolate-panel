package logger

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit_StdoutJSON(t *testing.T) {
	cfg := &Config{
		Level:  "debug",
		Format: "json",
		Output: "stdout",
	}
	err := Init(cfg)
	require.NoError(t, err)
	assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())
}

func TestInit_StdoutConsole(t *testing.T) {
	cfg := &Config{
		Level:  "info",
		Format: "console",
		Output: "stdout",
	}
	err := Init(cfg)
	require.NoError(t, err)
	assert.Equal(t, zerolog.InfoLevel, zerolog.GlobalLevel())
}

func TestInit_InvalidLevel_DefaultsToInfo(t *testing.T) {
	cfg := &Config{
		Level:  "not-a-level",
		Format: "json",
		Output: "stdout",
	}
	err := Init(cfg)
	require.NoError(t, err)
	assert.Equal(t, zerolog.InfoLevel, zerolog.GlobalLevel())
}

func TestInit_FileOutput(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "logs", "test.log")

	cfg := &Config{
		Level:      "warn",
		Format:     "json",
		Output:     "file",
		FilePath:   logPath,
		MaxSize:    10,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   false,
	}
	err := Init(cfg)
	require.NoError(t, err)

	// Log directory must be created
	_, statErr := os.Stat(filepath.Dir(logPath))
	assert.NoError(t, statErr)
}

func TestInit_BothOutputs(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "both.log")

	cfg := &Config{
		Level:      "error",
		Format:     "json",
		Output:     "both",
		FilePath:   logPath,
		MaxSize:    1,
		MaxBackups: 1,
		MaxAge:     1,
		Compress:   false,
	}
	err := Init(cfg)
	require.NoError(t, err)
}

func TestInit_NoOutput_DefaultsToStdout(t *testing.T) {
	// Output="" falls through to default (stdout)
	cfg := &Config{
		Level:  "info",
		Format: "json",
		Output: "none",
	}
	err := Init(cfg)
	require.NoError(t, err)
}

func TestWithComponent(t *testing.T) {
	// Initialize with a buffer writer so we can capture output
	var buf bytes.Buffer
	Log = zerolog.New(&buf).With().Timestamp().Logger()

	l := WithComponent("test-service")
	l.Info().Msg("component test")

	out := buf.String()
	assert.True(t, strings.Contains(out, "test-service"), "expected component field in log output")
}

func TestWithRequestID(t *testing.T) {
	var buf bytes.Buffer
	Log = zerolog.New(&buf).With().Timestamp().Logger()

	l := WithRequestID("req-abc-123")
	l.Info().Msg("request test")

	out := buf.String()
	assert.True(t, strings.Contains(out, "req-abc-123"), "expected request_id in log output")
}

func TestInit_FileOutput_InvalidPath(t *testing.T) {
	// /proc/invalid is not writable — MkdirAll should fail
	cfg := &Config{
		Level:    "info",
		Format:   "json",
		Output:   "file",
		FilePath: "/proc/invalid_isolate_test_dir/app.log",
		MaxSize:  1,
	}
	err := Init(cfg)
	assert.Error(t, err)
}
