package handlers

import (
	"fmt"
	"math/rand"
	"net/http"
	"project173/database"
	"project173/middleware"
	"project173/models"
	"project173/pkg/statemachine"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrderHandler struct{}

func NewOrderHandler() *OrderHandler {
	return &OrderHandler{}
}

type OrderCreateRequest struct {
	PropertyID   uint      `json:"property_id" binding:"required"`
	CheckInDate  time.Time `json:"check_in_date" binding:"required"`
	CheckOutDate time.Time `json:"check_out_date" binding:"required"`
}

type OrderTransitionRequest struct {
	Event string `json:"event" binding:"required"`
}

func generateOrderNo() string {
	now := time.Now().Format("20060102150405")
	random := rand.Intn(9000) + 1000
	return fmt.Sprintf("BN%s%d", now, random)
}

func (h *OrderHandler) Create(c *gin.Context) {
	tenantID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userRole, _ := middleware.GetUserRole(c)
	if userRole != string(models.RoleTenant) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only tenants can create orders"})
		return
	}
	var req OrderCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var property models.Property
	if err := database.DB.First(&property, req.PropertyID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "property not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if !property.IsOnline() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "property is not available (offline)"})
		return
	}
	if !req.CheckOutDate.After(req.CheckInDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "check_out_date must be after check_in_date"})
		return
	}
	days := int(req.CheckOutDate.Sub(req.CheckInDate).Hours() / 24)
	if days <= 0 {
		days = 1
	}
	totalAmount := float64(days) * property.PricePerDay
	order := models.Order{
		OrderNo:      generateOrderNo(),
		PropertyID:   req.PropertyID,
		TenantID:     tenantID,
		LandlordID:   property.LandlordID,
		CheckInDate:  req.CheckInDate,
		CheckOutDate: req.CheckOutDate,
		Days:         days,
		TotalAmount:  totalAmount,
		Status:       models.OrderUnpaid,
	}
	if err := order.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := database.DB.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create order"})
		return
	}
	c.JSON(http.StatusCreated, order)
}

func (h *OrderHandler) List(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userRole, _ := middleware.GetUserRole(c)
	status := c.Query("status")

	query := database.DB.Model(&models.Order{})
	switch userRole {
	case string(models.RoleTenant):
		query = query.Where("tenant_id = ?", userID)
	case string(models.RoleLandlord):
		query = query.Where("landlord_id = ?", userID)
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid user role"})
		return
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var orders []models.Order
	if err := query.Order("created_at desc").Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query orders"})
		return
	}
	c.JSON(http.StatusOK, orders)
}

func (h *OrderHandler) Get(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}
	var order models.Order
	if err := database.DB.First(&order, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	userRole, _ := middleware.GetUserRole(c)
	authorized := false
	switch models.UserRole(userRole) {
	case models.RoleTenant:
		authorized = order.TenantID == userID
	case models.RoleLandlord:
		authorized = order.LandlordID == userID
	}
	if !authorized {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not authorized to view this order"})
		return
	}
	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) Transition(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}
	var order models.Order
	if err := database.DB.First(&order, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	userRole, _ := middleware.GetUserRole(c)
	var req OrderTransitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	event := statemachine.OrderEvent(req.Event)
	switch event {
	case statemachine.EventPay, statemachine.EventCancel:
		if models.UserRole(userRole) != models.RoleTenant || order.TenantID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "only the tenant can pay or cancel the order"})
			return
		}
	case statemachine.EventCheckIn, statemachine.EventCheckOut:
		if models.UserRole(userRole) != models.RoleLandlord || order.LandlordID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "only the landlord can check in/out"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event"})
		return
	}
	newStatus, err := statemachine.Transition(order.Status, event)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	order.Status = newStatus
	if err := database.DB.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update order"})
		return
	}
	c.JSON(http.StatusOK, order)
}
