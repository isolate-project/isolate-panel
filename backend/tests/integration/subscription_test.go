package integration_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/isolate-project/isolate-panel/internal/api"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services"
	"github.com/isolate-project/isolate-panel/tests/testutil"
)

func setupSubscriptionApp(t *testing.T) (*fiber.App, string) {
	t.Helper()

	db := testutil.SetupTestDB(t)
	testutil.SeedTestData(t, db)

	// Create a test inbound and assign it to testuser1
	var core models.Core
	require.NoError(t, db.Where("name = ?", "xray").First(&core).Error)

	var user models.User
	require.NoError(t, db.Where("username = ?", "testuser1").First(&user).Error)

	inbound := &models.Inbound{
		Name:          "Test Inbound",
		Protocol:      "vless",
		CoreID:        core.ID,
		ListenAddress: "0.0.0.0",
		Port:          12345,
		IsEnabled:     true,
	}
	require.NoError(t, db.Create(inbound).Error)

	mapping := &models.UserInboundMapping{
		UserID:    user.ID,
		InboundID: inbound.ID,
	}
	require.NoError(t, db.Create(mapping).Error)

	// Create service and handler (without cache for simple testing)
	subscriptionService := services.NewSubscriptionService(db, "http://localhost")
	subscriptionsHandler := api.NewSubscriptionsHandler(subscriptionService)

	app := fiber.New()
	
	subscriptionRoutes := app.Group("/sub")
	subscriptionRoutes.Get("/:token", subscriptionsHandler.GetAutoDetectSubscription)
	subscriptionRoutes.Get("/:token/clash", subscriptionsHandler.GetClashSubscription)
	subscriptionRoutes.Get("/:token/singbox", subscriptionsHandler.GetSingboxSubscription)

	return app, user.SubscriptionToken
}

func TestSubscription_AutoDetectClash(t *testing.T) {
	app, token := setupSubscriptionApp(t)

	req := httptest.NewRequest(http.MethodGet, "/sub/"+token, nil)
	req.Header.Set("User-Agent", "ClashX Pro/1.0.0")
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/yaml; charset=utf-8", resp.Header.Get("Content-Type"))
	assert.Equal(t, "24", resp.Header.Get("Profile-Update-Interval"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.True(t, strings.Contains(string(body), "proxies:"))
	assert.True(t, strings.Contains(string(body), "port: 7890"))
}

func TestSubscription_AutoDetectSingbox(t *testing.T) {
	app, token := setupSubscriptionApp(t)

	req := httptest.NewRequest(http.MethodGet, "/sub/"+token, nil)
	req.Header.Set("User-Agent", "sing-box/1.8.0")
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var config map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &config))

	_, ok := config["outbounds"]
	assert.True(t, ok)
}

func TestSubscription_Base64Default(t *testing.T) {
	app, token := setupSubscriptionApp(t)

	req := httptest.NewRequest(http.MethodGet, "/sub/"+token, nil)
	req.Header.Set("User-Agent", "v2rayNG/1.8.5")
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/plain; charset=utf-8", resp.Header.Get("Content-Type"))

	// Body should be base64
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.NotEmpty(t, body)
}

func TestSubscription_ExplicitClash(t *testing.T) {
	app, token := setupSubscriptionApp(t)

	req := httptest.NewRequest(http.MethodGet, "/sub/"+token+"/clash", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/yaml; charset=utf-8", resp.Header.Get("Content-Type"))
}

func TestSubscription_InvalidToken(t *testing.T) {
	app, _ := setupSubscriptionApp(t)

	req := httptest.NewRequest(http.MethodGet, "/sub/invalid-token123", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
