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

type PropertyHandler struct{}

func NewPropertyHandler() *PropertyHandler {
	return &PropertyHandler{}
}

type PropertyCreateRequest struct {
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description"`
	City        string  `json:"city" binding:"required"`
	Address     string  `json:"address" binding:"required"`
	PricePerDay float64 `json:"price_per_day" binding:"required"`
}

type PropertyUpdateRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	City        string   `json:"city"`
	Address     string   `json:"address"`
	PricePerDay *float64 `json:"price_per_day"`
}

type PropertyStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

func (h *PropertyHandler) Create(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req PropertyCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	prop := models.Property{
		LandlordID:  userID,
		Title:       req.Title,
		Description: req.Description,
		City:        req.City,
		Address:     req.Address,
		PricePerDay: req.PricePerDay,
		Status:      models.PropertyOffline,
	}
	if err := prop.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := database.DB.Create(&prop).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create property"})
		return
	}
	c.JSON(http.StatusCreated, prop)
}

func (h *PropertyHandler) List(c *gin.Context) {
	city := c.Query("city")
	minPriceStr := c.Query("min_price")
	maxPriceStr := c.Query("max_price")
	status := c.DefaultQuery("status", string(models.PropertyOnline))

	query := database.DB.Model(&models.Property{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if city != "" {
		query = query.Where("city = ?", city)
	}
	if minPriceStr != "" {
		minPrice, err := strconv.ParseFloat(minPriceStr, 64)
		if err == nil {
			query = query.Where("price_per_day >= ?", minPrice)
		}
	}
	if maxPriceStr != "" {
		maxPrice, err := strconv.ParseFloat(maxPriceStr, 64)
		if err == nil {
			query = query.Where("price_per_day <= ?", maxPrice)
		}
	}
	var props []models.Property
	if err := query.Find(&props).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query properties"})
		return
	}
	c.JSON(http.StatusOK, props)
}

func (h *PropertyHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid property id"})
		return
	}
	var prop models.Property
	if err := database.DB.First(&prop, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "property not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	c.JSON(http.StatusOK, prop)
}

func (h *PropertyHandler) Update(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid property id"})
		return
	}
	var prop models.Property
	if err := database.DB.First(&prop, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "property not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if prop.LandlordID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not the owner of this property"})
		return
	}
	var req PropertyUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Title != "" {
		prop.Title = req.Title
	}
	if req.Description != "" {
		prop.Description = req.Description
	}
	if req.City != "" {
		prop.City = req.City
	}
	if req.Address != "" {
		prop.Address = req.Address
	}
	if req.PricePerDay != nil {
		prop.PricePerDay = *req.PricePerDay
	}
	if err := prop.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := database.DB.Save(&prop).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update property"})
		return
	}
	c.JSON(http.StatusOK, prop)
}

func (h *PropertyHandler) SetStatus(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid property id"})
		return
	}
	var prop models.Property
	if err := database.DB.First(&prop, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "property not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if prop.LandlordID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not the owner of this property"})
		return
	}
	var req PropertyStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	newStatus := models.PropertyStatus(req.Status)
	if newStatus != models.PropertyOnline && newStatus != models.PropertyOffline {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status, must be online or offline"})
		return
	}
	prop.Status = newStatus
	if err := database.DB.Save(&prop).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update status"})
		return
	}
	c.JSON(http.StatusOK, prop)
}

func (h *PropertyHandler) Delete(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid property id"})
		return
	}
	var prop models.Property
	if err := database.DB.First(&prop, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "property not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if prop.LandlordID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not the owner of this property"})
		return
	}
	if err := database.DB.Delete(&prop).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete property"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "property deleted"})
}
