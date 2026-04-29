package repository

import (
	"context"
	"time"

	"github.com/isolate-project/isolate-panel/internal/models"
	"gorm.io/gorm"
)

// GORMUserRepository implements UserRepository using GORM
type GORMUserRepository struct {
	db *gorm.DB
}

// NewGORMUserRepository creates a new GORM-based UserRepository
func NewGORMUserRepository(db *gorm.DB) UserRepository {
	return &GORMUserRepository{db: db}
}

func (r *GORMUserRepository) Create(ctx context.Context, user *models.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMUserRepository) GetByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, WrapError(err)
	}
	return &user, nil
}

func (r *GORMUserRepository) GetByUUID(ctx context.Context, uuid string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("uuid = ?", uuid).First(&user).Error; err != nil {
		return nil, WrapError(err)
	}
	return &user, nil
}

func (r *GORMUserRepository) GetByToken(ctx context.Context, token string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("subscription_token = ?", token).First(&user).Error; err != nil {
		return nil, WrapError(err)
	}
	return &user, nil
}

func (r *GORMUserRepository) Update(ctx context.Context, user *models.User) error {
	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMUserRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&models.User{}, id).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMUserRepository) List(ctx context.Context, offset, limit int, search, status string) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	query := r.db.WithContext(ctx).Model(&models.User{})

	if search != "" {
		query = query.Where("username LIKE ? OR email LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if status == "active" {
		query = query.Where("is_active = ?", true)
	} else if status == "inactive" {
		query = query.Where("is_active = ?", false)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, WrapError(err)
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, WrapError(err)
	}

	return users, total, nil
}

func (r *GORMUserRepository) Search(ctx context.Context, query string, offset, limit int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	dbQuery := r.db.WithContext(ctx).Model(&models.User{}).
		Where("username LIKE ? OR email LIKE ?", "%"+query+"%", "%"+query+"%")

	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, 0, WrapError(err)
	}

	if err := dbQuery.Order("created_at DESC").Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, WrapError(err)
	}

	return users, total, nil
}

func (r *GORMUserRepository) UpdateTrafficUsed(ctx context.Context, id uint, bytes int64) error {
	if err := r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", id).
		Update("traffic_used_bytes", gorm.Expr("traffic_used_bytes + ?", bytes)).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMUserRepository) UpdateOnlineStatus(ctx context.Context, id uint, isOnline bool) error {
	if err := r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", id).
		Update("is_online", isOnline).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMUserRepository) UpdateLastConnected(ctx context.Context, id uint) error {
	now := time.Now()
	if err := r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", id).
		Update("last_connected_at", now).Error; err != nil {
		return WrapError(err)
	}
	return nil
}
