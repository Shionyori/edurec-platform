package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"gorm.io/gorm"
)

type fakeRatingRepository struct {
	findResult            *model.Rating
	findErr               error
	lastFindUserID        uint
	lastFindResourceID    uint
	createErr             error
	lastCreated           *model.Rating
	updateErr             error
	lastUpdated           *model.Rating
	listResult            *repository.RatingListResult
	listErr               error
	lastQuery             repository.RatingListQuery
	average               float64
	averageErr            error
	lastAverageResourceID uint
}

func (f *fakeRatingRepository) FindByUserAndResource(userID uint, resourceID uint) (*model.Rating, error) {
	f.lastFindUserID = userID
	f.lastFindResourceID = resourceID
	if f.findErr != nil {
		return nil, f.findErr
	}
	return f.findResult, nil
}

func (f *fakeRatingRepository) Create(rating *model.Rating) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.lastCreated = rating
	return nil
}

func (f *fakeRatingRepository) Update(rating *model.Rating) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.lastUpdated = rating
	return nil
}

func (f *fakeRatingRepository) ListByResource(query repository.RatingListQuery) (*repository.RatingListResult, error) {
	f.lastQuery = query
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listResult, nil
}

func (f *fakeRatingRepository) AverageScore(resourceID uint) (float64, error) {
	f.lastAverageResourceID = resourceID
	if f.averageErr != nil {
		return 0, f.averageErr
	}
	return f.average, nil
}

func TestRatingUpsertCreatesNewRatingAndUpdatesAverage(t *testing.T) {
	ratings := &fakeRatingRepository{findErr: repository.ErrNotFound, average: 4.5}
	resources := &fakeResourceRepository{findResult: &model.Resource{Model: gorm.Model{ID: 9}}}
	svc := service.NewRatingService(ratings, resources)

	rating, err := svc.Upsert(context.Background(), 7, 9, 5, "很好")

	if err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if ratings.lastCreated == nil {
		t.Fatal("Upsert() did not create rating")
	}
	if ratings.lastCreated.UserID != 7 || ratings.lastCreated.ResourceID != 9 || ratings.lastCreated.Score != 5 {
		t.Fatalf("Upsert() rating fields are incorrect")
	}
	if rating.Comment != "很好" {
		t.Fatalf("Upsert() comment = %q, want 很好", rating.Comment)
	}
	if resources.lastUpdated == nil || resources.lastUpdated.AvgRating != 4.5 {
		t.Fatalf("Upsert() did not update resource average")
	}
}

func TestRatingUpsertUpdatesExistingRating(t *testing.T) {
	existing := &model.Rating{Model: gorm.Model{ID: 3}, UserID: 7, ResourceID: 9, Score: 2, Comment: "旧评论"}
	ratings := &fakeRatingRepository{findResult: existing, average: 4}
	resources := &fakeResourceRepository{findResult: &model.Resource{}}
	svc := service.NewRatingService(ratings, resources)

	rating, err := svc.Upsert(context.Background(), 7, 9, 4, "更新评论")

	if err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if ratings.lastUpdated == nil {
		t.Fatal("Upsert() did not update rating")
	}
	if rating.Score != 4 || rating.Comment != "更新评论" {
		t.Fatalf("Upsert() updated rating fields are incorrect")
	}
}

func TestRatingUpsertRejectsInvalidScore(t *testing.T) {
	svc := service.NewRatingService(&fakeRatingRepository{}, &fakeResourceRepository{})

	_, err := svc.Upsert(context.Background(), 7, 9, 6, "")
	assertErrorCode(t, err, apperror.CodeBadRequest)
}

func TestRatingUpsertMapsResourceNotFound(t *testing.T) {
	resources := &fakeResourceRepository{findErr: repository.ErrNotFound}
	svc := service.NewRatingService(&fakeRatingRepository{}, resources)

	_, err := svc.Upsert(context.Background(), 7, 9, 5, "")
	assertErrorCode(t, err, apperror.CodeNotFound)
}

func TestRatingListByResourcePassesQuery(t *testing.T) {
	ratings := &fakeRatingRepository{
		listResult: &repository.RatingListResult{
			Items: []model.Rating{{Score: 5}},
			Total: 1,
		},
	}
	resources := &fakeResourceRepository{findResult: &model.Resource{}}
	svc := service.NewRatingService(ratings, resources)

	result, err := svc.ListByResource(context.Background(), 9, 2, 10)

	if err != nil {
		t.Fatalf("ListByResource() error = %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("ListByResource() total = %d, want 1", result.Total)
	}
	if ratings.lastQuery.ResourceID != 9 || ratings.lastQuery.Page != 2 || ratings.lastQuery.PageSize != 10 {
		t.Fatalf("ListByResource() query was not passed through")
	}
}

func TestRatingListByResourceMapsRepositoryError(t *testing.T) {
	ratings := &fakeRatingRepository{listErr: errors.New("db error")}
	resources := &fakeResourceRepository{findResult: &model.Resource{}}
	svc := service.NewRatingService(ratings, resources)

	_, err := svc.ListByResource(context.Background(), 9, 1, 20)
	assertErrorCode(t, err, apperror.CodeInternal)
}
