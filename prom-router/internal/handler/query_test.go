package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/models"
	"github.com/gin-gonic/gin"
)

type mockQueryService struct{}

func (s *mockQueryService) Query(ctx context.Context, req models.PromQueryRequest) (*models.PromResponse, error) {
	return &models.PromResponse{Status: "success"}, nil
}
func (s *mockQueryService) QueryRange(ctx context.Context, req models.PromQueryRangeRequest) (*models.PromResponse, error) {
	return &models.PromResponse{Status: "success"}, nil
}

func TestHandleQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &mockQueryService{}
	h := NewQueryHandler(mockSvc)
	router := gin.Default()
	api := router.Group("/api/v1")
	h.RegisterRoutes(api)

	// Test missing query
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/query", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for missing query, got %d", w.Code)
	}

	// Test successful query
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/v1/query?query=up", nil)
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w2.Code)
	}
}
