package repository

import (
	"context"

	"github.com/isolate-project/isolate-panel/internal/models"
	"gorm.io/gorm"
)

// GORMOutboundRepository implements OutboundRepository using GORM
type GORMOutboundRepository struct {
	db *gorm.DB
}

// NewGORMOutboundRepository creates a new GORM-based OutboundRepository
func NewGORMOutboundRepository(db *gorm.DB) OutboundRepository {
	return &GORMOutboundRepository{db: db}
}

func (r *GORMOutboundRepository) Create(ctx context.Context, outbound *models.Outbound) error {
	if err := r.db.WithContext(ctx).Create(outbound).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMOutboundRepository) GetByID(ctx context.Context, id uint) (*models.Outbound, error) {
	var outbound models.Outbound
	if err := r.db.WithContext(ctx).First(&outbound, id).Error; err != nil {
		return nil, WrapError(err)
	}
	return &outbound, nil
}

func (r *GORMOutboundRepository) Update(ctx context.Context, outbound *models.Outbound) error {
	if err := r.db.WithContext(ctx).Save(outbound).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMOutboundRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&models.Outbound{}, id).Error; err != nil {
		return WrapError(err)
	}
	return nil
}

func (r *GORMOutboundRepository) List(ctx context.Context, coreID *uint, protocolFilter string) ([]models.Outbound, error) {
	var outbounds []models.Outbound

	query := r.db.WithContext(ctx).Model(&models.Outbound{})

	if coreID != nil {
		query = query.Where("core_id = ?", *coreID)
	}

	if protocolFilter != "" {
		query = query.Where("protocol = ?", protocolFilter)
	}

	if err := query.Order("priority DESC, created_at DESC").Find(&outbounds).Error; err != nil {
		return nil, WrapError(err)
	}

	return outbounds, nil
}

func (r *GORMOutboundRepository) GetByCore(ctx context.Context, coreID uint) ([]models.Outbound, error) {
	var outbounds []models.Outbound
	if err := r.db.WithContext(ctx).
		Where("core_id = ?", coreID).
		Order("priority DESC, created_at DESC").
		Find(&outbounds).Error; err != nil {
		return nil, WrapError(err)
	}
	return outbounds, nil
}
