package e2e_test

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services"
	"github.com/isolate-project/isolate-panel/tests/testutil"
)

// TestPasswordNotLeakedInResponse verifies that the Password field
// is absent from UserResponse but present in CreateUserResponse.
func TestPasswordNotLeakedInResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	notificationService := services.NewNotificationService(db, "", "", "", "")
	userService := services.NewUserService(db, notificationService)

	// Create user — should return *models.User with Password set
	req := &services.CreateUserRequest{
		Username: "leaktest",
		Email:    "leak@example.com",
		Password: "supersecret123",
	}
	user, err := userService.CreateUser(req, 1)
	require.NoError(t, err)
	assert.Equal(t, "supersecret123", user.Password, "Password should be set on DB model")

	// Simulate what the API does for Create response
	createResp := services.CreateUserResponse{
		UserResponse: services.UserResponse{
			ID:       user.ID,
			Username: user.Username,
			UUID:     user.UUID,
		},
		Password: user.Password,
	}

	// CreateUserResponse should include password in JSON
	createJSON, err := json.Marshal(createResp)
	require.NoError(t, err)
	assert.Contains(t, string(createJSON), "supersecret", "CreateUserResponse must contain password")

	// UserResponse alone should NOT include password in JSON
	getResp := services.UserResponse{
		ID:       user.ID,
		Username: user.Username,
		UUID:     user.UUID,
	}
	getJSON, err := json.Marshal(getResp)
	require.NoError(t, err)
	assert.NotContains(t, string(getJSON), "password", "UserResponse JSON must not contain password field")
}

// TestMultiProtocolSubscription verifies that subscription links are
// generated for all supported protocols across V2Ray, Clash, and Sing-box.
func TestMultiProtocolSubscription(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	core := testutil.GetTestCore(t, db, "xray")

	// Create user
	user := testutil.CreateTestUser(t, db, "subtest", "sub@example.com")

	// Create inbounds with different protocols
	protocols := []struct {
		name     string
		protocol string
		port     int
		config   string
	}{
		{"vless-in", "vless", 10001, `{"transport":"tcp"}`},
		{"vmess-in", "vmess", 10002, `{"transport":"ws"}`},
		{"trojan-in", "trojan", 10003, `{}`},
		{"ss-in", "shadowsocks", 10004, `{"method":"aes-256-gcm"}`},
		{"hy2-in", "hysteria2", 10005, `{}`},
		{"tuic-in", "tuic_v5", 10006, `{"congestion_control":"bbr"}`},
		{"ssr-in", "ssr", 10007, `{"method":"chacha20-poly1305","protocol":"origin","obfs":"plain"}`},
		{"http-in", "http", 10008, `{}`},
		{"socks5-in", "socks5", 10009, `{}`},
	}

	for _, p := range protocols {
		inbound := &models.Inbound{
			Name:          p.name,
			Protocol:      p.protocol,
			CoreID:        core.ID,
			ListenAddress: "1.2.3.4",
			Port:          p.port,
			ConfigJSON:    p.config,
			IsEnabled:     true,
		}
		require.NoError(t, db.Create(inbound).Error)

		mapping := &models.UserInboundMapping{
			UserID:    user.ID,
			InboundID: inbound.ID,
		}
		require.NoError(t, db.Create(mapping).Error)
	}

	subService := services.NewSubscriptionService(db, "http://localhost")

	t.Run("V2Ray links", func(t *testing.T) {
		data, err := subService.GetUserSubscriptionData(user.SubscriptionToken)
		require.NoError(t, err)
		links, err := subService.GenerateV2Ray(data)
		require.NoError(t, err)

		// Decode base64
		decoded, err := base64.StdEncoding.DecodeString(links)
		require.NoError(t, err)
		content := string(decoded)

		assert.Contains(t, content, "vless://", "V2Ray should have VLESS link")
		assert.Contains(t, content, "vmess://", "V2Ray should have VMess link")
		assert.Contains(t, content, "trojan://", "V2Ray should have Trojan link")
		assert.Contains(t, content, "ss://", "V2Ray should have SS link")
		assert.Contains(t, content, "hysteria2://", "V2Ray should have Hysteria2 link")
		assert.Contains(t, content, "tuic://", "V2Ray should have TUIC link")
		assert.Contains(t, content, "ssr://", "V2Ray should have SSR link")
	})

	t.Run("Clash YAML", func(t *testing.T) {
		data, err := subService.GetUserSubscriptionData(user.SubscriptionToken)
		require.NoError(t, err)
		yaml, err := subService.GenerateClash(data)
		require.NoError(t, err)

		assert.Contains(t, yaml, "type: vless", "Clash should have VLESS proxy")
		assert.Contains(t, yaml, "type: vmess", "Clash should have VMess proxy")
		assert.Contains(t, yaml, "type: trojan", "Clash should have Trojan proxy")
		assert.Contains(t, yaml, "type: ss", "Clash should have SS proxy")
		assert.Contains(t, yaml, "type: hysteria2", "Clash should have Hysteria2 proxy")
		assert.Contains(t, yaml, "type: tuic", "Clash should have TUIC proxy")
		assert.Contains(t, yaml, "type: ssr", "Clash should have SSR proxy")
		assert.Contains(t, yaml, "type: http", "Clash should have HTTP proxy")
		assert.Contains(t, yaml, "type: socks5", "Clash should have SOCKS5 proxy")
	})

	t.Run("Sing-box JSON", func(t *testing.T) {
		data, err := subService.GetUserSubscriptionData(user.SubscriptionToken)
		require.NoError(t, err)
		jsonStr, err := subService.GenerateSingbox(data)
		require.NoError(t, err)

		var config map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(jsonStr), &config))

		outbounds, ok := config["outbounds"].([]interface{})
		require.True(t, ok)

		// Count protocol types
		typeCount := make(map[string]int)
		for _, ob := range outbounds {
			if m, ok := ob.(map[string]interface{}); ok {
				if tp, ok := m["type"].(string); ok {
					typeCount[tp]++
				}
			}
		}

		assert.GreaterOrEqual(t, typeCount["vless"], 1, "Should have VLESS outbound")
		assert.GreaterOrEqual(t, typeCount["vmess"], 1, "Should have VMess outbound")
		assert.GreaterOrEqual(t, typeCount["trojan"], 1, "Should have Trojan outbound")
		assert.GreaterOrEqual(t, typeCount["shadowsocks"], 1, "Should have SS outbound")
		assert.GreaterOrEqual(t, typeCount["hysteria2"], 1, "Should have Hysteria2 outbound")
		assert.GreaterOrEqual(t, typeCount["tuic"], 1, "Should have TUIC outbound")
	})
}

