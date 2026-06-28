package handlers

import (
	"net/http"
	"project173/database"
	"project173/middleware"
	"project173/models"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ReviewHandler struct{}

func NewReviewHandler() *ReviewHandler {
	return &ReviewHandler{}
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
	var order models.Order
	if err := database.DB.First(&order, req.OrderID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if order.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not the tenant of this order"})
		return
	}
	if order.Status != models.OrderCheckedOut {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only checked-out orders can be reviewed"})
		return
	}
	var existing models.Review
	err := database.DB.Where("order_id = ?", req.OrderID).First(&existing).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "this order has already been reviewed"})
		return
	}
	if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	review := models.Review{
		OrderID:    req.OrderID,
		PropertyID: order.PropertyID,
		TenantID:   tenantID,
		Rating:     req.Rating,
		Comment:    req.Comment,
	}
	if err := review.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := database.DB.Create(&review).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create review"})
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
	var reviews []models.Review
	if err := database.DB.Where("property_id = ?", uint(propertyID)).
		Order("created_at desc").
		Find(&reviews).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query reviews"})
		return
	}
	type avgResult struct {
		AvgRating float64 `json:"avg_rating"`
		Count     int64   `json:"count"`
	}
	var result avgResult
	database.DB.Model(&models.Review{}).
		Where("property_id = ?", uint(propertyID)).
		Select("COALESCE(AVG(rating), 0) as avg_rating, COUNT(*) as count").
		Scan(&result)

	c.JSON(http.StatusOK, gin.H{
		"reviews":    reviews,
		"avg_rating": result.AvgRating,
		"count":      result.Count,
	})
}
