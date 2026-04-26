package services

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/protocol"
)

// OutboundService handles outbound management operations
type OutboundService struct {
	db            *gorm.DB
	configService *ConfigService
}

// NewOutboundService creates a new outbound service
func NewOutboundService(db *gorm.DB, configService *ConfigService) *OutboundService {
	return &OutboundService{
		db:            db,
		configService: configService,
	}
}

// CreateOutbound creates a new outbound and triggers config regeneration
func (s *OutboundService) CreateOutbound(outbound *models.Outbound) error {
	// Validate required fields
	if outbound.Name == "" {
		return fmt.Errorf("name is required")
	}
	if outbound.Protocol == "" {
		return fmt.Errorf("protocol is required")
	}
	if outbound.CoreID == 0 {
		return fmt.Errorf("core_id is required")
	}

	// Validate core exists
	var coreModel models.Core
	if err := s.db.First(&coreModel, outbound.CoreID).Error; err != nil {
		return fmt.Errorf("core not found: %w", err)
	}

	// Validate protocol against schema registry
	schema, ok := protocol.GetProtocolSchema(outbound.Protocol)
	if !ok {
		return fmt.Errorf("unknown protocol: %s", outbound.Protocol)
	}

	// Validate protocol supports this core
	if !protocol.ValidateProtocolForCore(outbound.Protocol, coreModel.Name) {
		return fmt.Errorf("protocol %s is not supported by core %s", outbound.Protocol, coreModel.Name)
	}

	// Validate protocol direction allows outbound
	if schema.Direction == "inbound" {
		return fmt.Errorf("protocol %s is inbound-only and cannot be used as outbound", outbound.Protocol)
	}

	// Validate config JSON if provided
	if outbound.ConfigJSON != "" {
		if err := validateJSON(outbound.ConfigJSON); err != nil {
			return fmt.Errorf("invalid config_json: %w", err)
		}
		if err := protocol.ValidateConfigJSON(outbound.Protocol, outbound.ConfigJSON); err != nil {
			return fmt.Errorf("config validation: %w", err)
		}
	} else {
		outbound.ConfigJSON = "{}"
	}

	// Check for duplicate name within same core
	var existing models.Outbound
	err := s.db.Where("name = ? AND core_id = ?", outbound.Name, outbound.CoreID).First(&existing).Error
	if err == nil {
		return fmt.Errorf("outbound with name '%s' already exists for this core", outbound.Name)
	}

	// Create outbound
	if err := s.db.Create(outbound).Error; err != nil {
		return fmt.Errorf("failed to create outbound: %w", err)
	}

	// Trigger config regeneration
	if s.configService != nil {
		if err := s.configService.RegenerateAndReload(coreModel.Name); err != nil {
			logger.Log.Warn().Err(err).Str("core", coreModel.Name).Msg("Failed to regenerate config")
		}
	}

	return nil
}

// GetOutbound retrieves an outbound by ID
func (s *OutboundService) GetOutbound(id uint) (*models.Outbound, error) {
	var outbound models.Outbound
	if err := s.db.Preload("Core").First(&outbound, id).Error; err != nil {
		return nil, fmt.Errorf("outbound not found: %w", err)
	}
	return &outbound, nil
}

// ListOutbounds returns all outbounds with optional filtering
func (s *OutboundService) ListOutbounds(coreID *uint, protocolFilter string) ([]models.Outbound, error) {
	var outbounds []models.Outbound
	query := s.db.Preload("Core")

	if coreID != nil {
		query = query.Where("core_id = ?", *coreID)
	}

	if protocolFilter != "" {
		query = query.Where("protocol = ?", protocolFilter)
	}

	if err := query.Order("priority DESC, id ASC").Find(&outbounds).Error; err != nil {
		return nil, fmt.Errorf("failed to list outbounds: %w", err)
	}

	return outbounds, nil
}

