package api

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/gofiber/fiber/v3"
	"github.com/pquerna/otp/totp"
	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/auth"
	"github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services"
)

const (
	SessionCookieName = "__Host-session"
	SessionCookieTTL  = 7 * 24 * time.Hour
)

type AuthHandler struct {
	db                  *gorm.DB
	tokenService        *auth.TokenService
	sessionManager      *auth.BFFSessionManager
	notificationService *services.NotificationService
	webAuthnService     *auth.WebAuthnService
}

func NewAuthHandler(db *gorm.DB, tokenService *auth.TokenService, sessionManager *auth.BFFSessionManager, notificationService *services.NotificationService, webAuthnService *auth.WebAuthnService) *AuthHandler {
	return &AuthHandler{
		db:                  db,
		tokenService:        tokenService,
		sessionManager:      sessionManager,
		notificationService: notificationService,
		webAuthnService:     webAuthnService,
	}
}

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
	TotpCode string `json:"totp_code"` // required when TOTP is enabled
}

type LoginResponse struct {
	AccessToken         string    `json:"access_token"`
	RefreshToken        string    `json:"refresh_token"`
	ExpiresIn           int64     `json:"expires_in"`
	Admin               AdminInfo `json:"admin"`
	MustChangePassword  bool      `json:"must_change_password"`
}

