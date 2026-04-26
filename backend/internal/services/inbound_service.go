package services

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/protocol"
)

// InboundService handles inbound management operations
type InboundService struct {
	db               *gorm.DB
	lifecycleManager *CoreLifecycleManager
	portManager      *PortManager
	subscriptions    *SubscriptionService
}

// NewInboundService creates a new inbound service
func NewInboundService(db *gorm.DB, lifecycleManager *CoreLifecycleManager, portManager *PortManager) *InboundService {
	return &InboundService{
		db:               db,
		lifecycleManager: lifecycleManager,
		portManager:      portManager,
	}
}

// SetSubscriptionService injects SubscriptionService for cache invalidation (breaks circular dep)
func (s *InboundService) SetSubscriptionService(subs *SubscriptionService) {
	s.subscriptions = subs
}

func (s *InboundService) invalidateInboundUsersCache(inboundID uint) {
	if s.subscriptions == nil {
		return
	}
	users, err := s.GetInboundUsers(inboundID)
	if err != nil {
		return
	}
	for _, user := range users {
		s.subscriptions.InvalidateUserCache(user.ID)
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

	if inbound.ConfigJSON != "" {
		if err := s.ValidateInboundConfig(inbound.Protocol, inbound.ConfigJSON); err != nil {
			return fmt.Errorf("invalid config: %w", err)
		}
	}

	// Check if port is already in use
	if s.portManager != nil {
		available, reason, err := s.portManager.IsPortAvailable(inbound.Port, nil)
		if err != nil {
			return fmt.Errorf("failed to check port: %w", err)
		}
		if !available {
			return fmt.Errorf("%s", reason)
		}
	} else {
		// Fallback if port manager is not available (should not happen in prod)
		var existing models.Inbound
		err := s.db.Where("port = ? AND core_id = ?", inbound.Port, inbound.CoreID).First(&existing).Error
		if err == nil {
			return fmt.Errorf("port %d is already in use by this core", inbound.Port)
		}
	}

	// Validate TLS certificate binding
	if inbound.TLSEnabled && inbound.TLSCertID != nil {
		var cert models.Certificate
		if err := s.db.First(&cert, *inbound.TLSCertID).Error; err != nil {
			return fmt.Errorf("certificate not found (id=%d): %w", *inbound.TLSCertID, err)
		}
		if cert.Status == models.CertificateStatusExpired || cert.Status == models.CertificateStatusRevoked {
			return fmt.Errorf("certificate %s is %s and cannot be used", cert.Domain, cert.Status)
		}
	}

	// Clear certificate binding when TLS is disabled
	if !inbound.TLSEnabled {
		inbound.TLSCertID = nil
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
	if s.lifecycleManager != nil {
		if err := s.lifecycleManager.OnInboundCreated(inbound); err != nil {
			// Log error but don't fail the creation
			logger.Log.Warn().Err(err).Uint("inbound_id", inbound.ID).Msg("Failed to notify lifecycle manager of inbound creation")
		}
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

// ListInboundsPaginated returns paginated inbounds with optional filtering
func (s *InboundService) ListInboundsPaginated(coreID *uint, isEnabled *bool, page, pageSize int) ([]models.Inbound, int64, error) {
	var inbounds []models.Inbound
	var total int64
	query := s.db.Model(&models.Inbound{})

	if coreID != nil {
		query = query.Where("core_id = ?", *coreID)
	}

	if isEnabled != nil {
		query = query.Where("is_enabled = ?", *isEnabled)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count inbounds: %w", err)
	}

	offset := (page - 1) * pageSize
	if err := query.Preload("Core").
		Offset(offset).
		Limit(pageSize).
		Order("id ASC").
		Find(&inbounds).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list inbounds: %w", err)
	}

	return inbounds, total, nil
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
	if portVal, ok := updates["port"]; ok {
		var newPort int
		var hasPort bool
		switch v := portVal.(type) {
		case float64:
			newPort, hasPort = int(v), true
		case int:
			newPort, hasPort = v, true
		case uint:
			newPort, hasPort = int(v), true
		}
		if hasPort && newPort != inbound.Port {
			if s.portManager != nil {
				available, reason, err := s.portManager.IsPortAvailable(newPort, &id)
				if err != nil {
					return nil, fmt.Errorf("failed to check port: %w", err)
				}
				if !available {
					return nil, fmt.Errorf("%s", reason)
				}
			} else {
				var existing models.Inbound
				err := s.db.Where("port = ? AND core_id = ? AND id != ?", newPort, inbound.CoreID, id).First(&existing).Error
				if err == nil {
					return nil, fmt.Errorf("port %d is already in use", newPort)
				}
			}
		}
	}

	if configJSON, ok := updates["config_json"]; ok {
		if configStr, isString := configJSON.(string); isString && configStr != "" {
			if err := s.ValidateInboundConfig(inbound.Protocol, configStr); err != nil {
				return nil, fmt.Errorf("invalid config: %w", err)
			}
		}
	}

	// Validate TLS certificate binding on update
	if tlsCertID, ok := updates["tls_cert_id"]; ok && tlsCertID != nil {
		var certIDVal uint
		switch v := tlsCertID.(type) {
		case float64:
			certIDVal = uint(v)
		case uint:
			certIDVal = v
		case int:
			certIDVal = uint(v)
		}
		if certIDVal > 0 {
			var cert models.Certificate
			if err := s.db.First(&cert, certIDVal).Error; err != nil {
				return nil, fmt.Errorf("certificate not found (id=%d): %w", certIDVal, err)
			}
			if cert.Status == models.CertificateStatusExpired || cert.Status == models.CertificateStatusRevoked {
				return nil, fmt.Errorf("certificate %s is %s and cannot be used", cert.Domain, cert.Status)
			}
		}
	}

	// Clear certificate binding when TLS is disabled
	if tlsEnabled, ok := updates["tls_enabled"]; ok {
		if enabled, isBool := tlsEnabled.(bool); isBool && !enabled {
			updates["tls_cert_id"] = nil
		}
	}

	// Update inbound
	if err := s.db.Model(&inbound).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update inbound: %w", err)
	}

	s.invalidateInboundUsersCache(id)

	// Reload the inbound to get updated values
	if err := s.db.Preload("Core").First(&inbound, id).Error; err != nil {
		return nil, fmt.Errorf("failed to reload inbound: %w", err)
	}

	// Trigger config regeneration and core lifecycle check
	if s.lifecycleManager != nil {
		if err := s.lifecycleManager.OnInboundUpdated(&inbound, wasEnabled); err != nil {
			logger.Log.Warn().Err(err).Msg("Failed to update core lifecycle")
		}
	}

	return &inbound, nil
}

// DeleteInbound deletes an inbound
func (s *InboundService) DeleteInbound(id uint) error {
	var inbound models.Inbound
	if err := s.db.First(&inbound, id).Error; err != nil {
		return fmt.Errorf("inbound not found: %w", err)
	}

	s.invalidateInboundUsersCache(id)

	// Delete inbound
	if err := s.db.Delete(&inbound).Error; err != nil {
		return fmt.Errorf("failed to delete inbound: %w", err)
	}

	// Trigger core lifecycle check (auto-stop if no more inbounds)
	if s.lifecycleManager != nil {
		if err := s.lifecycleManager.OnInboundDeleted(&inbound); err != nil {
			logger.Log.Warn().Err(err).Msg("Failed to update core lifecycle")
		}
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

	if s.subscriptions != nil {
		s.subscriptions.InvalidateUserCache(userID)
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

	if s.subscriptions != nil {
		s.subscriptions.InvalidateUserCache(userID)
	}

	return nil
}

// ValidateInboundConfig validates the inbound configuration JSON
func (s *InboundService) ValidateInboundConfig(protocolName string, configJSON string) error {
	if configJSON == "" {
		return nil
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	if err := protocol.ValidateConfigJSON(protocolName, configJSON); err != nil {
		return err
	}

	return nil
}

// GetInboundUsers returns all users assigned to a specific inbound
func (s *InboundService) GetInboundUsers(inboundID uint) ([]models.User, error) {
	// Check inbound exists
	var inbound models.Inbound
	if err := s.db.First(&inbound, inboundID).Error; err != nil {
		return nil, fmt.Errorf("inbound not found: %w", err)
	}

	var mappings []models.UserInboundMapping
	if err := s.db.Preload("User").Where("inbound_id = ?", inboundID).Find(&mappings).Error; err != nil {
		return nil, fmt.Errorf("failed to get inbound users: %w", err)
	}

	users := make([]models.User, 0, len(mappings))
	for _, mapping := range mappings {
		if mapping.User != nil {
			users = append(users, *mapping.User)
		}
	}

	return users, nil
}

// BulkAssignUsers adds and removes multiple users from an inbound in one operation
func (s *InboundService) BulkAssignUsers(inboundID uint, addUserIDs, removeUserIDs []uint) (int, int, error) {
	// Check inbound exists
	var inbound models.Inbound
	if err := s.db.First(&inbound, inboundID).Error; err != nil {
		return 0, 0, fmt.Errorf("inbound not found: %w", err)
	}

	added := 0
	removed := 0

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Remove users
		for _, userID := range removeUserIDs {
			result := tx.Where("user_id = ? AND inbound_id = ?", userID, inboundID).Delete(&models.UserInboundMapping{})
			if result.RowsAffected > 0 {
				removed++
			}
		}

		// Add users
		for _, userID := range addUserIDs {
			// Check user exists
			var user models.User
			if err := tx.First(&user, userID).Error; err != nil {
				continue // Skip invalid users
			}

			// Check if mapping already exists
			var existing models.UserInboundMapping
			err := tx.Where("user_id = ? AND inbound_id = ?", userID, inboundID).First(&existing).Error
			if err == nil {
				continue // Already assigned
			}

			mapping := &models.UserInboundMapping{
				UserID:    userID,
				InboundID: inboundID,
			}
			if err := tx.Create(mapping).Error; err == nil {
				added++
			}
		}

		return nil
	})

	if err != nil {
		return 0, 0, fmt.Errorf("failed to bulk assign users: %w", err)
	}

	if s.subscriptions != nil {
		for _, userID := range addUserIDs {
			s.subscriptions.InvalidateUserCache(userID)
		}
		for _, userID := range removeUserIDs {
			s.subscriptions.InvalidateUserCache(userID)
		}
	}

	return added, removed, nil
}
