package service_test

import (
	"errors"
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
)

type fakeStatsRepository struct {
	stats *repository.RecommendationStats
	err   error
}

func (f *fakeStatsRepository) RecommendationStats() (*repository.RecommendationStats, error) {
	return f.stats, f.err
}

func TestRecommendationAdminStatsComputesCTR(t *testing.T) {
	svc := service.NewRecommendationAdminService(
		&fakeRecommendationRunRepository{},
		&fakeStatsRepository{stats: &repository.RecommendationStats{
			Impressions: 200, Clicks: 50, Favorites: 10, Views: 80,
			RecommendationUsers: 30,
		}},
	)

	view, err := svc.Stats()
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if view.ClickThroughRate != 0.25 {
		t.Fatalf("CTR = %v, want 0.25", view.ClickThroughRate)
	}
	if view.RecommendationUsers != 30 {
		t.Fatalf("RecommendationUsers = %d, want 30", view.RecommendationUsers)
	}
}

func TestRecommendationAdminStatsZeroImpressions(t *testing.T) {
	svc := service.NewRecommendationAdminService(
		&fakeRecommendationRunRepository{},
		&fakeStatsRepository{stats: &repository.RecommendationStats{}},
	)

	view, err := svc.Stats()
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if view.ClickThroughRate != 0 {
		t.Fatalf("CTR = %v, want 0", view.ClickThroughRate)
	}
}

func TestRecommendationAdminStatsMapsError(t *testing.T) {
	svc := service.NewRecommendationAdminService(
		&fakeRecommendationRunRepository{},
		&fakeStatsRepository{err: errors.New("db error")},
	)

	_, err := svc.Stats()
	assertErrorCode(t, err, apperror.CodeInternal)
}
