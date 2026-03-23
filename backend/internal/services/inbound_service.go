package services

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
)

// InboundService handles inbound management operations
type InboundService struct {
	db               *gorm.DB
	lifecycleManager *CoreLifecycleManager
}

// NewInboundService creates a new inbound service
func NewInboundService(db *gorm.DB, lifecycleManager *CoreLifecycleManager) *InboundService {
	return &InboundService{
		db:               db,
		lifecycleManager: lifecycleManager,
	}
}

// CreateInbound creates a new inbound
func (s *InboundService) CreateInbound(inbound *models.Inbound) error {
	// Validate required fields
	if inbound.Name == "" {
		return fmt.Errorf("name is required")
	}
	if inbound.Protocol == "" {
		return fmt.Errorf("protocol is required")
	}
	if inbound.Port == 0 {
		return fmt.Errorf("port is required")
	}
	if inbound.CoreID == 0 {
		return fmt.Errorf("core_id is required")
	}

	// Check if port is already in use
	var existing models.Inbound
	err := s.db.Where("port = ? AND core_id = ?", inbound.Port, inbound.CoreID).First(&existing).Error
	if err == nil {
		return fmt.Errorf("port %d is already in use by this core", inbound.Port)
	}

	// Set defaults
	if inbound.ListenAddress == "" {
		inbound.ListenAddress = "0.0.0.0"
	}

	// Create inbound
	if err := s.db.Create(inbound).Error; err != nil {
		return fmt.Errorf("failed to create inbound: %w", err)
	}

	// Trigger core lifecycle check (auto-start if needed)
	if err := s.lifecycleManager.OnInboundCreated(inbound); err != nil {
		// Log error but don't fail the creation
		fmt.Printf("Warning: failed to start core: %v\n", err)
	}

	return nil
}

// GetInbound retrieves an inbound by ID
func (s *InboundService) GetInbound(id uint) (*models.Inbound, error) {
	var inbound models.Inbound
	if err := s.db.First(&inbound, id).Error; err != nil {
		return nil, fmt.Errorf("inbound not found: %w", err)
	}
	return &inbound, nil
}

// ListInbounds returns all inbounds with optional filtering
func (s *InboundService) ListInbounds(coreID *uint, isEnabled *bool) ([]models.Inbound, error) {
	var inbounds []models.Inbound
	query := s.db.Preload("Core")

	if coreID != nil {
		query = query.Where("core_id = ?", *coreID)
	}

	if isEnabled != nil {
		query = query.Where("is_enabled = ?", *isEnabled)
	}

	if err := query.Order("id ASC").Find(&inbounds).Error; err != nil {
		return nil, fmt.Errorf("failed to list inbounds: %w", err)
	}

	return inbounds, nil
}

// UpdateInbound updates an existing inbound
func (s *InboundService) UpdateInbound(id uint, updates map[string]interface{}) (*models.Inbound, error) {
	var inbound models.Inbound
	if err := s.db.First(&inbound, id).Error; err != nil {
		return nil, fmt.Errorf("inbound not found: %w", err)
	}

	// Store original enabled state
	wasEnabled := inbound.IsEnabled

	// Check if port is being changed and if it's available
	if newPort, ok := updates["port"].(int); ok && newPort != inbound.Port {
		var existing models.Inbound
		err := s.db.Where("port = ? AND core_id = ? AND id != ?", newPort, inbound.CoreID, id).First(&existing).Error
		if err == nil {
			return nil, fmt.Errorf("port %d is already in use", newPort)
		}
	}

	// Update inbound
	if err := s.db.Model(&inbound).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update inbound: %w", err)
	}

	// Reload the inbound to get updated values
	if err := s.db.Preload("Core").First(&inbound, id).Error; err != nil {
		return nil, fmt.Errorf("failed to reload inbound: %w", err)
	}

	// Trigger config regeneration and core lifecycle check
	if err := s.lifecycleManager.OnInboundUpdated(&inbound, wasEnabled); err != nil {
		fmt.Printf("Warning: failed to update core lifecycle: %v\n", err)
	}

	return &inbound, nil
}

