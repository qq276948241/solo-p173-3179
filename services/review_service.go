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
	ErrInvalidCursor        = errors.New("invalid cursor")
	ErrInvalidLimit         = errors.New("invalid limit")
)

const DefaultReviewLimit = 20
const MaxReviewLimit = 100

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

type ReviewListParams struct {
	PropertyID uint
	Cursor     uint
	Limit      int
}

type ReviewListResult struct {
	Reviews    []models.Review `json:"reviews"`
	AvgRating  float64         `json:"avg_rating"`
	Count      int64           `json:"count"`
	NextCursor uint            `json:"next_cursor"`
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

func (s *ReviewService) ListByProperty(params ReviewListParams) (*ReviewListResult, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = DefaultReviewLimit
	}
	if limit > MaxReviewLimit {
		limit = MaxReviewLimit
	}

	var totalCount int64
	if err := s.db.Model(&models.Review{}).
		Where("property_id = ?", params.PropertyID).
		Count(&totalCount).Error; err != nil {
		return nil, err
	}

	avgRating := 0.0
	if totalCount > 0 {
		type avgResult struct {
			AvgRating float64
		}
		var ar avgResult
		if err := s.db.Model(&models.Review{}).
			Where("property_id = ?", params.PropertyID).
			Select("AVG(rating) as avg_rating").
			Scan(&ar).Error; err != nil {
			return nil, err
		}
		avgRating = ar.AvgRating
	}

	var reviews []models.Review
	query := s.db.Where("property_id = ?", params.PropertyID).
		Order("id desc").
		Limit(limit + 1)

	if params.Cursor > 0 {
		query = query.Where("id < ?", params.Cursor)
	}

	if err := query.Find(&reviews).Error; err != nil {
		return nil, err
	}

	nextCursor := uint(0)
	if len(reviews) > limit {
		nextCursor = reviews[limit].ID
		reviews = reviews[:limit]
	}

	return &ReviewListResult{
		Reviews:    reviews,
		AvgRating:  avgRating,
		Count:      totalCount,
		NextCursor: nextCursor,
	}, nil
}
