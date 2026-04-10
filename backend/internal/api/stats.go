package api

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services"
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
//
// @Summary      User traffic stats
// @Description  Returns per-day traffic statistics for a specific user
// @Tags         stats
// @Produce      json
// @Param        user_id      path   int     true   "User ID"
// @Param        days         query  int     false  "Number of days"         default(30)
// @Param        granularity  query  string  false  "raw, hourly, or daily"  default(daily)
// @Success      200          {object}  map[string]interface{}
// @Router       /stats/user/{user_id}/traffic [get]
// @Security     BearerAuth
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
			"error": "Internal server error",
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
//
// @Summary      Active connections
// @Description  Returns current active connections; filter by user_id to get user-specific connections
// @Tags         stats
// @Produce      json
// @Param        user_id  query  int  false  "Filter by user ID"
// @Success      200      {object}  map[string]interface{}
// @Router       /stats/connections [get]
// @Security     BearerAuth
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
				"error": "Internal server error",
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
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"connections": connections,
		"total":       len(connections),
	})
}

// DisconnectUser disconnects all active connections for a user.
// First attempts to close connections via core APIs, then removes from DB.
//
// @Summary      Disconnect user
// @Description  Close all active connections for a user (soft disconnect)
// @Tags         stats
// @Produce      json
// @Param        user_id  path  int  true  "User ID"
// @Success      200      {object}  map[string]interface{}
// @Router       /stats/user/{user_id}/disconnect [post]
// @Security     BearerAuth
func (h *StatsHandler) DisconnectUser(c fiber.Ctx) error {
	userID, err := strconv.ParseUint(c.Params("user_id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Get user connections from DB
	connections, err := h.connectionTracker.GetUserConnections(uint(userID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	closed := 0
	var errors []string

	// Attempt to close each connection via core API
	for _, conn := range connections {
		// Try to close via core (Xray may not support this, Sing-box/Mihomo do)
		closeErr := h.connectionTracker.CloseUserConnection(
			c.Context(),
			conn.CoreName,
			conn.CoreID,
			strconv.FormatUint(uint64(conn.ID), 10),
		)
		if closeErr != nil {
			// Core-level close failed — just remove from DB tracking
			errors = append(errors, closeErr.Error())
		}

		// Always remove from DB
		h.connectionTracker.RemoveConnection(conn.ID)
		closed++
	}

	return c.JSON(fiber.Map{
		"message":            "User disconnected",
		"connections_closed": closed,
		"errors":             errors,
	})
}

// KickUser fully removes a user from all running cores (force disconnect).
// The user is removed from each core inbound they are assigned to.
// Config regeneration can optionally re-add them.
//
// @Summary      Kick user from cores
// @Description  Force-remove a user from all proxy core inbounds (hard disconnect)
// @Tags         stats
// @Produce      json
// @Param        user_id  path  int  true  "User ID"
// @Success      200      {object}  map[string]interface{}
// @Router       /stats/user/{user_id}/kick [post]
// @Security     BearerAuth
func (h *StatsHandler) KickUser(c fiber.Ctx) error {
	userID, err := strconv.ParseUint(c.Params("user_id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Get user with UUID
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Get all inbounds assigned to this user
	var inbounds []models.Inbound
	if err := h.db.Table("inbounds").
		Joins("JOIN user_inbound_mapping ON user_inbound_mapping.inbound_id = inbounds.id").
		Where("user_inbound_mapping.user_id = ?", userID).
		Find(&inbounds).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get user inbounds",
		})
	}

	kicked := 0
	var errors []string

	for _, inbound := range inbounds {
		// Get core name for this inbound
		var core models.Core
		if err := h.db.First(&core, inbound.CoreID).Error; err != nil {
			continue
		}

		// Remove user from this inbound in the core
		// Tag format matches config generators: "protocol_id"
		inboundTag := fmt.Sprintf("%s_%d", inbound.Protocol, inbound.ID)
		removeErr := h.connectionTracker.RemoveUserFromCore(
			c.Context(),
			core.Name,
			inboundTag,
			user.UUID,
		)
		if removeErr != nil {
			errors = append(errors, removeErr.Error())
		} else {
			kicked++
		}
	}

	// Also clean up all DB connections for this user
	connections, _ := h.connectionTracker.GetUserConnections(uint(userID))
	for _, conn := range connections {
		h.connectionTracker.RemoveConnection(conn.ID)
	}

	return c.JSON(fiber.Map{
		"message":          "User kicked from cores",
		"inbounds_kicked":  kicked,
		"connections_removed": len(connections),
		"errors":           errors,
	})
}

// GetDashboardStats returns overall dashboard statistics
//
// @Summary      Dashboard stats
// @Description  Returns summary stats: active connections, total users, traffic, running cores
// @Tags         stats
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /stats/dashboard [get]
// @Security     BearerAuth
func (h *StatsHandler) GetDashboardStats(c fiber.Ctx) error {
	// Get active connections count
	connCount, err := h.connectionTracker.GetActiveConnectionsCount()
	if err != nil {
		connCount = 0
	}

	// Get total users count
	var totalUsers int64
	if err := h.db.Model(&models.User{}).Where("is_active = ?", true).Count(&totalUsers).Error; err != nil {
		totalUsers = 0
	}

	// Get total traffic (sum from users)
	var totalTraffic int64
	if err := h.db.Model(&models.User{}).Select("SUM(traffic_used_bytes)").Scan(&totalTraffic).Error; err != nil {
		totalTraffic = 0
	}

	// Get cores running count
	var coresRunning int64
	if err := h.db.Model(&models.Core{}).Where("is_running = ?", true).Count(&coresRunning).Error; err != nil {
		coresRunning = 0
	}

	return c.JSON(fiber.Map{
		"active_connections": connCount,
		"total_users":        totalUsers,
		"total_traffic":      totalTraffic,
		"cores_running":      coresRunning,
	})
}

// GetTrafficOverview returns aggregated traffic overview for all users.
// Used by Dashboard charts.
//
// @Summary      Traffic overview
// @Description  Returns aggregated traffic data for all users over a time period (used by dashboard charts)
// @Tags         stats
// @Produce      json
// @Param        days         query  int     false  "Number of days"  default(7)
// @Param        granularity  query  string  false  "daily or hourly" default(daily)
// @Success      200          {object}  map[string]interface{}
// @Router       /stats/traffic/overview [get]
// @Security     BearerAuth
func (h *StatsHandler) GetTrafficOverview(c fiber.Ctx) error {
	days, _ := strconv.Atoi(c.Query("days", "7"))
	if days <= 0 || days > 365 {
		days = 7
	}

	granularity := c.Query("granularity", "daily")
	if granularity != "raw" && granularity != "hourly" && granularity != "daily" {
		granularity = "daily"
	}

	now := time.Now()
	startDate := now.AddDate(0, 0, -days)

	type TrafficPoint struct {
		Date     string `json:"date"`
		Upload   uint64 `json:"upload"`
		Download uint64 `json:"download"`
		Total    uint64 `json:"total"`
	}

	var points []TrafficPoint

	err := h.db.Table("traffic_stats").
		Select("DATE(recorded_at) as date, SUM(upload) as upload, SUM(download) as download, SUM(total) as total").
		Where("granularity = ?", granularity).
		Where("recorded_at >= ?", startDate).
		Group("DATE(recorded_at)").
		Order("date ASC").
		Scan(&points).Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Calculate totals
	var totalUpload, totalDownload uint64
	for _, p := range points {
		totalUpload += p.Upload
		totalDownload += p.Download
	}

	return c.JSON(fiber.Map{
		"days":           days,
		"granularity":    granularity,
		"start_date":     startDate.Format("2006-01-02"),
		"end_date":       now.Format("2006-01-02"),
		"points":         points,
		"total_upload":   totalUpload,
		"total_download": totalDownload,
		"total":          totalUpload + totalDownload,
	})
}

// GetTopUsers returns top traffic-consuming users.
// Used by Dashboard charts.
//
// @Summary      Top users by traffic
// @Description  Returns users with the highest traffic consumption (used by dashboard top-users chart)
// @Tags         stats
// @Produce      json
// @Param        limit  query  int  false  "Maximum number of users to return"  default(10)
// @Success      200    {object}  map[string]interface{}
// @Router       /stats/traffic/top-users [get]
// @Security     BearerAuth
func (h *StatsHandler) GetTopUsers(c fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	type TopUser struct {
		UserID           uint   `json:"user_id"`
		Username         string `json:"username"`
		TrafficUsedBytes int64  `json:"traffic_used_bytes"`
		TrafficLimitBytes *int64 `json:"traffic_limit_bytes"`
		IsActive         bool   `json:"is_active"`
	}

	var topUsers []TopUser

	err := h.db.Table("users").
		Select("id as user_id, username, traffic_used_bytes, traffic_limit_bytes, is_active").
		Where("traffic_used_bytes > 0").
		Order("traffic_used_bytes DESC").
		Limit(limit).
		Scan(&topUsers).Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"users": topUsers,
		"total": len(topUsers),
	})
}

