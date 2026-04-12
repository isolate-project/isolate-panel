package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/models"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

type UserService struct {
	db                  *gorm.DB
	notificationService *NotificationService
	subscriptions       *SubscriptionService
}

func NewUserService(db *gorm.DB, notificationService *NotificationService) *UserService {
	return &UserService{
		db:                  db,
		notificationService: notificationService,
	}
}

// SetSubscriptionService injects SubscriptionService for cache invalidation (breaks circular dep)
func (us *UserService) SetSubscriptionService(subs *SubscriptionService) {
	us.subscriptions = subs
}

type CreateUserRequest struct {
	Username          string `json:"username" validate:"required,min=3,max=50"`
	Email             string `json:"email" validate:"omitempty,email"`
	Password          string `json:"password"`
	TrafficLimitBytes *int64 `json:"traffic_limit_bytes"`
	ExpiryDays        *int   `json:"expiry_days"`
	InboundIDs        []uint `json:"inbound_ids"`
}

type UpdateUserRequest struct {
	Username          *string `json:"username" validate:"omitempty,min=3,max=50"`
	Email             *string `json:"email" validate:"omitempty,email"`
	Password          *string `json:"password" validate:"omitempty,min=8"`
	TrafficLimitBytes *int64  `json:"traffic_limit_bytes"`
	ExpiryDays        *int    `json:"expiry_days"`
	IsActive          *bool   `json:"is_active"`
	InboundIDs        []uint  `json:"inbound_ids"`
}

type UserResponse struct {
	ID                uint    `json:"id"`
	Username          string  `json:"username"`
	Email             string  `json:"email"`
	UUID              string  `json:"uuid"`
	Token             *string `json:"token,omitempty"`
	SubscriptionToken string  `json:"subscription_token"`
	TrafficLimitBytes *int64  `json:"traffic_limit_bytes"`
	TrafficUsedBytes  int64   `json:"traffic_used_bytes"`
	ExpiryDate        *string `json:"expiry_date"`
	IsActive          bool    `json:"is_active"`
	IsOnline          bool    `json:"is_online"`
	CreatedAt         string  `json:"created_at"`
}

// CreateUserResponse includes sensitive credentials — returned only on Create/Regenerate
type CreateUserResponse struct {
	UserResponse
	Password string  `json:"password"`
}

// CreateUser creates a new user with auto-generated credentials
func (us *UserService) CreateUser(req *CreateUserRequest, adminID uint) (*models.User, error) {
	// Validate input
	if len(req.Username) < 3 {
		return nil, fmt.Errorf("validation error: username must be at least 3 characters")
	}
	if len(req.Username) > 50 {
		return nil, fmt.Errorf("validation error: username must be less than 50 characters")
	}
	if req.Email != "" {
		if !isValidEmail(req.Email) {
			return nil, fmt.Errorf("validation error: invalid email format")
		}
	}
	// Validate password length
	if req.Password != "" && len(req.Password) < 8 {
		return nil, fmt.Errorf("validation error: password must be at least 8 characters")
	}
	// Auto-generate protocol password if not provided
	if req.Password == "" {
		generated, err := generateToken(16)
		if err != nil {
			return nil, fmt.Errorf("failed to generate password: %w", err)
		}
		req.Password = generated
	}

	// Check if username already exists
	var existing models.User
	if err := us.db.Where("username = ?", req.Username).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("username already exists")
	}

	// Generate UUID v4
	userUUID := uuid.New().String()

	// Generate subscription token
	subToken, err := generateToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate subscription token: %w", err)
	}

	// Generate TUIC token (optional)
	var tuicToken *string
	token, err := generateToken(16)
	if err == nil {
		tuicToken = &token
	}

	// Calculate expiry date
	var expiryDate *time.Time
	if req.ExpiryDays != nil && *req.ExpiryDays > 0 {
		expiry := time.Now().AddDate(0, 0, *req.ExpiryDays)
		expiryDate = &expiry
	}

	// Create user
	user := &models.User{
		Username:          req.Username,
		Email:             req.Email,
		UUID:              userUUID,
		Password:          req.Password,
		Token:             tuicToken,
		SubscriptionToken: subToken,
		TrafficLimitBytes: req.TrafficLimitBytes,
		TrafficUsedBytes:  0,
		ExpiryDate:        expiryDate,
		IsActive:          true,
		IsOnline:          false,
		CreatedByAdminID:  &adminID,
	}

	err = us.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		for _, inboundID := range req.InboundIDs {
			var count int64
			if err := tx.Model(&models.Inbound{}).Where("id = ?", inboundID).Count(&count).Error; err != nil {
				return fmt.Errorf("failed to verify inbound %d: %w", inboundID, err)
			}
			if count == 0 {
				return fmt.Errorf("inbound %d not found", inboundID)
			}
			mapping := &models.UserInboundMapping{
				UserID:    user.ID,
				InboundID: inboundID,
			}
			if err := tx.Create(mapping).Error; err != nil {
				return fmt.Errorf("failed to create inbound mapping: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Send notification
	if us.notificationService != nil {
		us.notificationService.NotifyUserCreated(user)
	}

	return user, nil
}

// GetUser retrieves a user by ID
func (us *UserService) GetUser(id uint) (*models.User, error) {
	var user models.User
	if err := us.db.Preload("CreatedByAdmin").First(&user, id).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &user, nil
}

// ListUsers retrieves users with pagination, optional search and status filter
func (us *UserService) ListUsers(page, pageSize int, search, status string) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	query := us.db.Model(&models.User{})

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("username LIKE ? OR email LIKE ? OR uuid LIKE ?", like, like, like)
	}
	if status == "active" {
		query = query.Where("is_active = ?", true)
	} else if status == "inactive" {
		query = query.Where("is_active = ?", false)
	}

	// Count total with filters
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	if err := query.Preload("CreatedByAdmin").
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	return users, total, nil
}

