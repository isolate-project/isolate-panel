package repository

import (
	"context"

	"github.com/isolate-project/isolate-panel/internal/models"
	"gorm.io/gorm"
)

// GORMInboundRepository implements InboundRepository using GORM
type GORMInboundRepository struct {
	db *gorm.DB
}

// NewGORMInboundRepository creates a new GORM-based InboundRepository
func NewGORMInboundRepository(db *gorm.DB) InboundRepository {
	return &GORMInboundRepository{db: db}
}

func (r *GORMInboundRepository) Create(ctx context.Context, inbound *models.Inbound) error {
	if err := r.db.WithContext(ctx).Create(inbound).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMInboundRepository) GetByID(ctx context.Context, id uint) (*models.Inbound, error) {
	var inbound models.Inbound
	if err := r.db.WithContext(ctx).First(&inbound, id).Error; err != nil {
		return nil, WrapError(err)
	}
	return &inbound, nil
}

func (r *GORMInboundRepository) Update(ctx context.Context, inbound *models.Inbound) error {
	if err := r.db.WithContext(ctx).Save(inbound).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMInboundRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&models.Inbound{}, id).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMInboundRepository) List(ctx context.Context, coreID *uint, isEnabled *bool) ([]models.Inbound, error) {
	var inbounds []models.Inbound

	query := r.db.WithContext(ctx).Model(&models.Inbound{})

	if coreID != nil {
		query = query.Where("core_id = ?", *coreID)
	}

	if isEnabled != nil {
		query = query.Where("is_enabled = ?", *isEnabled)
	}

	if err := query.Order("created_at DESC").Find(&inbounds).Error; err != nil {
		return nil, WrapError(err)
	}

	return inbounds, nil
}

func (r *GORMInboundRepository) ListPaginated(ctx context.Context, coreID *uint, isEnabled *bool, offset, limit int) ([]models.Inbound, int64, error) {
	var inbounds []models.Inbound
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Inbound{})

	if coreID != nil {
		query = query.Where("core_id = ?", *coreID)
	}

	if isEnabled != nil {
		query = query.Where("is_enabled = ?", *isEnabled)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, WrapError(err)
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&inbounds).Error; err != nil {
		return nil, 0, WrapError(err)
	}

	return inbounds, total, nil
}

func (r *GORMInboundRepository) GetByCore(ctx context.Context, coreID uint) ([]models.Inbound, error) {
	var inbounds []models.Inbound
	if err := r.db.WithContext(ctx).
		Where("core_id = ?", coreID).
		Order("created_at DESC").
		Find(&inbounds).Error; err != nil {
		return nil, WrapError(err)
	}
	return inbounds, nil
}

func (r *GORMInboundRepository) GetByCoreName(ctx context.Context, coreName string) ([]models.Inbound, error) {
	var inbounds []models.Inbound
	if err := r.db.WithContext(ctx).
		Joins("JOIN cores ON cores.id = inbounds.core_id").
		Where("cores.name = ?", coreName).
		Order("inbounds.created_at DESC").
		Find(&inbounds).Error; err != nil {
		return nil, WrapError(err)
	}
	return inbounds, nil
}

func (r *GORMInboundRepository) GetByUser(ctx context.Context, userID uint) ([]models.Inbound, error) {
	var inbounds []models.Inbound
	if err := r.db.WithContext(ctx).
		Joins("JOIN user_inbound_mapping ON user_inbound_mapping.inbound_id = inbounds.id").
		Where("user_inbound_mapping.user_id = ?", userID).
		Order("inbounds.created_at DESC").
		Find(&inbounds).Error; err != nil {
		return nil, WrapError(err)
	}
	return inbounds, nil
}

func (r *GORMInboundRepository) AssignToUser(ctx context.Context, userID, inboundID uint) error {
	mapping := models.UserInboundMapping{
		UserID:    userID,
		InboundID: inboundID,
	}
	if err := r.db.WithContext(ctx).Create(&mapping).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMInboundRepository) UnassignFromUser(ctx context.Context, userID, inboundID uint) error {
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND inbound_id = ?", userID, inboundID).
		Delete(&models.UserInboundMapping{}).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMInboundRepository) GetUsers(ctx context.Context, inboundID uint) ([]models.User, error) {
	var users []models.User
	if err := r.db.WithContext(ctx).
		Joins("JOIN user_inbound_mapping ON user_inbound_mapping.user_id = users.id").
		Where("user_inbound_mapping.inbound_id = ?", inboundID).
		Order("users.created_at DESC").
		Find(&users).Error; err != nil {
		return nil, WrapError(err)
	}
	return users, nil
}

func (r *GORMInboundRepository) BulkAssign(ctx context.Context, inboundID uint, addUserIDs, removeUserIDs []uint) (added, removed int, err error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return 0, 0, WrapError(tx.Error)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit().Error
			if err != nil {
				err = WrapError(err)
			}
		}
	}()

	// Remove users
	if len(removeUserIDs) > 0 {
		result := tx.Where("inbound_id = ? AND user_id IN ?", inboundID, removeUserIDs).
			Delete(&models.UserInboundMapping{})
		if result.Error != nil {
			err = result.Error
			return
		}
		removed = int(result.RowsAffected)
	}

	// Add users
	for _, userID := range addUserIDs {
		mapping := models.UserInboundMapping{
			UserID:    userID,
			InboundID: inboundID,
		}
		if err = tx.Create(&mapping).Error; err != nil {
			return
		}
		added++
	}

	return added, removed, nil
}
