package repository

import (
	"context"

	"github.com/isolate-project/isolate-panel/internal/models"
	"gorm.io/gorm"
)

// GORMBackupRepository implements BackupRepository using GORM
type GORMBackupRepository struct {
	db *gorm.DB
}

// NewGORMBackupRepository creates a new GORM-based BackupRepository
func NewGORMBackupRepository(db *gorm.DB) BackupRepository {
	return &GORMBackupRepository{db: db}
}

func (r *GORMBackupRepository) Create(ctx context.Context, backup *models.Backup) error {
	if err := r.db.WithContext(ctx).Create(backup).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMBackupRepository) GetByID(ctx context.Context, id uint) (*models.Backup, error) {
	var backup models.Backup
	if err := r.db.WithContext(ctx).First(&backup, id).Error; err != nil {
		return nil, WrapError(err)
	}
	return &backup, nil
}

func (r *GORMBackupRepository) Update(ctx context.Context, backup *models.Backup) error {
	if err := r.db.WithContext(ctx).Save(backup).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMBackupRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&models.Backup{}, id).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMBackupRepository) List(ctx context.Context, offset, limit int) ([]models.Backup, int64, error) {
	var backups []models.Backup
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Backup{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, WrapError(err)
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&backups).Error; err != nil {
		return nil, 0, WrapError(err)
	}

	return backups, total, nil
}

func (r *GORMBackupRepository) ListByStatus(ctx context.Context, status models.BackupStatus) ([]models.Backup, error) {
	var backups []models.Backup
	if err := r.db.WithContext(ctx).
		Where("status = ?", status).
		Order("created_at DESC").
		Find(&backups).Error; err != nil {
		return nil, WrapError(err)
	}
	return backups, nil
}

func (r *GORMBackupRepository) UpdateStatus(ctx context.Context, id uint, status models.BackupStatus, errorMessage string) error {
	updates := map[string]interface{}{
		"status":        status,
		"error_message": errorMessage,
	}
	if err := r.db.WithContext(ctx).Model(&models.Backup{}).
		Where("id = ?", id).
		Updates(updates).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMBackupRepository) UpdateCompletion(ctx context.Context, id uint, fileSizeBytes int64, checksumSHA256 string, durationMs int) error {
	updates := map[string]interface{}{
		"file_size_bytes":  fileSizeBytes,
		"checksum_sha256":  checksumSHA256,
		"duration_ms":      durationMs,
		"status":           models.BackupStatusCompleted,
	}
	if err := r.db.WithContext(ctx).Model(&models.Backup{}).
		Where("id = ?", id).
		Updates(updates).Error; err != nil {
		return WrapError(err)
	}
	return nil
}
