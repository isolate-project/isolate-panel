package repository

import (
	"context"

	"github.com/isolate-project/isolate-panel/internal/models"
	"gorm.io/gorm"
)

// GORMCoreRepository implements CoreRepository using GORM
type GORMCoreRepository struct {
	db *gorm.DB
}

// NewGORMCoreRepository creates a new GORM-based CoreRepository
func NewGORMCoreRepository(db *gorm.DB) CoreRepository {
	return &GORMCoreRepository{db: db}
}

func (r *GORMCoreRepository) Create(ctx context.Context, core *models.Core) error {
	if err := r.db.WithContext(ctx).Create(core).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMCoreRepository) GetByID(ctx context.Context, id uint) (*models.Core, error) {
	var core models.Core
	if err := r.db.WithContext(ctx).First(&core, id).Error; err != nil {
		return nil, WrapError(err)
	}
	return &core, nil
}

func (r *GORMCoreRepository) GetByName(ctx context.Context, name string) (*models.Core, error) {
	var core models.Core
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&core).Error; err != nil {
		return nil, WrapError(err)
	}
	return &core, nil
}

func (r *GORMCoreRepository) Update(ctx context.Context, core *models.Core) error {
	if err := r.db.WithContext(ctx).Save(core).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMCoreRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&models.Core{}, id).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMCoreRepository) List(ctx context.Context) ([]models.Core, error) {
	var cores []models.Core
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&cores).Error; err != nil {
		return nil, WrapError(err)
	}
	return cores, nil
}

func (r *GORMCoreRepository) UpdateStatus(ctx context.Context, id uint, isRunning bool, pid *int) error {
	updates := map[string]interface{}{
		"is_running": isRunning,
	}
	if pid != nil {
		updates["p_id"] = *pid
	} else {
		updates["p_id"] = nil
	}
	if err := r.db.WithContext(ctx).Model(&models.Core{}).
		Where("id = ?", id).
		Updates(updates).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMCoreRepository) UpdateHealth(ctx context.Context, id uint, healthStatus, lastError string) error {
	updates := map[string]interface{}{
		"health_status": healthStatus,
		"last_error":    lastError,
	}
	if err := r.db.WithContext(ctx).Model(&models.Core{}).
		Where("id = ?", id).
		Updates(updates).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMCoreRepository) UpdateUptime(ctx context.Context, id uint, uptimeSeconds int, restartCount int) error {
	updates := map[string]interface{}{
		"uptime_seconds": uptimeSeconds,
		"restart_count":  restartCount,
	}
	if err := r.db.WithContext(ctx).Model(&models.Core{}).
		Where("id = ?", id).
		Updates(updates).Error; err != nil {
		return WrapError(err)
	}
	return nil
}
