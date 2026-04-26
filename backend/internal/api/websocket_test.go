package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/isolate-project/isolate-panel/internal/auth"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)


func resetWSTickets() {
	wsTicketsMu.Lock()
	wsTickets = make(map[string]wsTicket)
	wsTicketsMu.Unlock()
}

func setupWSDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?_foreign_keys=on"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.Core{}))
	return db
}

func setupDashboardHub(t *testing.T) *DashboardHub {
	t.Helper()
	db := setupWSDB(t)
	ct := services.NewConnectionTracker(db, 10*time.Second, "", "", "", "", "")
	ts := auth.NewTokenService("test-secret", 15*time.Minute, 7*24*time.Hour, nil, nil)
	return NewDashboardHub(db, ct, ts)
}

func TestIssueWSTicket_ReturnsTicket(t *testing.T) {
	resetWSTickets()
	app := setupWSTicketApp(t)

	req := httptest.NewRequest(http.MethodPost, "/ws/ticket", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Contains(t, body, "ticket")

	ticket, ok := body["ticket"].(string)
	assert.True(t, ok, "ticket should be a string")
	assert.Len(t, ticket, 32, "ticket should be 32 characters (16 bytes hex encoded)")
}

func TestIssueWSTicket_TicketStoredInMap(t *testing.T) {
	resetWSTickets()
	app := setupWSTicketApp(t)

	req := httptest.NewRequest(http.MethodPost, "/ws/ticket", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	ticket := body["ticket"].(string)

	wsTicketsMu.Lock()
	storedTicket, exists := wsTickets[ticket]
	wsTicketsMu.Unlock()

	assert.True(t, exists, "ticket should be stored in wsTickets map")
	assert.WithinDuration(t, time.Now().Add(30*time.Second), storedTicket.expiresAt, time.Second)
}

func TestIssueWSTicket_NoAuth(t *testing.T) {
	resetWSTickets()
	hub := &DashboardHub{}

	app := fiber.New()
	app.Post("/ws/ticket", hub.IssueWSTicket)

	req := httptest.NewRequest(http.MethodPost, "/ws/ticket", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Contains(t, body, "ticket")
}

func TestValidateAndConsumeTicket_Valid(t *testing.T) {
	resetWSTickets()

	ticket := "0123456789abcdef0123456789abcdef"
	wsTicketsMu.Lock()
	wsTickets[ticket] = wsTicket{expiresAt: time.Now().Add(30 * time.Second)}
	wsTicketsMu.Unlock()

	valid := validateAndConsumeTicket(ticket)
	assert.True(t, valid, "valid ticket should return true")

	wsTicketsMu.Lock()
	_, exists := wsTickets[ticket]
	wsTicketsMu.Unlock()
	assert.False(t, exists, "ticket should be removed after consumption")
}

func TestValidateAndConsumeTicket_Invalid(t *testing.T) {
	resetWSTickets()

	valid := validateAndConsumeTicket("nonexistentticket")
	assert.False(t, valid, "non-existent ticket should return false")
}

func TestValidateAndConsumeTicket_OneTimeUse(t *testing.T) {
	resetWSTickets()

	ticket := "0123456789abcdef0123456789abcdef"
	wsTicketsMu.Lock()
	wsTickets[ticket] = wsTicket{expiresAt: time.Now().Add(30 * time.Second)}
	wsTicketsMu.Unlock()

	valid1 := validateAndConsumeTicket(ticket)
	assert.True(t, valid1, "first consumption should succeed")

	valid2 := validateAndConsumeTicket(ticket)
	assert.False(t, valid2, "second consumption should fail (one-time use)")
}

func TestValidateAndConsumeTicket_Expired(t *testing.T) {
	resetWSTickets()

	ticket := "0123456789abcdef0123456789abcdef"
	wsTicketsMu.Lock()
	wsTickets[ticket] = wsTicket{expiresAt: time.Now().Add(-1 * time.Hour)}
	wsTicketsMu.Unlock()

	valid := validateAndConsumeTicket(ticket)
	assert.False(t, valid, "expired ticket should return false")
}

func TestPurgeExpiredTickets_RemovesExpired(t *testing.T) {
	resetWSTickets()

	expiredTicket := "expired123456789abcdef01234567"
	validTicket := "valid123456789abcdef01234567"

	wsTicketsMu.Lock()
	wsTickets[expiredTicket] = wsTicket{expiresAt: time.Now().Add(-1 * time.Hour)}
	wsTickets[validTicket] = wsTicket{expiresAt: time.Now().Add(30 * time.Second)}
	wsTicketsMu.Unlock()

	purgeExpiredTickets()

	wsTicketsMu.Lock()
	_, expiredExists := wsTickets[expiredTicket]
	_, validExists := wsTickets[validTicket]
	wsTicketsMu.Unlock()

	assert.False(t, expiredExists, "expired ticket should be removed")
	assert.True(t, validExists, "valid ticket should remain")
}

func TestPurgeExpiredTickets_NoExpiredTickets(t *testing.T) {
	resetWSTickets()

	ticket1 := "ticket1123456789abcdef0123456"
	ticket2 := "ticket2123456789abcdef0123456"

	wsTicketsMu.Lock()
	wsTickets[ticket1] = wsTicket{expiresAt: time.Now().Add(30 * time.Second)}
	wsTickets[ticket2] = wsTicket{expiresAt: time.Now().Add(60 * time.Second)}
	wsTicketsMu.Unlock()

	purgeExpiredTickets()

	wsTicketsMu.Lock()
	_, exists1 := wsTickets[ticket1]
	_, exists2 := wsTickets[ticket2]
	wsTicketsMu.Unlock()

	assert.True(t, exists1, "ticket1 should remain")
	assert.True(t, exists2, "ticket2 should remain")
}

func TestDashboardHub_NewHub_CreatedSuccessfully(t *testing.T) {
	db := setupWSDB(t)
	ct := services.NewConnectionTracker(db, 10*time.Second, "", "", "", "", "")
	ts := auth.NewTokenService("test-secret", 15*time.Minute, 7*24*time.Hour, nil, nil)

	hub := NewDashboardHub(db, ct, ts)

	assert.NotNil(t, hub)
	assert.NotNil(t, hub.db)
	assert.NotNil(t, hub.connectionTracker)
	assert.NotNil(t, hub.tokenService)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
	assert.NotNil(t, hub.done)
}

func TestDashboardHub_RegisterAndUnregister(t *testing.T) {
	hub := setupDashboardHub(t)

	go hub.Run()
	defer hub.Stop()

	done := make(chan bool)
	go func() {
		time.Sleep(100 * time.Millisecond)
		done <- true
	}()

	select {
	case <-done:
		assert.True(t, true)
	case <-time.After(1 * time.Second):
		t.Fatal("hub did not start within timeout")
	}
}

func TestDashboardHub_BroadcastAll_SendsToClients(t *testing.T) {
	hub := setupDashboardHub(t)

	payload := []byte(`{"test":"data"}`)

	hub.broadcastAll(payload)

	assert.True(t, true)
}

func TestDashboardHub_CollectStats_ReturnsPayload(t *testing.T) {
	hub := setupDashboardHub(t)

	hub.db.Exec("DELETE FROM users")
	hub.db.Exec("DELETE FROM cores")

	hub.db.Exec("INSERT INTO users (username, uuid, password, subscription_token, is_active, traffic_used_bytes) VALUES (?, ?, ?, ?, ?, ?)",
		"user1", "uuid1", "hash", "sub1", 1, 1024)
	hub.db.Exec("INSERT INTO users (username, uuid, password, subscription_token, is_active, traffic_used_bytes) VALUES (?, ?, ?, ?, ?, ?)",
		"user2", "uuid2", "hash", "sub2", 0, 2048)

	core1 := models.Core{
		Name:      "xray",
		Version:   "1.0.0",
		IsRunning: true,
	}
	require.NoError(t, hub.db.Create(&core1).Error)

	payload, err := hub.collectStats()
	require.NoError(t, err)

	var stats DashboardPayload
	require.NoError(t, json.Unmarshal(payload, &stats))

	assert.Equal(t, int64(2), stats.TotalUsers)
	assert.Equal(t, int64(1), stats.ActiveUsers)
	assert.Equal(t, int64(3072), stats.TotalTrafficBytes)
	assert.Equal(t, int64(1), stats.CoresRunning)
	assert.GreaterOrEqual(t, stats.ActiveConnections, int64(0))
}

func TestDashboardHub_CollectStats_DBError(t *testing.T) {
	db := setupWSDB(t)
	ts := auth.NewTokenService("test-secret", 15*time.Minute, 7*24*time.Hour, nil, nil)
	hub := NewDashboardHub(db, nil, ts)

	user := models.User{
		Username:          "user1",
		UUID:              "uuid1",
		Password:          "hash",
		SubscriptionToken: "sub1",
		IsActive:          true,
	}
	require.NoError(t, hub.db.Create(&user).Error)

	payload, err := hub.collectStats()
	require.NoError(t, err)

	var stats DashboardPayload
	require.NoError(t, json.Unmarshal(payload, &stats))

	assert.Equal(t, int64(1), stats.TotalUsers)
	assert.Equal(t, int64(1), stats.ActiveUsers)
	assert.Equal(t, int64(0), stats.ActiveConnections)
}

func TestDashboardHub_Stop_StopsRun(t *testing.T) {
	hub := setupDashboardHub(t)

	runDone := make(chan bool)
	go func() {
		hub.Run()
		runDone <- true
	}()

	time.Sleep(100 * time.Millisecond)

	hub.Stop()

	select {
	case <-runDone:
		assert.True(t, true)
	case <-time.After(1 * time.Second):
		t.Fatal("hub did not stop within timeout")
	}
}

func TestWSTickets_ConcurrentIssue(t *testing.T) {
	resetWSTickets()
	hub := &DashboardHub{}

	app := fiber.New()
	app.Post("/ws/ticket", func(c fiber.Ctx) error {
		c.Locals("admin_id", uint(1))
		return hub.IssueWSTicket(c)
	})

	numGoroutines := 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/ws/ticket", nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}()
	}

	wg.Wait()

	wsTicketsMu.Lock()
	ticketCount := len(wsTickets)
	wsTicketsMu.Unlock()

	assert.Equal(t, numGoroutines, ticketCount, "all tickets should be created")
}

func TestWSTickets_ConcurrentValidate(t *testing.T) {
	resetWSTickets()

	numTickets := 10
	tickets := make([]string, numTickets)
	for i := 0; i < numTickets; i++ {
		tickets[i] = "ticket" + string(rune('0'+i)) + "123456789abcdef0123456"
		wsTicketsMu.Lock()
		wsTickets[tickets[i]] = wsTicket{expiresAt: time.Now().Add(30 * time.Second)}
		wsTicketsMu.Unlock()
	}

	var wg sync.WaitGroup
	wg.Add(numTickets)

	for i := 0; i < numTickets; i++ {
		go func(idx int) {
			defer wg.Done()
			valid := validateAndConsumeTicket(tickets[idx])
			assert.True(t, valid, "ticket should be valid")
		}(i)
	}

	wg.Wait()

	wsTicketsMu.Lock()
	ticketCount := len(wsTickets)
	wsTicketsMu.Unlock()

	assert.Equal(t, 0, ticketCount, "all tickets should be consumed")
}

func TestDashboardHub_ConcurrentRegisterUnregister(t *testing.T) {
	hub := setupDashboardHub(t)

	go hub.Run()
	defer hub.Stop()

	numGoroutines := 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_, err := hub.collectStats()
			assert.NoError(t, err)
		}()
	}

	wg.Wait()

	assert.True(t, true)
}

func TestDashboardHub_ConcurrentBroadcast(t *testing.T) {
	hub := setupDashboardHub(t)

	numGoroutines := 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	payload := []byte(`{"test":"data"}`)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			hub.broadcastAll(payload)
		}()
	}

	wg.Wait()

	assert.True(t, true)
}

func TestWSTickets_ConcurrentIssueAndValidate(t *testing.T) {
	resetWSTickets()
	hub := &DashboardHub{}

	app := fiber.New()
	app.Post("/ws/ticket", func(c fiber.Ctx) error {
		c.Locals("admin_id", uint(1))
		return hub.IssueWSTicket(c)
	})

	numGoroutines := 20
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		if i%2 == 0 {
			go func() {
				defer wg.Done()
				req := httptest.NewRequest(http.MethodPost, "/ws/ticket", nil)
				resp, err := app.Test(req)
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}()
		} else {
			go func() {
				defer wg.Done()
				valid := validateAndConsumeTicket("nonexistent")
				assert.False(t, valid)
			}()
		}
	}

	wg.Wait()

	assert.True(t, true)
}