package service

import (
	"context"
	"errors"
	"strings"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)

type RatingService struct {
	ratings   repository.RatingRepository
	resources repository.ResourceRepository
}

func NewRatingService(ratings repository.RatingRepository, resources repository.ResourceRepository) *RatingService {
	return &RatingService{ratings: ratings, resources: resources}
}

func (s *RatingService) Upsert(ctx context.Context, userID uint, resourceID uint, score uint8, comment string) (*model.Rating, error) {
	comment = strings.TrimSpace(comment)
	if userID == 0 || resourceID == 0 || score < 1 || score > 5 {
		return nil, apperror.BadRequest("请求参数错误")
	}

	if err := s.ensureResourceExists(ctx, resourceID); err != nil {
		return nil, err
	}

	rating, err := s.ratings.FindByUserAndResource(userID, resourceID)
	if errors.Is(err, repository.ErrNotFound) {
		rating = &model.Rating{
			UserID:     userID,
			ResourceID: resourceID,
			Score:      score,
			Comment:    comment,
		}
		if err := s.ratings.Create(rating); err != nil {
			if !errors.Is(err, repository.ErrDuplicate) {
				return nil, apperror.Internal(err)
			}
			rating, err = s.ratings.FindByUserAndResource(userID, resourceID)
			if err != nil {
				return nil, apperror.Internal(err)
			}
			rating.Score = score
			rating.Comment = comment
			if err := s.ratings.Update(rating); err != nil {
				return nil, apperror.Internal(err)
			}
		}
	} else if err != nil {
		return nil, apperror.Internal(err)
	} else {
		rating.Score = score
		rating.Comment = comment
		if err := s.ratings.Update(rating); err != nil {
			return nil, apperror.Internal(err)
		}
	}

	if err := s.updateResourceAverage(ctx, resourceID); err != nil {
		return nil, err
	}
	return rating, nil
}

func (s *RatingService) ListByResource(ctx context.Context, resourceID uint, page int, pageSize int) (*repository.RatingListResult, error) {
	if resourceID == 0 {
		return nil, apperror.BadRequest("请求参数错误")
	}
	if err := s.ensureResourceExists(ctx, resourceID); err != nil {
		return nil, err
	}

	result, err := s.ratings.ListByResource(repository.RatingListQuery{
		ResourceID: resourceID,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return result, nil
}

func (s *RatingService) ensureResourceExists(ctx context.Context, resourceID uint) error {
	_, err := s.resources.FindByID(resourceID)
	if errors.Is(err, repository.ErrNotFound) {
		return apperror.NotFound("资源不存在")
	}
	if err != nil {
		return apperror.Internal(err)
	}
	return nil
}

func (s *RatingService) updateResourceAverage(ctx context.Context, resourceID uint) error {
	average, err := s.ratings.AverageScore(resourceID)
	if err != nil {
		return apperror.Internal(err)
	}

	resource, err := s.resources.FindByID(resourceID)
	if errors.Is(err, repository.ErrNotFound) {
		return apperror.NotFound("资源不存在")
	}
	if err != nil {
		return apperror.Internal(err)
	}

	resource.AvgRating = float32(average)
	if err := s.resources.Update(resource); err != nil {
		return apperror.Internal(err)
	}
	return nil
}
