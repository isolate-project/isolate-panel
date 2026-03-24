package api

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/vovk4morkovk4/isolate-panel/internal/models"
	"github.com/vovk4morkovk4/isolate-panel/internal/services"
	"gorm.io/gorm"
)

type StatsHandler struct {
	db                *gorm.DB
	trafficCollector  *services.TrafficCollector
	connectionTracker *services.ConnectionTracker
}

func NewStatsHandler(
	db *gorm.DB,
	trafficCollector *services.TrafficCollector,
	connectionTracker *services.ConnectionTracker,
) *StatsHandler {
	return &StatsHandler{
		db:                db,
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

	granularity := c.Query("granularity", "daily")
	days, _ := strconv.Atoi(c.Query("days", "30"))

	// Validate granularity
	if granularity != "raw" && granularity != "hourly" && granularity != "daily" {
		granularity = "daily"
	}

	// Calculate date range
	now := time.Now()
	startDate := now.AddDate(0, 0, -days)

	// Query traffic stats from database
	type TrafficStat struct {
		Date     string `json:"date"`
		Upload   uint64 `json:"upload"`
		Download uint64 `json:"download"`
		Total    uint64 `json:"total"`
	}

	var stats []TrafficStat

	// Build query based on granularity
	query := h.db.Table("traffic_stats").
		Select("DATE(recorded_at) as date, SUM(upload) as upload, SUM(download) as download, SUM(total) as total").
		Where("user_id = ?", userID).
		Where("granularity = ?", granularity).
		Where("recorded_at >= ?", startDate).
		Group("DATE(recorded_at)").
		Order("date ASC")

	if err := query.Scan(&stats).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Calculate totals
	var totalUpload, totalDownload uint64
	for _, s := range stats {
		totalUpload += s.Upload
		totalDownload += s.Download
	}

	return c.JSON(fiber.Map{
		"user_id":        userID,
		"granularity":    granularity,
		"days":           days,
		"start_date":     startDate.Format("2006-01-02"),
		"end_date":       now.Format("2006-01-02"),
		"stats":          stats,
		"total_upload":   totalUpload,
		"total_download": totalDownload,
		"total":          totalUpload + totalDownload,
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
	var connections []models.ActiveConnection
	if err := h.db.Find(&connections).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"connections": connections,
		"total":       len(connections),
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

	// Get total users count
	var totalUsers int64
	h.db.Model(&models.User{}).Where("is_active = ?", true).Count(&totalUsers)

	// Get total traffic (sum from users)
	var totalTraffic int64
	h.db.Model(&models.User{}).Select("SUM(traffic_used_bytes)").Scan(&totalTraffic)

	// Get cores running count
	var coresRunning int64
	h.db.Model(&models.Core{}).Where("is_running = ?", true).Count(&coresRunning)

	return c.JSON(fiber.Map{
		"active_connections": connCount,
		"total_users":        totalUsers,
		"total_traffic":      totalTraffic,
		"cores_running":      coresRunning,
	})
}
