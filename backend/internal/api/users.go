package api

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
	"github.com/vovk4morkovk4/isolate-panel/internal/services"
)

type UsersHandler struct {
	userService *services.UserService
}

func NewUsersHandler(userService *services.UserService) *UsersHandler {
	return &UsersHandler{
		userService: userService,
	}
}

// CreateUser creates a new user
func (h *UsersHandler) CreateUser(c fiber.Ctx) error {
	adminID := c.Locals("admin_id").(uint)

	var req services.CreateUserRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	user, err := h.userService.CreateUser(&req, adminID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(h.formatUserResponse(user))
}

// ListUsers lists all users
func (h *UsersHandler) ListUsers(c fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	users, total, err := h.userService.ListUsers(page, pageSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list users",
		})
	}

	userResponses := make([]services.UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = h.formatUserResponse(&user)
	}

	return c.JSON(fiber.Map{
		"users":     userResponses,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"pages":     (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// GetUser retrieves a specific user
func (h *UsersHandler) GetUser(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	user, err := h.userService.GetUser(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	return c.JSON(h.formatUserResponse(user))
}

// UpdateUser updates a user
func (h *UsersHandler) UpdateUser(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	var req services.UpdateUserRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	user, err := h.userService.UpdateUser(uint(id), &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(h.formatUserResponse(user))
}

// DeleteUser deletes a user
func (h *UsersHandler) DeleteUser(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	if err := h.userService.DeleteUser(uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "User deleted successfully",
	})
}

// RegenerateCredentials regenerates user credentials
func (h *UsersHandler) RegenerateCredentials(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	user, err := h.userService.RegenerateCredentials(uint(id))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Credentials regenerated successfully",
		"user":    h.formatUserResponse(user),
	})
}

// GetUserInbounds retrieves inbounds for a user
func (h *UsersHandler) GetUserInbounds(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	inbounds, err := h.userService.GetUserInbounds(uint(id))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get user inbounds",
		})
	}

	return c.JSON(fiber.Map{
		"inbounds": inbounds,
	})
}

// formatUserResponse formats a user model to response
func (h *UsersHandler) formatUserResponse(user *models.User) services.UserResponse {
	response := services.UserResponse{
		ID:                user.ID,
		Username:          user.Username,
		Email:             user.Email,
		UUID:              user.UUID,
		Password:          user.Password,
		Token:             user.Token,
		SubscriptionToken: user.SubscriptionToken,
		TrafficLimitBytes: user.TrafficLimitBytes,
		TrafficUsedBytes:  user.TrafficUsedBytes,
		IsActive:          user.IsActive,
		IsOnline:          user.IsOnline,
		CreatedAt:         user.CreatedAt.Format(time.RFC3339),
	}

	if user.ExpiryDate != nil {
		expiryStr := user.ExpiryDate.Format(time.RFC3339)
		response.ExpiryDate = &expiryStr
	}

	return response
}
