package services

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/vovk4morkovk4/isolate-panel/internal/core"
	"github.com/vovk4morkovk4/isolate-panel/internal/models"
)

// CoreLifecycleManager manages automatic starting/stopping of cores based on inbounds
type CoreLifecycleManager struct {
	db          *gorm.DB
	coreManager *core.CoreManager
}

// NewCoreLifecycleManager creates a new lifecycle manager
func NewCoreLifecycleManager(db *gorm.DB, coreManager *core.CoreManager) *CoreLifecycleManager {
	return &CoreLifecycleManager{
		db:          db,
		coreManager: coreManager,
	}
}

// InitializeCores starts only necessary cores at system startup (lazy loading)
func (clm *CoreLifecycleManager) InitializeCores() error {
	cores := []string{"singbox", "xray", "mihomo"}

	for _, coreName := range cores {
		shouldStart, err := clm.shouldCoreBeRunning(coreName)
		if err != nil {
			return fmt.Errorf("failed to check if core should run: %w", err)
		}

		if shouldStart {
			fmt.Printf("Starting core %s (has active inbounds)\n", coreName)
			if err := clm.coreManager.StartCore(coreName); err != nil {
				fmt.Printf("Failed to start core %s: %v\n", coreName, err)
				// Don't return error, continue with other cores
			}
		} else {
			fmt.Printf("Skipping core %s (no active inbounds)\n", coreName)
		}
	}

	return nil
}

// shouldCoreBeRunning checks if a core has any active inbounds
func (clm *CoreLifecycleManager) shouldCoreBeRunning(coreName string) (bool, error) {
	var coreModel models.Core
	if err := clm.db.Where("name = ?", coreName).First(&coreModel).Error; err != nil {
		return false, err
	}

	// Check if there are any enabled inbounds for this core
	var count int64
	err := clm.db.Model(&models.Inbound{}).
		Where("core_id = ? AND is_enabled = ?", coreModel.ID, true).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// OnInboundCreated is called when a new inbound is created
func (clm *CoreLifecycleManager) OnInboundCreated(inbound *models.Inbound) error {
	// Load core
	var coreModel models.Core
	if err := clm.db.First(&coreModel, inbound.CoreID).Error; err != nil {
		return err
	}

	// Check if core is running
	isRunning, err := clm.coreManager.IsCoreRunning(coreModel.Name)
	if err != nil {
		return err
	}

	if !isRunning {
		fmt.Printf("Starting core %s (first inbound created: %d)\n", coreModel.Name, inbound.ID)

		if err := clm.coreManager.StartCore(coreModel.Name); err != nil {
			return fmt.Errorf("failed to start core: %w", err)
		}
	}

	return nil
}

// OnInboundDeleted is called when an inbound is deleted
func (clm *CoreLifecycleManager) OnInboundDeleted(inbound *models.Inbound) error {
	// Load core
	var coreModel models.Core
	if err := clm.db.First(&coreModel, inbound.CoreID).Error; err != nil {
		return err
	}

	// Check if there are any other enabled inbounds for this core
	var count int64
	err := clm.db.Model(&models.Inbound{}).
		Where("core_id = ? AND is_enabled = ? AND id != ?", coreModel.ID, true, inbound.ID).
		Count(&count).Error

	if err != nil {
		return err
	}

	// If this was the last inbound, stop the core
	if count == 0 {
		fmt.Printf("Stopping core %s (last inbound deleted: %d)\n", coreModel.Name, inbound.ID)

		if err := clm.coreManager.StopCore(coreModel.Name); err != nil {
			fmt.Printf("Failed to stop core %s: %v\n", coreModel.Name, err)
			// Don't return error, this is not critical
		}
	}

	return nil
}

// OnInboundUpdated is called when an inbound is updated
func (clm *CoreLifecycleManager) OnInboundUpdated(inbound *models.Inbound, wasEnabled bool) error {
	// If inbound was disabled and now enabled, check if we need to start core
	if !wasEnabled && inbound.IsEnabled {
		return clm.OnInboundCreated(inbound)
	}

	// If inbound was enabled and now disabled, check if we need to stop core
	if wasEnabled && !inbound.IsEnabled {
		return clm.OnInboundDeleted(inbound)
	}

	return nil
}
