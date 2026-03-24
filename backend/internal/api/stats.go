package api

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/vovk4morkovk4/isolate-panel/internal/services"
)

type StatsHandler struct {
	trafficCollector  *services.TrafficCollector
	connectionTracker *services.ConnectionTracker
}

func NewStatsHandler(
	trafficCollector *services.TrafficCollector,
	connectionTracker *services.ConnectionTracker,
) *StatsHandler {
	return &StatsHandler{
		trafficCollector:  trafficCollector,
		connectionTracker: connectionTracker,
	}
}

// GetUserTrafficStats returns traffic statistics for a user
func (h *StatsHandler) GetUserTrafficStats(c fiber.Ctx) error {
	userID, err := strconv.ParseUint(c.Params("user_id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	granularity := c.Query("granularity", "raw")
	days, _ := strconv.Atoi(c.Query("days", "7"))

	// Query traffic stats from database
	// This is a placeholder - actual implementation would query the database
	startTime := time.Now().AddDate(0, 0, -days)

	type TrafficStat struct {
		Date     string `json:"date"`
		Upload   uint64 `json:"upload"`
		Download uint64 `json:"download"`
		Total    uint64 `json:"total"`
	}

	// Placeholder response
	return c.JSON(fiber.Map{
		"user_id":        userID,
		"granularity":    granularity,
		"days":           days,
		"start_date":     startTime.Format("2006-01-02"),
		"end_date":       time.Now().Format("2006-01-02"),
		"stats":          []TrafficStat{},
		"total_upload":   0,
		"total_download": 0,
	})
}

// GetActiveConnections returns active connections
func (h *StatsHandler) GetActiveConnections(c fiber.Ctx) error {
	userIDStr := c.Query("user_id")

	if userIDStr != "" {
		userID, err := strconv.ParseUint(userIDStr, 10, 32)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid user ID",
			})
		}

		connections, err := h.connectionTracker.GetUserConnections(uint(userID))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"connections": connections,
			"total":       len(connections),
		})
	}

	// Get all connections
	count, err := h.connectionTracker.GetActiveConnectionsCount()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Placeholder - would return full list with pagination
	return c.JSON(fiber.Map{
		"connections": []interface{}{},
		"total":       count,
		"message":     "Use ?user_id=X to filter by user",
	})
}

// DisconnectUser disconnects all active connections for a user
func (h *StatsHandler) DisconnectUser(c fiber.Ctx) error {
	userID, err := strconv.ParseUint(c.Params("user_id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Get user connections
	connections, err := h.connectionTracker.GetUserConnections(uint(userID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Remove all connections
	for _, conn := range connections {
		h.connectionTracker.RemoveConnection(conn.ID)
	}

	return c.JSON(fiber.Map{
		"message":            "User disconnected",
		"connections_closed": len(connections),
	})
}

// GetDashboardStats returns overall dashboard statistics
func (h *StatsHandler) GetDashboardStats(c fiber.Ctx) error {
	// Get active connections count
	connCount, err := h.connectionTracker.GetActiveConnectionsCount()
	if err != nil {
		connCount = 0
	}

	// Placeholder for other stats
	return c.JSON(fiber.Map{
		"active_connections": connCount,
		"total_users":        0, // Would query from database
		"total_traffic":      0, // Would aggregate from traffic_stats
		"cores_running":      0, // Would query from cores table
	})
}
