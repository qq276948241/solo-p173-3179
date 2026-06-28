package services

import (
	"errors"
	"project173/models"

	"gorm.io/gorm"
)

var (
	ErrOrderNotFound        = errors.New("order not found")
	ErrNotOrderTenant       = errors.New("you are not the tenant of this order")
	ErrOrderNotCheckedOut   = errors.New("only checked-out orders can be reviewed")
	ErrAlreadyReviewed      = errors.New("this order has already been reviewed")
	ErrInvalidRating        = errors.New("rating must be between 1 and 5")
	ErrPropertyNotFound     = errors.New("property not found")
)

type ReviewService struct {
	db *gorm.DB
}

func NewReviewService(db *gorm.DB) *ReviewService {
	return &ReviewService{db: db}
}

type ReviewCreateParams struct {
	OrderID  uint
	TenantID uint
	Rating   int
	Comment  string
}

type ReviewListResult struct {
	Reviews   []models.Review `json:"reviews"`
	AvgRating float64         `json:"avg_rating"`
	Count     int64           `json:"count"`
}

func (s *ReviewService) CreateReview(params ReviewCreateParams) (*models.Review, error) {
	var order models.Order
	if err := s.db.First(&order, params.OrderID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}

	if order.TenantID != params.TenantID {
		return nil, ErrNotOrderTenant
	}

	if order.Status != models.OrderCheckedOut {
		return nil, ErrOrderNotCheckedOut
	}

	var existing models.Review
	err := s.db.Where("order_id = ?", params.OrderID).First(&existing).Error
	if err == nil {
		return nil, ErrAlreadyReviewed
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if params.Rating < 1 || params.Rating > 5 {
		return nil, ErrInvalidRating
	}

	review := &models.Review{
		OrderID:    params.OrderID,
		PropertyID: order.PropertyID,
		TenantID:   params.TenantID,
		Rating:     params.Rating,
		Comment:    params.Comment,
	}

	if err := review.Validate(); err != nil {
		return nil, err
	}

	if err := s.db.Create(review).Error; err != nil {
		return nil, err
	}

	return review, nil
}

func (s *ReviewService) ListByProperty(propertyID uint) (*ReviewListResult, error) {
	var reviews []models.Review
	if err := s.db.Where("property_id = ?", propertyID).
		Order("created_at desc").
		Find(&reviews).Error; err != nil {
		return nil, err
	}

	type avgResult struct {
		AvgRating float64
		Count     int64
	}
	var result avgResult
	if err := s.db.Model(&models.Review{}).
		Where("property_id = ?", propertyID).
		Select("COALESCE(AVG(rating), 0) as avg_rating, COUNT(*) as count").
		Scan(&result).Error; err != nil {
		return nil, err
	}

	return &ReviewListResult{
		Reviews:   reviews,
		AvgRating: result.AvgRating,
		Count:     result.Count,
	}, nil
}