// UpdateUser updates a user
func (us *UserService) UpdateUser(id uint, req *UpdateUserRequest) (*models.User, error) {
	var user models.User
	if err := us.db.First(&user, id).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Update fields
	if req.Username != nil {
		user.Username = *req.Username
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.Password != nil {
		user.Password = *req.Password
	}
	if req.TrafficLimitBytes != nil {
		user.TrafficLimitBytes = req.TrafficLimitBytes
	}
	if req.ExpiryDays != nil {
		if *req.ExpiryDays > 0 {
			expiry := time.Now().AddDate(0, 0, *req.ExpiryDays)
			user.ExpiryDate = &expiry
		} else {
			user.ExpiryDate = nil // unlimited
		}
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	err := us.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&user).Error; err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}

		if req.InboundIDs != nil {
			if err := tx.Where("user_id = ?", user.ID).Delete(&models.UserInboundMapping{}).Error; err != nil {
				return fmt.Errorf("failed to delete old mappings: %w", err)
			}
			for _, inboundID := range req.InboundIDs {
				var count int64
				if err := tx.Model(&models.Inbound{}).Where("id = ?", inboundID).Count(&count).Error; err != nil {
					return fmt.Errorf("failed to verify inbound %d: %w", inboundID, err)
				}
				if count == 0 {
					return fmt.Errorf("inbound %d not found", inboundID)
				}
				mapping := &models.UserInboundMapping{
					UserID:    user.ID,
					InboundID: inboundID,
				}
				if err := tx.Create(mapping).Error; err != nil {
					return fmt.Errorf("failed to create inbound mapping: %w", err)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Invalidate subscription cache on any user change
	if us.subscriptions != nil {
		us.subscriptions.InvalidateUserCache(user.ID)
	}

	return &user, nil
}

// DeleteUser deletes a user
func (us *UserService) DeleteUser(id uint) error {
	var user models.User
	if err := us.db.First(&user, id).Error; err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Invalidate subscription cache before deletion
	if us.subscriptions != nil {
		us.subscriptions.InvalidateUserCache(user.ID)
	}

	// Delete user (cascades to mappings)
	if err := us.db.Delete(&user).Error; err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Send notification
	if us.notificationService != nil {
		us.notificationService.NotifyUserDeleted(&user)
	}

	return nil
}

// RegenerateCredentials regenerates user credentials
func (us *UserService) RegenerateCredentials(id uint) (*models.User, error) {
	var user models.User
	if err := us.db.First(&user, id).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Regenerate UUID
	user.UUID = uuid.New().String()

	// Regenerate subscription token
	subToken, err := generateToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate subscription token: %w", err)
	}
	user.SubscriptionToken = subToken

	// Regenerate TUIC token
	token, err := generateToken(16)
	if err == nil {
		user.Token = &token
	}

	if err := us.db.Save(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &user, nil
}

// GetUserInbounds retrieves inbounds for a user
func (us *UserService) GetUserInbounds(userID uint) ([]models.Inbound, error) {
	var mappings []models.UserInboundMapping
	if err := us.db.Preload("Inbound").Where("user_id = ?", userID).Find(&mappings).Error; err != nil {
		return nil, fmt.Errorf("failed to get user inbounds: %w", err)
	}

	inbounds := make([]models.Inbound, 0, len(mappings))
	for _, mapping := range mappings {
		if mapping.Inbound != nil {
			inbounds = append(inbounds, *mapping.Inbound)
		}
	}

	return inbounds, nil
}

// CheckExpiringUsers checks for users expiring in 7, 3, and 1 days and sends notifications
func (us *UserService) CheckExpiringUsers() {
	if us.notificationService == nil {
		return
	}

	var users []models.User
	// Get users with expiry dates in the next 7 days
	if err := us.db.Where("expiry_date IS NOT NULL AND is_active = ?", true).
		Find(&users).Error; err != nil {
		return
	}

	now := time.Now()
	for i := range users {
		user := &users[i]
		if user.ExpiryDate == nil {
			continue
		}

		daysLeft := int(user.ExpiryDate.Sub(now).Hours() / 24)

		// Send notification for 7, 3, and 1 days — but only if not already notified for this threshold
		if daysLeft == 7 || daysLeft == 3 || daysLeft == 1 {
			if user.LastExpiryNotifiedDays == nil || *user.LastExpiryNotifiedDays != daysLeft {
				us.notificationService.NotifyExpiryWarning(user, daysLeft)
				us.db.Model(user).Update("last_expiry_notified_days", daysLeft)
			}
		}
	}
}

// generateToken generates a random hex token
func generateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
