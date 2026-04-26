package api

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/isolate-project/isolate-panel/internal/cores"
	"github.com/isolate-project/isolate-panel/internal/services"
)

// SystemHandler handles system-level endpoints (resources, connections, emergency cleanup).
type SystemHandler struct {
	connectionTracker *services.ConnectionTracker
	coreManager       *cores.CoreManager
}

// NewSystemHandler creates a new system handler.
func NewSystemHandler(ct *services.ConnectionTracker, cm *cores.CoreManager) *SystemHandler {
	return &SystemHandler{
		connectionTracker: ct,
		coreManager:       cm,
	}
}

// GetResources returns current system resource usage (RAM, CPU).
//
// @Summary      System resources
// @Description  Returns RAM and CPU utilisation percentages
// @Tags         system
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /system/resources [get]
// @Security     BearerAuth
func (h *SystemHandler) GetResources(c fiber.Ctx) error {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	// Try to read total system memory from /proc/meminfo (Linux)
	totalMemory := uint64(0)
	availableMemory := uint64(0)
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				totalMemory = parseMemInfoKB(line) * 1024
			} else if strings.HasPrefix(line, "MemAvailable:") {
				availableMemory = parseMemInfoKB(line) * 1024
			}
		}
	}

	ramPercent := float64(0)
	if totalMemory > 0 {
		usedMemory := totalMemory - availableMemory
		ramPercent = float64(usedMemory) / float64(totalMemory) * 100
	}

	cpuPercent := float64(runtime.NumGoroutine()) / float64(runtime.NumCPU()*100) * 100
	if cpuPercent > 100 {
		cpuPercent = 100
	}

	return c.JSON(fiber.Map{
		"ram_percent": ramPercent,
		"cpu_percent": cpuPercent,
		"ram_total":   totalMemory,
		"ram_used":    totalMemory - availableMemory,
		"goroutines":  runtime.NumGoroutine(),
		"num_cpu":     runtime.NumCPU(),
	})
}

// GetConnections returns the active connection count.
//
// @Summary      Active connection count
// @Description  Returns the number of currently tracked active connections
// @Tags         system
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /system/connections [get]
// @Security     BearerAuth
func (h *SystemHandler) GetConnections(c fiber.Ctx) error {
	count, err := h.connectionTracker.GetActiveConnectionsCount()
	if err != nil {
		count = 0
	}
	return c.JSON(fiber.Map{
		"count": count,
	})
}

// EmergencyCleanup frees memory and runs GC.
//
// @Summary      Emergency cleanup
// @Description  Triggers garbage collection and frees OS memory (RAM panic button)
// @Tags         system
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /system/emergency-cleanup [post]
// @Security     BearerAuth
func (h *SystemHandler) EmergencyCleanup(c fiber.Ctx) error {
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	runtime.GC()
	debug.FreeOSMemory()

	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	freed := int64(before.Alloc) - int64(after.Alloc)
	if freed < 0 {
		freed = 0
	}

	return c.JSON(fiber.Map{
		"message":      "Emergency cleanup completed",
		"freed_bytes":  freed,
		"alloc_before": before.Alloc,
		"alloc_after":  after.Alloc,
	})
}

// parseMemInfoKB parses a /proc/meminfo line and returns the value in KB.
func parseMemInfoKB(line string) uint64 {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return 0
	}
	val, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return 0
	}
	return val
}

// GetCoreLogs returns recent logs for a core process.
//
// @Summary      Core logs
// @Description  Returns recent stdout/stderr log lines for a core process
// @Tags         cores
// @Produce      json
// @Param        name   path   string  true   "Core name"
// @Param        lines  query  int     false  "Number of lines"  default(100)
// @Success      200    {object}  map[string]interface{}
// @Failure      404    {object}  map[string]interface{}
// @Router       /cores/{name}/logs [get]
// @Security     BearerAuth
func GetCoreLogs(coreManager *cores.CoreManager) fiber.Handler {
	return func(c fiber.Ctx) error {
		name := c.Params("name")
		if name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Core name is required",
			})
		}

		safeName := filepath.Base(name)
		if safeName != name {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid core name",
			})
		}

		// Verify the core exists
		_, err := coreManager.GetCoreStatus(c.Context(), safeName)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Core not found",
			})
		}

		// Try to read log files from common supervisord log locations
		logPaths := []string{
			fmt.Sprintf("/var/log/supervisor/%s-stdout.log", safeName),
			fmt.Sprintf("/var/log/supervisor/%s.log", safeName),
			fmt.Sprintf("/var/log/%s.log", safeName),
		}

		maxBytes := 64 * 1024 // Read last 64KB
		for _, logPath := range logPaths {
			data, err := readTail(logPath, maxBytes)
			if err == nil {
				lines := strings.Split(strings.TrimSpace(string(data)), "\n")
				// Limit number of lines
				linesParam, _ := strconv.Atoi(c.Query("lines", "100"))
				if linesParam < 1 {
					linesParam = 100
				}
				if len(lines) > linesParam {
					lines = lines[len(lines)-linesParam:]
				}
				return c.JSON(fiber.Map{
					"core":  safeName,
					"lines": lines,
					"total": len(lines),
				})
			}
		}

		return c.JSON(fiber.Map{
			"core":  safeName,
			"lines": []string{},
			"total": 0,
		})
	}
}

// readTail reads the last maxBytes from a file.
func readTail(path string, maxBytes int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	size := stat.Size()
	offset := int64(0)
	if size > int64(maxBytes) {
		offset = size - int64(maxBytes)
	}

	buf := make([]byte, size-offset)
	_, err = f.ReadAt(buf, offset)
	if err != nil {
		return nil, err
	}

	return buf, nil
}
