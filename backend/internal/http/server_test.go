package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"smart-cost-optimizer/backend/internal/models"
)

type stubActionService struct{}

func (stubActionService) List() []models.ActionRecord {
	return []models.ActionRecord{}
}

func (stubActionService) HibernateCluster(context.Context, string) (models.ActionRecord, error) {
	return models.ActionRecord{}, nil
}

func (stubActionService) WakeCluster(context.Context, string) (models.ActionRecord, error) {
	return models.ActionRecord{}, nil
}

func (stubActionService) ScaleNodePool(context.Context, string, string, int64, int64) (models.ActionRecord, error) {
	return models.ActionRecord{}, nil
}

func (stubActionService) MoveWorkload(context.Context, string, string, string, string) (models.ActionRecord, error) {
	return models.ActionRecord{}, nil
}

func TestRecommendationsEndpointReturnsEmptyArray(t *testing.T) {
	server := &Server{
		actions: stubActionService{},
	}
	server.snapshot = models.InventorySnapshot{
		Recommendations: []models.Recommendation{},
		Summary:         models.SavingsSummary{},
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations", nil)
	recorder := httptest.NewRecorder()

	server.handleRecommendations(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	body := strings.TrimSpace(recorder.Body.String())
	if body != "[]" {
		t.Fatalf("expected empty array response, got %q", body)
	}

	var recommendations []models.Recommendation
	if err := json.Unmarshal(recorder.Body.Bytes(), &recommendations); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if recommendations == nil {
		t.Fatalf("expected empty slice, got nil")
	}
}