type AdminInfo struct {
	ID                  uint   `json:"id"`
	Username            string `json:"username"`
	Email               string `json:"email"`
	IsSuperAdmin        bool   `json:"is_super_admin"`
	MustChangePassword  bool   `json:"must_change_password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// Login handles admin login
//
// @Summary      Login
// @Description  Authenticate with username and password. If TOTP is enabled, provide totp_code as well. Returns requires_totp:true when TOTP code is missing.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body  LoginRequest  true  "Login credentials"
// @Success      200   {object}  LoginResponse
// @Failure      401   {object}  map[string]interface{}
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req LoginRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Check if IP is temporarily blocked due to too many failed attempts
	failedCount, _ := h.countFailedAttempts(c.IP(), 15*time.Minute)
	if failedCount >= 5 {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": "Too many failed login attempts. Try again later.",
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
		if dbErr := h.db.Create(&attempt).Error; dbErr != nil {
			logger.Log.Error().Err(dbErr).Msg("Failed to record login attempt")
		}

		logger.Log.Warn().
			Str("event", "auth_failure").
			Str("username", req.Username).
			Str("ip", c.IP()).
			Str("user_agent", c.Get("User-Agent")).
			Str("reason", "invalid_credentials").
			Msg("Login failed")

		// Check for multiple failed attempts
		h.checkFailedLoginAttempts(req.Username, c.IP())

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid username or password",
		})
	}

	// Verify password
	valid, err := auth.VerifyPassword(req.Password, admin.PasswordHash)
	if err != nil || !valid {
		if dbErr := h.db.Create(&attempt).Error; dbErr != nil {
			logger.Log.Error().Err(dbErr).Msg("Failed to record login attempt")
		}
		h.checkFailedLoginAttempts(req.Username, c.IP())
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid username or password",
		})
	}

	// Rehash password if it was created with legacy Argon2id parameters
	if auth.NeedsRehash(req.Password, admin.PasswordHash) {
		if newHash, err := auth.HashPassword(req.Password); err == nil {
			if dbErr := h.db.Model(&admin).Update("password_hash", newHash).Error; dbErr != nil {
				logger.Log.Error().Err(dbErr).Msg("Failed to rehash password")
			}
		}
	}

	// TOTP check
	if admin.TOTPEnabled {
		if req.TotpCode == "" {
			return c.JSON(fiber.Map{"requires_totp": true})
		}
		if !totp.Validate(req.TotpCode, admin.TOTPSecret) {
			if dbErr := h.db.Create(&attempt).Error; dbErr != nil {
				logger.Log.Error().Err(dbErr).Msg("Failed to record login attempt")
			}
			logger.Log.Warn().
				Str("event", "auth_failure").
				Str("username", req.Username).
				Str("ip", c.IP()).
				Str("user_agent", c.Get("User-Agent")).
				Str("reason", "invalid_totp").
				Msg("Login failed")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid TOTP code",
			})
		}
	}

	accessToken, err := h.tokenService.GenerateAccessTokenWithPermissions(admin.ID, admin.Username, admin.IsSuperAdmin, admin.MustChangePassword, admin.Permissions)
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
	if dbErr := h.db.Save(&admin).Error; dbErr != nil {
		logger.Log.Error().Err(dbErr).Msg("Failed to update last login time")
	}

	// Record successful login attempt
	attempt.Success = true
	if dbErr := h.db.Create(&attempt).Error; dbErr != nil {
		logger.Log.Error().Err(dbErr).Msg("Failed to record login attempt")
	}

	return c.JSON(LoginResponse{
		AccessToken:         accessToken,
		RefreshToken:        refreshToken,
		ExpiresIn:           int64(h.tokenService.GetAccessTokenTTL().Seconds()),
		Admin: AdminInfo{
			ID:           admin.ID,
			Username:     admin.Username,
			Email:        admin.Email,
			IsSuperAdmin: admin.IsSuperAdmin,
		},
		MustChangePassword: admin.MustChangePassword,
	})
}

// Refresh handles token refresh
//
// @Summary      Refresh access token
// @Description  Exchange a valid refresh token for a new access token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body  RefreshRequest  true  "Refresh token"
// @Success      200   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Router       /auth/refresh [post]
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
	if err := h.db.Preload("Admin").Joins("Admin").Where("refresh_tokens.token_hash = ? AND refresh_tokens.revoked = ? AND refresh_tokens.expires_at > ? AND Admin.is_active = ?",
		tokenHash, false, time.Now(), true).First(&refreshToken).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired refresh token",
		})
	}

	// Guard against orphaned tokens (admin deleted after token was issued)
	if refreshToken.Admin.ID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired refresh token",
		})
	}

	if !refreshToken.Admin.IsActive {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Admin account is deactivated",
		})
	}

	accessToken, err := h.tokenService.GenerateAccessTokenWithPermissions(
		refreshToken.Admin.ID,
		refreshToken.Admin.Username,
		refreshToken.Admin.IsSuperAdmin,
		refreshToken.Admin.MustChangePassword,
		refreshToken.Admin.Permissions,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate access token",
		})
	}

	if err := h.db.Model(&models.RefreshToken{}).
		Where("id = ?", refreshToken.ID).
		Update("revoked", true).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to revoke old token",
		})
	}

	newRefreshToken, err := h.tokenService.GenerateRefreshToken()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate refresh token",
		})
	}

	newHash := sha256.Sum256([]byte(newRefreshToken))
	newTokenHash := hex.EncodeToString(newHash[:])

	newRefreshTokenModel := models.RefreshToken{
		TokenHash: newTokenHash,
		AdminID:   refreshToken.AdminID,
		ExpiresAt: time.Now().Add(h.tokenService.GetRefreshTokenTTL()),
		UserAgent: c.Get("User-Agent"),
		IPAddress: c.IP(),
	}
	if err := h.db.Create(&newRefreshTokenModel).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to store new refresh token",
		})
	}

	return c.JSON(fiber.Map{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
		"expires_in":    int64(h.tokenService.GetAccessTokenTTL().Seconds()),
	})
}

// Logout handles admin logout
//
// @Summary      Logout
// @Description  Revoke the current refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body  RefreshRequest  true  "Refresh token to revoke"
// @Success      200   {object}  map[string]interface{}
// @Router       /auth/logout [post]
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

	// Blacklist the access token so it cannot be used after logout
	if authHeader := c.Get("Authorization"); len(authHeader) > 7 {
		accessToken := authHeader[7:] // strip "Bearer "
		h.tokenService.BlacklistAccessToken(accessToken)
	}

	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

// Me returns current admin info
//
// @Summary      Current admin
// @Description  Returns the profile of the currently authenticated admin
// @Tags         auth
// @Produce      json
// @Success      200  {object}  AdminInfo
// @Failure      401  {object}  map[string]interface{}
// @Router       /me [get]
// @Security     BearerAuth
func (h *AuthHandler) Me(c fiber.Ctx) error {
	adminID, ok := c.Locals("admin_id").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var admin models.Admin
	if err := h.db.First(&admin, adminID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Admin not found",
		})
	}

	return c.JSON(AdminInfo{
		ID:                 admin.ID,
		Username:           admin.Username,
		Email:              admin.Email,
		IsSuperAdmin:       admin.IsSuperAdmin,
		MustChangePassword: admin.MustChangePassword,
	})
}

// TOTPSetup generates a new TOTP secret and returns the provisioning URI + QR data.
// The secret is stored but TOTP is not enabled until TOTPVerify confirms it.
//
// @Summary      Setup TOTP
// @Description  Generate a new TOTP secret and provisioning URI. Call TOTPVerify after to activate.
// @Tags         auth
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /auth/totp/setup [post]
// @Security     BearerAuth
func (h *AuthHandler) TOTPSetup(c fiber.Ctx) error {
	adminID, ok := c.Locals("admin_id").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var admin models.Admin
	if err := h.db.First(&admin, adminID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Admin not found"})
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Isolate Panel",
		AccountName: admin.Username,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate TOTP secret"})
	}

	// Persist the secret but keep TOTPEnabled = false until verified
	if err := h.db.Model(&admin).Update("totp_secret", key.Secret()).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save TOTP secret"})
	}

	return c.JSON(fiber.Map{
		"secret":          key.Secret(),
		"provisioning_uri": key.URL(),
	})
}

// TOTPVerify confirms the TOTP code and enables 2FA for the admin account.
//
// @Summary      Verify and enable TOTP
// @Description  Confirm the TOTP code from the authenticator app and activate 2FA
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body  map[string]string  true  "TOTP code: {code}"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Router       /auth/totp/verify [post]
// @Security     BearerAuth
func (h *AuthHandler) TOTPVerify(c fiber.Ctx) error {
	adminID, ok := c.Locals("admin_id").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req struct {
		Code string `json:"code" validate:"required"`
	}
	if err := c.Bind().JSON(&req); err != nil || req.Code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "code is required"})
	}

	var admin models.Admin
	if err := h.db.First(&admin, adminID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Admin not found"})
	}
	if admin.TOTPSecret == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Run TOTP setup first"})
	}
	if !totp.Validate(req.Code, admin.TOTPSecret) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid TOTP code"})
	}

	if err := h.db.Model(&admin).Update("totp_enabled", true).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to enable TOTP"})
	}

	return c.JSON(fiber.Map{"message": "TOTP enabled successfully"})
}

// TOTPDisable disables TOTP after verifying the current password.
//
// @Summary      Disable TOTP
// @Description  Disable 2FA after verifying the admin password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body  map[string]string  true  "Admin password: {password}"
// @Success      200   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Router       /auth/totp/disable [post]
// @Security     BearerAuth
func (h *AuthHandler) TOTPDisable(c fiber.Ctx) error {
	adminID, ok := c.Locals("admin_id").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req struct {
		Password string `json:"password" validate:"required"`
	}
	if err := c.Bind().JSON(&req); err != nil || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "password is required"})
	}

	var admin models.Admin
	if err := h.db.First(&admin, adminID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Admin not found"})
	}

	valid, err := auth.VerifyPassword(req.Password, admin.PasswordHash)
	if err != nil || !valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid password"})
	}

	if err := h.db.Model(&admin).Updates(map[string]any{
		"totp_enabled": false,
		"totp_secret":  "",
	}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to disable TOTP"})
	}

	return c.JSON(fiber.Map{"message": "TOTP disabled successfully"})
}

// TOTPStatus returns whether TOTP is enabled for the current admin.
//
// @Summary      TOTP status
// @Description  Check if TOTP 2FA is enabled for the current admin account
// @Tags         auth
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /auth/totp/status [get]
// @Security     BearerAuth
func (h *AuthHandler) TOTPStatus(c fiber.Ctx) error {
	adminID, ok := c.Locals("admin_id").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	var admin models.Admin
	if err := h.db.First(&admin, adminID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Admin not found"})
	}
	return c.JSON(fiber.Map{
		"totp_enabled": admin.TOTPEnabled,
	})
}

// ChangePassword allows an admin to change their password
//
// @Summary      Change password
// @Description  Change the current admin's password. Requires current password verification.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body  ChangePasswordRequest  true  "Password change request"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Router       /auth/change-password [post]
// @Security     BearerAuth
// SessionLogin handles BFF session-based login for web SPA.
// Sets httpOnly cookie with session ID instead of returning tokens.
func (h *AuthHandler) SessionLogin(c fiber.Ctx) error {
	var req LoginRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	failedCount, _ := h.countFailedAttempts(c.IP(), 15*time.Minute)
	if failedCount >= 5 {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": "Too many failed login attempts. Try again later.",
		})
	}

	attempt := models.LoginAttempt{
		IPAddress: c.IP(),
		Username:  req.Username,
		Success:   false,
		UserAgent: c.Get("User-Agent"),
	}

	var admin models.Admin
	if err := h.db.Where("username = ? AND is_active = ?", req.Username, true).First(&admin).Error; err != nil {
		if dbErr := h.db.Create(&attempt).Error; dbErr != nil {
			logger.Log.Error().Err(dbErr).Msg("Failed to record login attempt")
		}
		h.checkFailedLoginAttempts(req.Username, c.IP())
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid username or password",
		})
	}

	valid, err := auth.VerifyPassword(req.Password, admin.PasswordHash)
	if err != nil || !valid {
		if dbErr := h.db.Create(&attempt).Error; dbErr != nil {
			logger.Log.Error().Err(dbErr).Msg("Failed to record login attempt")
		}
		h.checkFailedLoginAttempts(req.Username, c.IP())
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid username or password",
		})
	}

	if auth.NeedsRehash(req.Password, admin.PasswordHash) {
		if newHash, err := auth.HashPassword(req.Password); err == nil {
			if dbErr := h.db.Model(&admin).Update("password_hash", newHash).Error; dbErr != nil {
				logger.Log.Error().Err(dbErr).Msg("Failed to rehash password")
			}
		}
	}

	if admin.TOTPEnabled {
		if req.TotpCode == "" {
			return c.JSON(fiber.Map{"requires_totp": true})
		}
		if !totp.Validate(req.TotpCode, admin.TOTPSecret) {
			if dbErr := h.db.Create(&attempt).Error; dbErr != nil {
				logger.Log.Error().Err(dbErr).Msg("Failed to record login attempt")
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid TOTP code",
			})
		}
	}

	accessToken, err := h.tokenService.GenerateAccessTokenWithPermissions(admin.ID, admin.Username, admin.IsSuperAdmin, admin.MustChangePassword, admin.Permissions)
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

	hash := sha256.Sum256([]byte(refreshToken))
	tokenHash := hex.EncodeToString(hash[:])

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

	sessionID, err := h.sessionManager.CreateSession(
		admin.ID,
		admin.Username,
		admin.IsSuperAdmin,
		admin.MustChangePassword,
		accessToken,
		refreshToken,
		h.tokenService.GetAccessTokenTTL(),
		h.tokenService.GetRefreshTokenTTL(),
		c.Get("User-Agent"),
		c.IP(),
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create session",
		})
	}

	now := time.Now()
	admin.LastLoginAt = &now
	if dbErr := h.db.Save(&admin).Error; dbErr != nil {
		logger.Log.Error().Err(dbErr).Msg("Failed to update last login time")
	}

	attempt.Success = true
	if dbErr := h.db.Create(&attempt).Error; dbErr != nil {
		logger.Log.Error().Err(dbErr).Msg("Failed to record login attempt")
	}

	c.Cookie(&fiber.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(SessionCookieTTL.Seconds()),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})

	return c.JSON(LoginResponse{
		AccessToken:  "",
		RefreshToken: "",
		ExpiresIn:    int64(h.tokenService.GetAccessTokenTTL().Seconds()),
		Admin: AdminInfo{
			ID:           admin.ID,
			Username:     admin.Username,
			Email:        admin.Email,
			IsSuperAdmin: admin.IsSuperAdmin,
		},
		MustChangePassword: admin.MustChangePassword,
	})
}

// SessionLogout handles BFF session logout.
// Clears httpOnly cookie and removes server-side session.
func (h *AuthHandler) SessionLogout(c fiber.Ctx) error {
	sessionID := c.Cookies(SessionCookieName)
	if sessionID != "" {
		h.sessionManager.DeleteSession(sessionID)
	}

	c.Cookie(&fiber.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})

	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

// SessionRefresh handles transparent token refresh for BFF sessions.
// The client does not need to send a refresh token — it's stored server-side.
func (h *AuthHandler) SessionRefresh(c fiber.Ctx) error {
	sessionID := c.Cookies(SessionCookieName)
	if sessionID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No session",
		})
	}

	session := h.sessionManager.GetSession(sessionID)
	if session == nil {
		c.Cookie(&fiber.Cookie{
			Name:     SessionCookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HTTPOnly: true,
			Secure:   true,
			SameSite: "Strict",
		})
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Session expired",
		})
	}

	if time.Now().Before(session.AccessExpiry) {
		return c.JSON(fiber.Map{
			"expires_in": int64(h.tokenService.GetAccessTokenTTL().Seconds()),
		})
	}

	hash := sha256.Sum256([]byte(session.RefreshToken))
	tokenHash := hex.EncodeToString(hash[:])

	var refreshToken models.RefreshToken
	if err := h.db.Preload("Admin").Joins("Admin").Where("refresh_tokens.token_hash = ? AND refresh_tokens.revoked = ? AND refresh_tokens.expires_at > ? AND Admin.is_active = ?",
		tokenHash, false, time.Now(), true).First(&refreshToken).Error; err != nil {
		h.sessionManager.DeleteSession(sessionID)
		c.Cookie(&fiber.Cookie{
			Name:     SessionCookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HTTPOnly: true,
			Secure:   true,
			SameSite: "Strict",
		})
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired refresh token",
		})
	}

	if refreshToken.Admin.ID == 0 || !refreshToken.Admin.IsActive {
		h.sessionManager.DeleteSession(sessionID)
		c.ClearCookie(SessionCookieName)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Admin account is deactivated",
		})
	}

	accessToken, err := h.tokenService.GenerateAccessTokenWithPermissions(
		refreshToken.Admin.ID,
		refreshToken.Admin.Username,
		refreshToken.Admin.IsSuperAdmin,
		refreshToken.Admin.MustChangePassword,
		refreshToken.Admin.Permissions,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate access token",
		})
	}

	if err := h.db.Model(&models.RefreshToken{}).Where("id = ?", refreshToken.ID).Update("revoked", true).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to revoke old token",
		})
	}

	newRefreshToken, err := h.tokenService.GenerateRefreshToken()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate refresh token",
		})
	}

	newHash := sha256.Sum256([]byte(newRefreshToken))
	newTokenHash := hex.EncodeToString(newHash[:])

	newRefreshTokenModel := models.RefreshToken{
		TokenHash: newTokenHash,
		AdminID:   refreshToken.AdminID,
		ExpiresAt: time.Now().Add(h.tokenService.GetRefreshTokenTTL()),
		UserAgent: c.Get("User-Agent"),
		IPAddress: c.IP(),
	}
	if err := h.db.Create(&newRefreshTokenModel).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to store new refresh token",
		})
	}

	if !h.sessionManager.UpdateRefreshToken(sessionID, accessToken, newRefreshToken, h.tokenService.GetAccessTokenTTL(), h.tokenService.GetRefreshTokenTTL()) {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update session",
		})
	}

	return c.JSON(fiber.Map{
		"expires_in": int64(h.tokenService.GetAccessTokenTTL().Seconds()),
	})
}

func (h *AuthHandler) ChangePassword(c fiber.Ctx) error {
	adminID, ok := c.Locals("admin_id").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req ChangePasswordRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	var admin models.Admin
	if err := h.db.First(&admin, adminID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Admin not found",
		})
	}

	valid, err := auth.VerifyPassword(req.CurrentPassword, admin.PasswordHash)
	if err != nil || !valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid current password",
		})
	}

	newPasswordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to hash new password",
		})
	}

	if err := h.db.Model(&admin).Updates(map[string]any{
		"password_hash":        newPasswordHash,
		"must_change_password": false,
	}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update password",
		})
	}

	accessToken, err := h.tokenService.GenerateAccessTokenWithPermissions(admin.ID, admin.Username, admin.IsSuperAdmin, false, admin.Permissions)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate access token",
		})
	}

	return c.JSON(fiber.Map{
		"message":      "Password changed successfully",
		"access_token": accessToken,
	})
}

// countFailedAttempts counts failed login attempts from an IP within a time window
func (h *AuthHandler) countFailedAttempts(ip string, window time.Duration) (int64, error) {
	var count int64
	windowStart := time.Now().Add(-window)
	err := h.db.Model(&models.LoginAttempt{}).
		Where("ip_address = ? AND success = ? AND attempted_at > ?", ip, false, windowStart).
		Count(&count).Error
	return count, err
}

// checkFailedLoginAttempts checks for multiple failed login attempts and sends notification
func (h *AuthHandler) checkFailedLoginAttempts(username, ip string) {
	if h.notificationService == nil {
		return
	}

	// Count failed attempts from this IP in the last hour
	count, _ := h.countFailedAttempts(ip, 1*time.Hour)

	// Send notification when count reaches exactly 5 (not on every subsequent attempt)
	if count == 5 {
		h.notificationService.NotifyFailedLogin(ip, username, int(count))
	}
}

// WebAuthnRegisterBegin starts the WebAuthn registration process
//
// @Summary      Begin WebAuthn registration
// @Description  Initiates registration of a new FIDO2/WebAuthn credential (passkey)
// @Tags         auth
// @Produce      json
// @Success      200  {object}  protocol.CredentialCreation
// @Failure      401  {object}  map[string]interface{}
// @Router       /auth/webauthn/register/begin [post]
// @Security     BearerAuth
func (h *AuthHandler) WebAuthnRegisterBegin(c fiber.Ctx) error {
	adminID, ok := c.Locals("admin_id").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	if h.webAuthnService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "WebAuthn not configured",
		})
	}

	options, err := h.webAuthnService.BeginRegistration(adminID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to begin registration: " + err.Error(),
		})
	}

	return c.JSON(options)
}

// WebAuthnRegisterFinish completes the WebAuthn registration process
//
// @Summary      Finish WebAuthn registration
// @Description  Completes registration of a new FIDO2/WebAuthn credential
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body  protocol.ParsedCredentialCreationData  true  "Credential creation response"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Router       /auth/webauthn/register/finish [post]
// @Security     BearerAuth
func (h *AuthHandler) WebAuthnRegisterFinish(c fiber.Ctx) error {
	adminID, ok := c.Locals("admin_id").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	if h.webAuthnService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "WebAuthn not configured",
		})
	}

	var response protocol.ParsedCredentialCreationData
	if err := c.Bind().JSON(&response); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.webAuthnService.FinishRegistration(adminID, &response); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Registration failed: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "WebAuthn credential registered successfully",
	})
}

// WebAuthnAuthenticateBegin starts the WebAuthn authentication process
//
// @Summary      Begin WebAuthn authentication
// @Description  Initiates FIDO2/WebAuthn authentication (discoverable credential login)
// @Tags         auth
// @Produce      json
// @Success      200  {object}  protocol.CredentialAssertion
// @Router       /auth/webauthn/authenticate/begin [post]
func (h *AuthHandler) WebAuthnAuthenticateBegin(c fiber.Ctx) error {
	if h.webAuthnService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "WebAuthn not configured",
		})
	}

	options, err := h.webAuthnService.BeginAuthentication()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to begin authentication: " + err.Error(),
		})
	}

	return c.JSON(options)
}

// WebAuthnAuthenticateFinish completes the WebAuthn authentication process
//
// @Summary      Finish WebAuthn authentication
// @Description  Completes FIDO2/WebAuthn authentication and returns tokens on success
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body  protocol.ParsedCredentialAssertionData  true  "Credential assertion response"
// @Success      200   {object}  LoginResponse
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Router       /auth/webauthn/authenticate/finish [post]
func (h *AuthHandler) WebAuthnAuthenticateFinish(c fiber.Ctx) error {
	if h.webAuthnService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "WebAuthn not configured",
		})
	}

	var response protocol.ParsedCredentialAssertionData
	if err := c.Bind().JSON(&response); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	adminID, err := h.webAuthnService.FinishAuthentication(&response)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication failed: " + err.Error(),
		})
	}

	var admin models.Admin
	if err := h.db.First(&admin, adminID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve admin",
		})
	}

	accessToken, err := h.tokenService.GenerateAccessTokenWithPermissions(admin.ID, admin.Username, admin.IsSuperAdmin, admin.MustChangePassword, admin.Permissions)
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

	hash := sha256.Sum256([]byte(refreshToken))
	tokenHash := hex.EncodeToString(hash[:])

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

	now := time.Now()
	admin.LastLoginAt = &now
	if dbErr := h.db.Save(&admin).Error; dbErr != nil {
		logger.Log.Error().Err(dbErr).Msg("Failed to update last login time")
	}

	return c.JSON(LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(h.tokenService.GetAccessTokenTTL().Seconds()),
		Admin: AdminInfo{
			ID:           admin.ID,
			Username:     admin.Username,
			Email:        admin.Email,
			IsSuperAdmin: admin.IsSuperAdmin,
		},
		MustChangePassword: admin.MustChangePassword,
	})
}

// WebAuthnListCredentials returns all WebAuthn credentials for the current admin
//
// @Summary      List WebAuthn credentials
// @Description  Returns all registered FIDO2/WebAuthn credentials for the current admin
// @Tags         auth
// @Produce      json
// @Success      200  {array}   models.WebAuthnCredential
// @Failure      401  {object}  map[string]interface{}
// @Router       /auth/webauthn/credentials [get]
// @Security     BearerAuth
func (h *AuthHandler) WebAuthnListCredentials(c fiber.Ctx) error {
	adminID, ok := c.Locals("admin_id").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	if h.webAuthnService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "WebAuthn not configured",
		})
	}

	creds, err := h.webAuthnService.GetCredentialsForAdmin(adminID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve credentials",
		})
	}

	return c.JSON(creds)
}

// WebAuthnDeleteCredential removes a WebAuthn credential
//
// @Summary      Delete WebAuthn credential
// @Description  Removes a FIDO2/WebAuthn credential by ID
// @Tags         auth
// @Produce      json
// @Param        id   path      string  true  "Credential ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /auth/webauthn/credentials/{id} [delete]
// @Security     BearerAuth
func (h *AuthHandler) WebAuthnDeleteCredential(c fiber.Ctx) error {
	adminID, ok := c.Locals("admin_id").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	if h.webAuthnService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "WebAuthn not configured",
		})
	}

	credentialID := c.Params("id")
	if credentialID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "credential ID is required",
		})
	}

	if err := h.webAuthnService.DeleteCredential(adminID, credentialID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Credential not found",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Credential deleted successfully",
	})
}

// WebAuthnStatus returns WebAuthn status for the current admin
//
// @Summary      WebAuthn status
// @Description  Check if WebAuthn is enabled and configured for the current admin
// @Tags         auth
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /auth/webauthn/status [get]
// @Security     BearerAuth
func (h *AuthHandler) WebAuthnStatus(c fiber.Ctx) error {
	adminID, ok := c.Locals("admin_id").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	if h.webAuthnService == nil {
		return c.JSON(fiber.Map{
			"enabled":     false,
			"configured":  false,
			"credentials": 0,
		})
	}

	creds, _ := h.webAuthnService.GetCredentialsForAdmin(adminID)

	return c.JSON(fiber.Map{
		"enabled":     true,
		"configured":  len(creds) > 0,
		"credentials": len(creds),
	})
}
