package api

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/isolate-project/isolate-panel/internal/auth"
	"github.com/isolate-project/isolate-panel/internal/middleware"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services"
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
//
// @Summary      Create user
// @Description  Create a new proxy user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body  services.CreateUserRequest  true  "User data"
// @Success      201   {object}  services.UserResponse
// @Failure      400   {object}  map[string]interface{}
// @Router       /users [post]
// @Security     BearerAuth
func (h *UsersHandler) CreateUser(c fiber.Ctx) error {
	adminID, ok := c.Locals("admin_id").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	req, err := middleware.BindAndValidate[services.CreateUserRequest](c)
	if err != nil {
		return err
	}

	user, err := h.userService.CreateUser(&req, adminID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(h.formatUserResponseWithCredentials(user))
}

// ListUsers lists all users
//
// @Summary      List users
// @Description  Returns paginated list of all proxy users
// @Tags         users
// @Produce      json
// @Param        page       query  int  false  "Page number"       default(1)
// @Param        page_size  query  int  false  "Items per page"    default(20)
// @Success      200        {object}  map[string]interface{}
// @Router       /users [get]
// @Security     BearerAuth
func (h *UsersHandler) ListUsers(c fiber.Ctx) error {
	params := GetPagination(c)
	search := c.Query("search")
	status := c.Query("status")

	users, total, err := h.userService.ListUsers(params.Page, params.PageSize, search, status)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list users",
		})
	}

	userResponses := make([]services.UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = h.formatUserResponse(&user)
	}

	totalPages := (total + int64(params.PageSize) - 1) / int64(params.PageSize)

	return c.JSON(fiber.Map{
		"success":   true,
		"users":     userResponses,
		"total":     total,
		"page":      params.Page,
		"page_size": params.PageSize,
		"pages":     totalPages,
	})
}

// GetUser retrieves a specific user
//
// @Summary      Get user
// @Description  Returns a single user by ID
// @Tags         users
// @Produce      json
// @Param        id   path  int  true  "User ID"
// @Success      200  {object}  services.UserResponse
// @Failure      404  {object}  map[string]interface{}
// @Router       /users/{id} [get]
// @Security     BearerAuth
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
//
// @Summary      Update user
// @Description  Update user fields (all fields optional)
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id    path  int                          true  "User ID"
// @Param        body  body  services.UpdateUserRequest   true  "Fields to update"
// @Success      200   {object}  services.UserResponse
// @Failure      404   {object}  map[string]interface{}
// @Router       /users/{id} [put]
// @Security     BearerAuth
func (h *UsersHandler) UpdateUser(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	req, err := middleware.BindAndValidate[services.UpdateUserRequest](c)
	if err != nil {
		return err
	}

	user, err := h.userService.UpdateUser(uint(id), &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(h.formatUserResponse(user))
}

// DeleteUser deletes a user
//
// @Summary      Delete user
// @Description  Permanently delete a user and all associated data
// @Tags         users
// @Produce      json
// @Param        id   path  int  true  "User ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /users/{id} [delete]
// @Security     BearerAuth
func (h *UsersHandler) DeleteUser(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	if err := h.userService.DeleteUser(uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"message": "User deleted successfully",
	})
}

// RegenerateCredentials regenerates user credentials
//
// @Summary      Regenerate credentials
// @Description  Generate a new UUID, password, and subscription token for the user
// @Tags         users
// @Produce      json
// @Param        id   path  int  true  "User ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /users/{id}/regenerate [post]
// @Security     BearerAuth
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
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Credentials regenerated successfully",
		"user":    h.formatUserResponseWithCredentials(user),
	})
}

// GetUserInbounds retrieves inbounds for a user
//
// @Summary      Get user inbounds
// @Description  Returns all inbounds assigned to a specific user
// @Tags         users
// @Produce      json
// @Param        id   path  int  true  "User ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /users/{id}/inbounds [get]
// @Security     BearerAuth
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

	return c.JSON(inbounds)
}

// formatUserResponse formats a user model to response (without sensitive credentials)
func (h *UsersHandler) formatUserResponse(user *models.User) services.UserResponse {
	response := services.UserResponse{
		ID:                user.ID,
		Username:          user.Username,
		Email:             user.Email,
		UUID:              user.UUID,
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

// formatUserResponseWithCredentials includes password and token (for create/regenerate only)
func (h *UsersHandler) formatUserResponseWithCredentials(user *models.User) services.CreateUserResponse {
	response := h.formatUserResponse(user)
	response.Token = user.Token

	decryptedPassword, err := auth.DecryptCredential(user.Password)
	if err != nil {
		return services.CreateUserResponse{
			UserResponse: response,
			Password:     "",
		}
	}

	return services.CreateUserResponse{
		UserResponse: response,
		Password:     decryptedPassword,
	}
}
