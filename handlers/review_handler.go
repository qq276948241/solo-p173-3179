package handlers

import (
	"errors"
	"net/http"
	"project173/middleware"
	"project173/models"
	"project173/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ReviewHandler struct {
	reviewService *services.ReviewService
}

func NewReviewHandler(reviewService *services.ReviewService) *ReviewHandler {
	return &ReviewHandler{reviewService: reviewService}
}

type ReviewCreateRequest struct {
	OrderID uint   `json:"order_id" binding:"required"`
	Rating  int    `json:"rating" binding:"required"`
	Comment string `json:"comment"`
}

func (h *ReviewHandler) Create(c *gin.Context) {
	tenantID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userRole, _ := middleware.GetUserRole(c)
	if userRole != string(models.RoleTenant) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only tenants can create reviews"})
		return
	}

	var req ReviewCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	review, err := h.reviewService.CreateReview(services.ReviewCreateParams{
		OrderID:  req.OrderID,
		TenantID: tenantID,
		Rating:   req.Rating,
		Comment:  req.Comment,
	})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrOrderNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		case errors.Is(err, services.ErrNotOrderTenant):
			c.JSON(http.StatusForbidden, gin.H{"error": "you are not the tenant of this order"})
		case errors.Is(err, services.ErrOrderNotCheckedOut):
			c.JSON(http.StatusBadRequest, gin.H{"error": "only checked-out orders can be reviewed"})
		case errors.Is(err, services.ErrAlreadyReviewed):
			c.JSON(http.StatusConflict, gin.H{"error": "this order has already been reviewed"})
		case errors.Is(err, services.ErrInvalidRating):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create review"})
		}
		return
	}

	c.JSON(http.StatusCreated, review)
}

func (h *ReviewHandler) ListByProperty(c *gin.Context) {
	propertyID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid property id"})
		return
	}

	cursor := uint(0)
	cursorStr := c.Query("cursor")
	if cursorStr != "" {
		parsed, err := strconv.ParseUint(cursorStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cursor"})
			return
		}
		cursor = uint(parsed)
	}

	limit := 0
	limitStr := c.Query("limit")
	if limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
			return
		}
		limit = parsed
	}

	result, err := h.reviewService.ListByProperty(services.ReviewListParams{
		PropertyID: uint(propertyID),
		Cursor:     cursor,
		Limit:      limit,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query reviews"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reviews":      result.Reviews,
		"avg_rating":   result.AvgRating,
		"count":        result.Count,
		"next_cursor":  result.NextCursor,
	})
}