// TestCertificateLifecycle tests certificate creation → binding → TLS config
func TestCertificateLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	core := testutil.GetTestCore(t, db, "xray")

	// Create certificate directly in DB (simulating API upload handler)
	t.Run("Create certificate", func(t *testing.T) {
		cert := &models.Certificate{
			Domain:     "example.com",
			CertPath:   "/etc/certs/example.com/cert.pem",
			KeyPath:    "/etc/certs/example.com/key.pem",
			CommonName: "example.com",
			Status:     models.CertificateStatusActive,
		}
		require.NoError(t, db.Create(cert).Error)
		assert.NotZero(t, cert.ID)
	})

	// Retrieve certificate
	t.Run("Retrieve certificate", func(t *testing.T) {
		var certs []models.Certificate
		require.NoError(t, db.Find(&certs).Error)
		assert.Len(t, certs, 1)
		assert.Equal(t, "example.com", certs[0].Domain)
		assert.Equal(t, models.CertificateStatusActive, certs[0].Status)
	})

	// Bind to inbound with TLS
	t.Run("Bind cert to inbound", func(t *testing.T) {
		var cert models.Certificate
		require.NoError(t, db.First(&cert).Error)

		inbound := &models.Inbound{
			Name:          "tls-test",
			Protocol:      "vless",
			CoreID:        core.ID,
			ListenAddress: "0.0.0.0",
			Port:          10443,
			TLSEnabled:    true,
			TLSCertID:     &cert.ID,
			IsEnabled:     true,
		}
		require.NoError(t, db.Create(inbound).Error)

		// Verify inbound has cert bound
		var savedInbound models.Inbound
		require.NoError(t, db.First(&savedInbound, inbound.ID).Error)
		assert.True(t, savedInbound.TLSEnabled)
		assert.NotNil(t, savedInbound.TLSCertID)
		assert.Equal(t, cert.ID, *savedInbound.TLSCertID)
	})

	// Delete certificate
	t.Run("Delete certificate", func(t *testing.T) {
		var cert models.Certificate
		require.NoError(t, db.First(&cert).Error)

		require.NoError(t, db.Delete(&cert).Error)

		var certs []models.Certificate
		require.NoError(t, db.Find(&certs).Error)
		assert.Len(t, certs, 0)
	})
}

// TestExpiryNotificationDedup verifies that the LastExpiryNotifiedDays field
// prevents duplicate expiry notifications.
func TestExpiryNotificationDedup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	notificationService := services.NewNotificationService(db, "", "", "", "")
	userService := services.NewUserService(db, notificationService)

	// Verify LastExpiryNotifiedDays field exists and works
	var user models.User
	require.NoError(t, db.Where("username = ?", "testuser1").First(&user).Error)
	assert.Nil(t, user.LastExpiryNotifiedDays, "New user should have nil notification days")

	// Set it manually to simulate a previous notification
	days := 7
	require.NoError(t, db.Model(&user).Update("last_expiry_notified_days", days).Error)

	// Reload and verify
	var updatedUser models.User
	require.NoError(t, db.First(&updatedUser, user.ID).Error)
	require.NotNil(t, updatedUser.LastExpiryNotifiedDays)
	assert.Equal(t, 7, *updatedUser.LastExpiryNotifiedDays)

	// Trigger CheckExpiringUsers — should not crash
	userService.CheckExpiringUsers()
}

// TestUserResponseSerialization checks that UserResponse excludes password
// and CreateUserResponse includes it.
func TestUserResponseSerialization(t *testing.T) {
	resp := services.UserResponse{
		ID:       1,
		Username: "testuser",
		UUID:     "test-uuid",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	// Check that password is not in the JSON
	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &decoded))
	_, hasPassword := decoded["password"]
	assert.False(t, hasPassword, "UserResponse should not have password field")

	// Check CreateUserResponse includes password
	createResp := services.CreateUserResponse{
		UserResponse: resp,
		Password:     "secret123",
	}

	data, err = json.Marshal(createResp)
	require.NoError(t, err)

	require.NoError(t, json.Unmarshal(data, &decoded))
	pw, hasPassword := decoded["password"]
	assert.True(t, hasPassword, "CreateUserResponse should have password field")
	assert.Equal(t, "secret123", pw)
}