// UpdateOutbound updates an existing outbound and triggers config regeneration
func (s *OutboundService) UpdateOutbound(id uint, updates map[string]interface{}) (*models.Outbound, error) {
	var outbound models.Outbound
	if err := s.db.First(&outbound, id).Error; err != nil {
		return nil, fmt.Errorf("outbound not found: %w", err)
	}

	// Get core for this outbound
	var coreModel models.Core
	if err := s.db.First(&coreModel, outbound.CoreID).Error; err != nil {
		return nil, fmt.Errorf("core not found: %w", err)
	}

	// If protocol is being changed, validate it
	if newProtocol, ok := updates["protocol"].(string); ok && newProtocol != outbound.Protocol {
		schema, exists := protocol.GetProtocolSchema(newProtocol)
		if !exists {
			return nil, fmt.Errorf("unknown protocol: %s", newProtocol)
		}
		if !protocol.ValidateProtocolForCore(newProtocol, coreModel.Name) {
			return nil, fmt.Errorf("protocol %s is not supported by core %s", newProtocol, coreModel.Name)
		}
		if schema.Direction == "inbound" {
			return nil, fmt.Errorf("protocol %s is inbound-only and cannot be used as outbound", newProtocol)
		}
	}

	// If config_json is being changed, validate it
	if newConfig, ok := updates["config_json"].(string); ok && newConfig != "" {
		if err := validateJSON(newConfig); err != nil {
			return nil, fmt.Errorf("invalid config_json: %w", err)
		}
	}

	// If name is being changed, check for duplicates
	if newName, ok := updates["name"].(string); ok && newName != outbound.Name {
		var existing models.Outbound
		err := s.db.Where("name = ? AND core_id = ? AND id != ?", newName, outbound.CoreID, id).First(&existing).Error
		if err == nil {
			return nil, fmt.Errorf("outbound with name '%s' already exists for this core", newName)
		}
	}

	// Update outbound
	if err := s.db.Model(&outbound).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update outbound: %w", err)
	}

	// Reload the outbound to get updated values
	if err := s.db.Preload("Core").First(&outbound, id).Error; err != nil {
		return nil, fmt.Errorf("failed to reload outbound: %w", err)
	}

	// Trigger config regeneration
	if s.configService != nil {
		if err := s.configService.RegenerateAndReload(coreModel.Name); err != nil {
			logger.Log.Warn().Err(err).Str("core", coreModel.Name).Msg("Failed to regenerate config")
		}
	}

	return &outbound, nil
}

// DeleteOutbound deletes an outbound and triggers config regeneration
func (s *OutboundService) DeleteOutbound(id uint) error {
	var outbound models.Outbound
	if err := s.db.First(&outbound, id).Error; err != nil {
		return fmt.Errorf("outbound not found: %w", err)
	}

	// Get core before deleting
	var coreModel models.Core
	if err := s.db.First(&coreModel, outbound.CoreID).Error; err != nil {
		return fmt.Errorf("core not found: %w", err)
	}

	// Delete outbound
	if err := s.db.Delete(&outbound).Error; err != nil {
		return fmt.Errorf("failed to delete outbound: %w", err)
	}

	// Trigger config regeneration
	if s.configService != nil {
		if err := s.configService.RegenerateAndReload(coreModel.Name); err != nil {
			logger.Log.Warn().Err(err).Str("core", coreModel.Name).Msg("Failed to regenerate config")
		}
	}

	return nil
}

// GetOutboundsByCore returns all outbounds for a specific core
func (s *OutboundService) GetOutboundsByCore(coreID uint) ([]models.Outbound, error) {
	var outbounds []models.Outbound
	if err := s.db.Where("core_id = ?", coreID).Preload("Core").Order("priority DESC, id ASC").Find(&outbounds).Error; err != nil {
		return nil, fmt.Errorf("failed to get outbounds for core: %w", err)
	}
	return outbounds, nil
}

// validateJSON checks that a string is valid JSON
func validateJSON(s string) error {
	var js map[string]interface{}
	if err := json.Unmarshal([]byte(s), &js); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}
