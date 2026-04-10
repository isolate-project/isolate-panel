package api

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/isolate-project/isolate-panel/internal/middleware"
	"github.com/isolate-project/isolate-panel/internal/scheduler"
	"github.com/isolate-project/isolate-panel/internal/services"
)

// BackupHandler handles backup API requests
type BackupHandler struct {
	backupService   *services.BackupService
	backupScheduler *scheduler.BackupScheduler
}

// NewBackupHandler creates a new backup handler
func NewBackupHandler(backupService *services.BackupService, backupScheduler *scheduler.BackupScheduler) *BackupHandler {
	return &BackupHandler{
		backupService:   backupService,
		backupScheduler: backupScheduler,
	}
}

// RegisterRoutes registers backup API routes
func (h *BackupHandler) RegisterRoutes(router fiber.Router) {
	backup := router.Group("/backups")

	// Read-only operations (any authenticated admin)
	backup.Get("/", h.ListBackups)
	// Static routes MUST be before parameterized /:id
	backup.Get("/schedule", h.GetSchedule)
	backup.Get("/:id", h.GetBackup)

	// Destructive operations (super-admin only)
	backup.Post("/create", middleware.RequireSuperAdmin(), h.CreateBackup)
	backup.Post("/schedule", middleware.RequireSuperAdmin(), h.SetSchedule)
	backup.Post("/:id/restore", middleware.RequireSuperAdmin(), h.RestoreBackup)
	backup.Delete("/:id", middleware.RequireSuperAdmin(), h.DeleteBackup)
	backup.Get("/:id/download", middleware.RequireSuperAdmin(), h.DownloadBackup)
}

// ListBackups returns list of all backups
//
// @Summary      List backups
// @Description  Returns all available backup files with metadata
// @Tags         backups
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /backups [get]
// @Security     BearerAuth
func (h *BackupHandler) ListBackups(c fiber.Ctx) error {
	backups, err := h.backupService.ListBackups()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"data": backups,
	})
}

// GetBackup returns a single backup by ID
//
// @Summary      Get backup
// @Description  Returns metadata for a specific backup
// @Tags         backups
// @Produce      json
// @Param        id   path  int  true  "Backup ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /backups/{id} [get]
// @Security     BearerAuth
func (h *BackupHandler) GetBackup(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid backup ID",
		})
	}

	backup, err := h.backupService.GetBackup(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "backup not found",
		})
	}

	return c.JSON(fiber.Map{
		"data": backup,
	})
}

// CreateBackup creates a new backup
//
// @Summary      Create backup
// @Description  Start a new AES-256-GCM encrypted backup of the database and core configurations
// @Tags         backups
// @Accept       json
// @Produce      json
// @Param        body  body  services.BackupRequest  true  "Backup options"
// @Success      201   {object}  map[string]interface{}
// @Failure      500   {object}  map[string]interface{}
// @Router       /backups/create [post]
// @Security     BearerAuth
func (h *BackupHandler) CreateBackup(c fiber.Ctx) error {
	var req services.BackupRequest

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Set defaults
	if req.Type == "" {
		req.Type = "manual"
	}
	if !req.IncludeCores {
		req.IncludeCores = true // Default to true
	}

	backup, err := h.backupService.CreateBackup(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data":    backup,
		"message": "Backup creation started",
	})
}

// RestoreBackup restores from a backup
//
// @Summary      Restore backup
// @Description  Restore database and configurations from a backup file
// @Tags         backups
// @Accept       json
// @Produce      json
// @Param        id    path  int                     true  "Backup ID"
// @Param        body  body  map[string]interface{}  false "{force: false}"
// @Success      200   {object}  map[string]interface{}
// @Failure      500   {object}  map[string]interface{}
// @Router       /backups/{id}/restore [post]
// @Security     BearerAuth
func (h *BackupHandler) RestoreBackup(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid backup ID",
		})
	}

	if err := h.backupService.RestoreBackup(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Restore operation started",
	})
}

// DeleteBackup deletes a backup
//
// @Summary      Delete backup
// @Description  Delete a backup file and its record
// @Tags         backups
// @Produce      json
// @Param        id   path  int  true  "Backup ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /backups/{id} [delete]
// @Security     BearerAuth
func (h *BackupHandler) DeleteBackup(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid backup ID",
		})
	}

	if err := h.backupService.DeleteBackup(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Backup deleted",
	})
}

// DownloadBackup downloads a backup file
//
// @Summary      Download backup
// @Description  Download a backup file as a binary stream
// @Tags         backups
// @Produce      application/octet-stream
// @Param        id   path  int  true  "Backup ID"
// @Success      200  {file}  binary
// @Failure      404  {object}  map[string]interface{}
// @Router       /backups/{id}/download [get]
// @Security     BearerAuth
func (h *BackupHandler) DownloadBackup(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid backup ID",
		})
	}

	filePath, filename, err := h.backupService.DownloadBackup(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	c.Set("Content-Type", "application/octet-stream")
	safeName := strings.Map(func(r rune) rune {
		if r == '"' || r == '\\' || r == '\r' || r == '\n' {
			return '_'
		}
		return r
	}, filename)
	c.Set("Content-Disposition", "attachment; filename=\""+safeName+"\"")

	return c.SendFile(filePath)
}

// GetSchedule returns the current backup schedule
//
// @Summary      Get backup schedule
// @Description  Returns current backup cron schedule and next scheduled run time
// @Tags         backups
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /backups/schedule [get]
// @Security     BearerAuth
func (h *BackupHandler) GetSchedule(c fiber.Ctx) error {
	schedule, err := h.backupScheduler.GetSchedule()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	nextRun, _ := h.backupScheduler.GetNextRun()

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"schedule": schedule,
			"next_run": nextRun,
		},
	})
}

// SetSchedule sets the backup schedule
//
// @Summary      Set backup schedule
// @Description  Set a cron expression for automatic backups (empty string to disable)
// @Tags         backups
// @Accept       json
// @Produce      json
// @Param        body  body  map[string]string  true  "{cron: '0 2 * * *'}"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Router       /backups/schedule [post]
// @Security     BearerAuth
func (h *BackupHandler) SetSchedule(c fiber.Ctx) error {
	var req struct {
		Cron string `json:"cron"`
	}

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := h.backupScheduler.UpdateSchedule(req.Cron); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	message := "Schedule cleared"
	if req.Cron != "" {
		message = "Schedule updated"
	}

	return c.JSON(fiber.Map{
		"message": message,
		"data": fiber.Map{
			"cron": req.Cron,
		},
	})
}
