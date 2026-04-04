package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync"
	"time"

	fwebsocket "github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/auth"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services"
)

// DashboardPayload is the JSON pushed to each connected WebSocket client.
type DashboardPayload struct {
	ActiveConnections int64 `json:"active_connections"`
	TotalUsers        int64 `json:"total_users"`
	ActiveUsers       int64 `json:"active_users"`
	TotalTrafficBytes int64 `json:"total_traffic_bytes"`
	CoresRunning      int64 `json:"cores_running"`
}

var wsUpgrader = fwebsocket.FastHTTPUpgrader{
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		origin := string(ctx.Request.Header.Peek("Origin"))
		if origin == "" {
			return true // same-origin requests omit Origin
		}
		// Panel is accessed via SSH tunnel on localhost
		for _, allowed := range []string{
			"http://localhost", "https://localhost",
			"http://127.0.0.1", "https://127.0.0.1",
		} {
			if origin == allowed || strings.HasPrefix(origin, allowed+":") {
				return true
			}
		}
		return false
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// DashboardHub manages WebSocket clients and broadcasts dashboard stats.
type DashboardHub struct {
	db                *gorm.DB
	connectionTracker *services.ConnectionTracker
	tokenService      *auth.TokenService

	mu      sync.RWMutex
	clients map[*fwebsocket.Conn]struct{}

	register   chan *fwebsocket.Conn
	unregister chan *fwebsocket.Conn
	done       chan struct{}
}

// NewDashboardHub creates a new hub. Call Run() in a goroutine to start it.
func NewDashboardHub(
	db *gorm.DB,
	connectionTracker *services.ConnectionTracker,
	tokenService *auth.TokenService,
) *DashboardHub {
	return &DashboardHub{
		db:                db,
		connectionTracker: connectionTracker,
		tokenService:      tokenService,
		clients:           make(map[*fwebsocket.Conn]struct{}),
		register:          make(chan *fwebsocket.Conn, 32),
		unregister:        make(chan *fwebsocket.Conn, 32),
		done:              make(chan struct{}),
	}
}

// Run processes register/unregister events and broadcasts stats every 5 seconds.
// Must be called in its own goroutine.
func (h *DashboardHub) Run() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = struct{}{}
			h.mu.Unlock()
			// Send initial payload immediately on connect
			if payload, err := h.collectStats(); err == nil {
				_ = conn.WriteMessage(fwebsocket.TextMessage, payload)
			}

		case conn := <-h.unregister:
			h.mu.Lock()
			delete(h.clients, conn)
			h.mu.Unlock()

		case <-ticker.C:
			payload, err := h.collectStats()
			if err != nil {
				continue
			}
			h.broadcastAll(payload)

		case <-h.done:
			return
		}
	}
}

// Stop shuts down the hub's goroutine.
func (h *DashboardHub) Stop() {
	close(h.done)
}

func (h *DashboardHub) collectStats() ([]byte, error) {
	var connCount int64
	if h.connectionTracker != nil {
		if n, err := h.connectionTracker.GetActiveConnectionsCount(); err == nil {
			connCount = n
		}
	}

	var totalUsers, activeUsers, coresRunning, totalTraffic int64
	h.db.Model(&models.User{}).Count(&totalUsers)
	h.db.Model(&models.User{}).Where("is_active = ?", true).Count(&activeUsers)
	h.db.Model(&models.Core{}).Where("is_running = ?", true).Count(&coresRunning)
	h.db.Model(&models.User{}).Select("COALESCE(SUM(traffic_used_bytes), 0)").Scan(&totalTraffic)

	return json.Marshal(DashboardPayload{
		ActiveConnections: connCount,
		TotalUsers:        totalUsers,
		ActiveUsers:       activeUsers,
		TotalTrafficBytes: totalTraffic,
		CoresRunning:      coresRunning,
	})
}

func (h *DashboardHub) broadcastAll(payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for conn := range h.clients {
		_ = conn.WriteMessage(fwebsocket.TextMessage, payload)
	}
}

// wsTicket stores a one-time WebSocket authentication ticket.
type wsTicket struct {
	expiresAt time.Time
}

var (
	wsTickets   = make(map[string]wsTicket)
	wsTicketsMu sync.Mutex
)

// IssueWSTicket creates a short-lived one-time ticket for WebSocket auth.
// The ticket replaces passing the JWT access token in the URL query string,
// preventing token leakage into logs and browser history.
// POST /api/ws/ticket (requires JWT auth via middleware)
func (h *DashboardHub) IssueWSTicket(c fiber.Ctx) error {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate ticket"})
	}
	ticket := hex.EncodeToString(b)

	wsTicketsMu.Lock()
	wsTickets[ticket] = wsTicket{expiresAt: time.Now().Add(30 * time.Second)}
	wsTicketsMu.Unlock()

	return c.JSON(fiber.Map{"ticket": ticket})
}

func validateAndConsumeTicket(ticket string) bool {
	wsTicketsMu.Lock()
	defer wsTicketsMu.Unlock()
	t, ok := wsTickets[ticket]
	if !ok {
		return false
	}
	delete(wsTickets, ticket) // one-time use
	return time.Now().Before(t.expiresAt)
}

// DashboardWS is the Fiber handler for GET /api/ws/dashboard
// Auth is done via ?ticket=<one-time-ticket> issued by IssueWSTicket.
// Falls back to ?token=<access_token> for backward compatibility.
func (h *DashboardHub) DashboardWS(c fiber.Ctx) error {
	// Prefer one-time ticket
	ticket := c.Query("ticket")
	if ticket != "" {
		if !validateAndConsumeTicket(ticket) {
			return c.Status(fiber.StatusUnauthorized).SendString("invalid or expired ticket")
		}
	} else {
		// Backward compatibility: accept JWT token directly
		token := c.Query("token")
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).SendString("ticket or token required")
		}
		if _, err := h.tokenService.ValidateAccessToken(token); err != nil {
			return c.Status(fiber.StatusUnauthorized).SendString("invalid or expired token")
		}
	}

	// Upgrade the fasthttp connection to WebSocket (use RequestCtx, not Context)
	upgradeErr := wsUpgrader.Upgrade(c.RequestCtx(), func(conn *fwebsocket.Conn) {
		h.register <- conn
		defer func() {
			h.unregister <- conn
			conn.Close()
		}()
		// Read and discard client messages; exit on error (disconnect)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})
	if upgradeErr != nil {
		return upgradeErr
	}
	return nil
}
