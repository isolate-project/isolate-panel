package api

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// PaginationParams holds pagination query parameters
type PaginationParams struct {
	Page     int `query:"page" validate:"min=1"`
	PageSize int `query:"page_size" validate:"min=1,max=100"`
}

// PaginatedResponse holds paginated response data
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// GetPagination extracts pagination parameters from the request context
// Returns default values (page=1, page_size=50) if not provided
func GetPagination(c fiber.Ctx) PaginationParams {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "50"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return PaginationParams{
		Page:     page,
		PageSize: pageSize,
	}
}

// Paginate executes a paginated query on the given GORM model
// Returns a PaginatedResponse with the data and pagination metadata
func Paginate(db *gorm.DB, model interface{}, params PaginationParams) (*PaginatedResponse, error) {
	var total int64

	if err := db.Model(model).Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (params.Page - 1) * params.PageSize
	if err := db.Offset(offset).Limit(params.PageSize).Find(model).Error; err != nil {
		return nil, err
	}

	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	return &PaginatedResponse{
		Data:       model,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}