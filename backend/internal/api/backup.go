package api

import (
	"github.com/gofiber/fiber/v3"

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

	// Backup operations
	backup.Get("/", h.ListBackups)
	backup.Get("/:id", h.GetBackup)
	backup.Post("/create", h.CreateBackup)
	backup.Post("/:id/restore", h.RestoreBackup)
	backup.Delete("/:id", h.DeleteBackup)
	backup.Get("/:id/download", h.DownloadBackup)

	// Schedule operations
	backup.Get("/schedule", h.GetSchedule)
	backup.Post("/schedule", h.SetSchedule)
}

// ListBackups returns list of all backups
func (h *BackupHandler) ListBackups(c fiber.Ctx) error {
	backups, err := h.backupService.ListBackups()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": backups,
	})
}

// GetBackup returns a single backup by ID
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
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data":    backup,
		"message": "Backup creation started",
	})
}

// RestoreBackup restores from a backup
func (h *BackupHandler) RestoreBackup(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid backup ID",
		})
	}

	var req struct {
		Force bool `json:"force"`
	}

	if err := c.Bind().JSON(&req); err != nil {
		req.Force = false
	}

	if err := h.backupService.RestoreBackup(id, req.Force); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Restore operation started",
	})
}

// DeleteBackup deletes a backup
func (h *BackupHandler) DeleteBackup(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid backup ID",
		})
	}

	if err := h.backupService.DeleteBackup(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Backup deleted",
	})
}

// DownloadBackup downloads a backup file
func (h *BackupHandler) DownloadBackup(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid backup ID",
		})
	}

	data, filename, err := h.backupService.DownloadBackup(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	c.Set("Content-Type", "application/octet-stream")
	c.Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Set("Content-Length", string(rune(len(data))))

	return c.Send(data)
}

// GetSchedule returns the current backup schedule
func (h *BackupHandler) GetSchedule(c fiber.Ctx) error {
	schedule, err := h.backupScheduler.GetSchedule()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
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
