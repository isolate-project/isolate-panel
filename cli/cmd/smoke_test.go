package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLoginCmd_ReturnsCommand tests that LoginCmd returns a valid cobra command
func TestLoginCmd_ReturnsCommand(t *testing.T) {
	cmd := LoginCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "login")
	assert.NotEmpty(t, cmd.Short)
}

// TestLogoutCmd_ReturnsCommand tests that LogoutCmd returns a valid cobra command
func TestLogoutCmd_ReturnsCommand(t *testing.T) {
	cmd := LogoutCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "logout")
}

// TestProfileCmd_ReturnsCommand tests that ProfileCmd returns a valid cobra command
func TestProfileCmd_ReturnsCommand(t *testing.T) {
	cmd := ProfileCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "profile", cmd.Use)
}

// TestUserCmd_ReturnsCommand tests that UserCmd returns a valid cobra command
func TestUserCmd_ReturnsCommand(t *testing.T) {
	cmd := UserCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "user", cmd.Use)
}

// TestInboundCmd_ReturnsCommand tests that InboundCmd returns a valid cobra command
func TestInboundCmd_ReturnsCommand(t *testing.T) {
	cmd := InboundCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "inbound", cmd.Use)
}

// TestOutboundCmd_ReturnsCommand tests that OutboundCmd returns a valid cobra command
func TestOutboundCmd_ReturnsCommand(t *testing.T) {
	cmd := OutboundCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "outbound", cmd.Use)
}

// TestCoreCmd_ReturnsCommand tests that CoreCmd returns a valid cobra command
func TestCoreCmd_ReturnsCommand(t *testing.T) {
	cmd := CoreCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "core", cmd.Use)
}

// TestBackupCmd_ReturnsCommand tests that BackupCmd returns a valid cobra command
func TestBackupCmd_ReturnsCommand(t *testing.T) {
	cmd := BackupCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "backup", cmd.Use)
}

// TestStatsCmd_ReturnsCommand tests that StatsCmd returns a valid cobra command
func TestStatsCmd_ReturnsCommand(t *testing.T) {
	cmd := StatsCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "stats", cmd.Use)
}

// TestCertCmd_ReturnsCommand tests that CertCmd returns a valid cobra command
func TestCertCmd_ReturnsCommand(t *testing.T) {
	cmd := CertCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "cert", cmd.Use)
}

// TestCompletionCmd_ReturnsCommand tests that CompletionCmd returns a valid cobra command
func TestCompletionCmd_ReturnsCommand(t *testing.T) {
	cmd := CompletionCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "completion")
}
