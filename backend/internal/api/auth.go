package api

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/auth"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services"
)

type AuthHandler struct {
	db                  *gorm.DB
	tokenService        *auth.TokenService
	notificationService *services.NotificationService
}

func NewAuthHandler(db *gorm.DB, tokenService *auth.TokenService, notificationService *services.NotificationService) *AuthHandler {
	return &AuthHandler{
		db:                  db,
		tokenService:        tokenService,
		notificationService: notificationService,
	}
}

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int64     `json:"expires_in"`
	Admin        AdminInfo `json:"admin"`
}

type AdminInfo struct {
	ID           uint   `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	IsSuperAdmin bool   `json:"is_super_admin"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// Login handles admin login
func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req LoginRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Record login attempt
	attempt := models.LoginAttempt{
		IPAddress: c.IP(),
		Username:  req.Username,
		Success:   false,
		UserAgent: c.Get("User-Agent"),
	}

	// Find admin by username
	var admin models.Admin
	if err := h.db.Where("username = ? AND is_active = ?", req.Username, true).First(&admin).Error; err != nil {
		h.db.Create(&attempt)

		// Check for multiple failed attempts
		h.checkFailedLoginAttempts(req.Username, c.IP())

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid username or password",
		})
	}

	// Verify password
	valid, err := auth.VerifyPassword(req.Password, admin.PasswordHash)
	if err != nil || !valid {
		h.db.Create(&attempt)

		// Check for multiple failed attempts
		h.checkFailedLoginAttempts(req.Username, c.IP())

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid username or password",
		})
	}

	// Generate tokens
	accessToken, err := h.tokenService.GenerateAccessToken(admin.ID, admin.Username, admin.IsSuperAdmin)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate access token",
		})
	}

	refreshToken, err := h.tokenService.GenerateRefreshToken()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate refresh token",
		})
	}

	// Hash refresh token before storing
	hash := sha256.Sum256([]byte(refreshToken))
	tokenHash := hex.EncodeToString(hash[:])

	// Store refresh token in database
	refreshTokenModel := models.RefreshToken{
		AdminID:   admin.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(h.tokenService.GetRefreshTokenTTL()),
		UserAgent: c.Get("User-Agent"),
		IPAddress: c.IP(),
	}
	if err := h.db.Create(&refreshTokenModel).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to store refresh token",
		})
	}

	// Update last login time
	now := time.Now()
	admin.LastLoginAt = &now
	h.db.Save(&admin)

	// Record successful login attempt
	attempt.Success = true
	h.db.Create(&attempt)

	return c.JSON(LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(h.tokenService.GetRefreshTokenTTL().Seconds()),
		Admin: AdminInfo{
			ID:           admin.ID,
			Username:     admin.Username,
			Email:        admin.Email,
			IsSuperAdmin: admin.IsSuperAdmin,
		},
	})
}

// Refresh handles token refresh
func (h *AuthHandler) Refresh(c fiber.Ctx) error {
	var req RefreshRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Hash the provided refresh token
	hash := sha256.Sum256([]byte(req.RefreshToken))
	tokenHash := hex.EncodeToString(hash[:])

	// Find refresh token in database
	var refreshToken models.RefreshToken
	if err := h.db.Preload("Admin").Where("token_hash = ? AND revoked = ? AND expires_at > ?",
		tokenHash, false, time.Now()).First(&refreshToken).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired refresh token",
		})
	}

	// Generate new access token
	accessToken, err := h.tokenService.GenerateAccessToken(
		refreshToken.Admin.ID,
		refreshToken.Admin.Username,
		refreshToken.Admin.IsSuperAdmin,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate access token",
		})
	}

	return c.JSON(fiber.Map{
		"access_token": accessToken,
		"expires_in":   int64(h.tokenService.GetRefreshTokenTTL().Seconds()),
	})
}

// Logout handles admin logout
func (h *AuthHandler) Logout(c fiber.Ctx) error {
	var req RefreshRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Hash the refresh token
	hash := sha256.Sum256([]byte(req.RefreshToken))
	tokenHash := hex.EncodeToString(hash[:])

	// Revoke the refresh token
	if err := h.db.Model(&models.RefreshToken{}).
		Where("token_hash = ?", tokenHash).
		Update("revoked", true).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to revoke token",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

// Me returns current admin info
func (h *AuthHandler) Me(c fiber.Ctx) error {
	adminID := c.Locals("admin_id").(uint)

	var admin models.Admin
	if err := h.db.First(&admin, adminID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Admin not found",
		})
	}

	return c.JSON(AdminInfo{
		ID:           admin.ID,
		Username:     admin.Username,
		Email:        admin.Email,
		IsSuperAdmin: admin.IsSuperAdmin,
	})
}

// checkFailedLoginAttempts checks for multiple failed login attempts and sends notification
func (h *AuthHandler) checkFailedLoginAttempts(username, ip string) {
	if h.notificationService == nil {
		return
	}

	// Count failed attempts from this IP in the last hour
	var count int64
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	h.db.Model(&models.LoginAttempt{}).
		Where("ip_address = ? AND success = ? AND created_at > ?", ip, false, oneHourAgo).
		Count(&count)

	// Send notification after 5 failed attempts
	if count >= 5 {
		h.notificationService.NotifyFailedLogin(ip, username, int(count))
	}
}
