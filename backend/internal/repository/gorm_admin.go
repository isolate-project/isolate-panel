package repository

import (
	"context"
	"time"

	"github.com/isolate-project/isolate-panel/internal/models"
	"gorm.io/gorm"
)

// GORMAdminRepository implements AdminRepository using GORM
type GORMAdminRepository struct {
	db *gorm.DB
}

// NewGORMAdminRepository creates a new GORM-based AdminRepository
func NewGORMAdminRepository(db *gorm.DB) AdminRepository {
	return &GORMAdminRepository{db: db}
}

func (r *GORMAdminRepository) Create(ctx context.Context, admin *models.Admin) error {
	if err := r.db.WithContext(ctx).Create(admin).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMAdminRepository) GetByID(ctx context.Context, id uint) (*models.Admin, error) {
	var admin models.Admin
	if err := r.db.WithContext(ctx).First(&admin, id).Error; err != nil {
		return nil, WrapError(err)
	}
	return &admin, nil
}

func (r *GORMAdminRepository) GetByUsername(ctx context.Context, username string) (*models.Admin, error) {
	var admin models.Admin
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&admin).Error; err != nil {
		return nil, WrapError(err)
	}
	return &admin, nil
}

func (r *GORMAdminRepository) Update(ctx context.Context, admin *models.Admin) error {
	if err := r.db.WithContext(ctx).Save(admin).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMAdminRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&models.Admin{}, id).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMAdminRepository) List(ctx context.Context) ([]models.Admin, error) {
	var admins []models.Admin
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&admins).Error; err != nil {
		return nil, WrapError(err)
	}
	return admins, nil
}

func (r *GORMAdminRepository) UpdateLastLogin(ctx context.Context, id uint) error {
	now := time.Now()
	if err := r.db.WithContext(ctx).Model(&models.Admin{}).
		Where("id = ?", id).
		Update("last_login_at", now).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMAdminRepository) UpdatePassword(ctx context.Context, id uint, passwordHash string) error {
	if err := r.db.WithContext(ctx).Model(&models.Admin{}).
		Where("id = ?", id).
		Update("password_hash", passwordHash).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMAdminRepository) UpdateTOTPSecret(ctx context.Context, id uint, secret string) error {
	if err := r.db.WithContext(ctx).Model(&models.Admin{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"totp_secret":  secret,
			"totp_enabled": secret != "",
		}).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMAdminRepository) UpdatePermissions(ctx context.Context, id uint, permissions uint64) error {
	if err := r.db.WithContext(ctx).Model(&models.Admin{}).
		Where("id = ?", id).
		Update("permissions", permissions).Error; err != nil {
		return WrapError(err)
	}
	return nil
}