// DeleteInbound deletes an inbound
func (s *InboundService) DeleteInbound(id uint) error {
	var inbound models.Inbound
	if err := s.db.First(&inbound, id).Error; err != nil {
		return fmt.Errorf("inbound not found: %w", err)
	}

	// Delete inbound
	if err := s.db.Delete(&inbound).Error; err != nil {
		return fmt.Errorf("failed to delete inbound: %w", err)
	}

	// Trigger core lifecycle check (auto-stop if no more inbounds)
	if err := s.lifecycleManager.OnInboundDeleted(&inbound); err != nil {
		fmt.Printf("Warning: failed to update core lifecycle: %v\n", err)
	}

	return nil
}

// GetInboundsByCore returns all inbounds for a specific core
func (s *InboundService) GetInboundsByCore(coreID uint) ([]models.Inbound, error) {
	var inbounds []models.Inbound
	if err := s.db.Where("core_id = ?", coreID).Preload("Core").Find(&inbounds).Error; err != nil {
		return nil, fmt.Errorf("failed to get inbounds for core: %w", err)
	}
	return inbounds, nil
}

// GetInboundsByCoreName returns all inbounds for a specific core by name
func (s *InboundService) GetInboundsByCoreName(coreName string) ([]models.Inbound, error) {
	var core models.Core
	if err := s.db.Where("name = ?", coreName).First(&core).Error; err != nil {
		return nil, fmt.Errorf("core not found: %w", err)
	}
	return s.GetInboundsByCore(core.ID)
}

// GetInboundsByUser returns all inbounds assigned to a user
func (s *InboundService) GetInboundsByUser(userID uint) ([]models.Inbound, error) {
	var mappings []models.UserInboundMapping
	if err := s.db.Where("user_id = ?", userID).Find(&mappings).Error; err != nil {
		return nil, fmt.Errorf("failed to get user inbound mappings: %w", err)
	}

	if len(mappings) == 0 {
		return []models.Inbound{}, nil
	}

	inboundIDs := make([]uint, len(mappings))
	for i, m := range mappings {
		inboundIDs[i] = m.InboundID
	}

	var inbounds []models.Inbound
	if err := s.db.Where("id IN ?", inboundIDs).Preload("Core").Find(&inbounds).Error; err != nil {
		return nil, fmt.Errorf("failed to get inbounds: %w", err)
	}

	return inbounds, nil
}

// AssignInboundToUser assigns an inbound to a user
func (s *InboundService) AssignInboundToUser(userID uint, inboundID uint) error {
	// Check if user exists
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Check if inbound exists
	var inbound models.Inbound
	if err := s.db.First(&inbound, inboundID).Error; err != nil {
		return fmt.Errorf("inbound not found: %w", err)
	}

	// Check if mapping already exists
	var existing models.UserInboundMapping
	err := s.db.Where("user_id = ? AND inbound_id = ?", userID, inboundID).First(&existing).Error
	if err == nil {
		return fmt.Errorf("user is already assigned to this inbound")
	}

	// Create mapping
	mapping := &models.UserInboundMapping{
		UserID:    userID,
		InboundID: inboundID,
	}

	if err := s.db.Create(mapping).Error; err != nil {
		return fmt.Errorf("failed to assign inbound to user: %w", err)
	}

	return nil
}

// UnassignInboundFromUser removes an inbound assignment from a user
func (s *InboundService) UnassignInboundFromUser(userID uint, inboundID uint) error {
	result := s.db.Where("user_id = ? AND inbound_id = ?", userID, inboundID).Delete(&models.UserInboundMapping{})
	if result.Error != nil {
		return fmt.Errorf("failed to unassign inbound from user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("mapping not found")
	}
	return nil
}

// ValidateInboundConfig validates the inbound configuration JSON
func (s *InboundService) ValidateInboundConfig(protocol string, configJSON string) error {
	if configJSON == "" {
		return nil // Empty config is valid
	}

	// Try to parse as JSON
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Protocol-specific validation can be added here
	// For now, just ensure it's valid JSON

	return nil
}
