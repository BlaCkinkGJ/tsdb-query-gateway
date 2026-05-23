package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/BlaCkinkGJ/tsdb-query-gateway/pkg/models"
	"github.com/gin-gonic/gin"
)

type mockQueryService struct{}

func (s *mockQueryService) Query(ctx context.Context, req models.PromQueryRequest) (*models.PromResponse, error) {
	return &models.PromResponse{Status: "success"}, nil
}
func (s *mockQueryService) QueryRange(ctx context.Context, req models.PromQueryRangeRequest) (*models.PromResponse, error) {
	return &models.PromResponse{Status: "success"}, nil
}

type mockQueryServiceError struct{}

func (s *mockQueryServiceError) Query(ctx context.Context, req models.PromQueryRequest) (*models.PromResponse, error) {
	return nil, context.Canceled
}
func (s *mockQueryServiceError) QueryRange(ctx context.Context, req models.PromQueryRangeRequest) (*models.PromResponse, error) {
	return nil, context.Canceled
}

func TestHandleQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := &mockQueryService{}
	h := NewQueryHandler(mockService)

	router := gin.Default()
	api := router.Group("/api/v1")
	h.RegisterRoutes(api)

	// Test missing query
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/query", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for missing query, got %d", w.Code)
	}

	// Test successful query
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/api/v1/query?query=up", nil)
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w2.Code)
	}
}

func TestHandleQuery_ErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewQueryHandler(&mockQueryServiceError{})

	router := gin.Default()
	api := router.Group("/api/v1")
	h.RegisterRoutes(api)

	// Query endpoint should return 500 with generic error, not leak internal details
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/query?query=up", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 Internal Server Error, got %d", w.Code)
	}

	body := w.Body.String()
	if strings.Contains(body, "context canceled") {
		t.Error("error response leaked internal error details")
	}
	if !strings.Contains(body, "internal server error") {
		t.Errorf("expected generic error message, got: %s", body)
	}
}

func TestHandleQueryRange_ErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewQueryHandler(&mockQueryServiceError{})

	router := gin.Default()
	api := router.Group("/api/v1")
	h.RegisterRoutes(api)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/query_range?query=up&start=0&end=1&step=1", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 Internal Server Error, got %d", w.Code)
	}

	body := w.Body.String()
	if strings.Contains(body, "context canceled") {
		t.Error("error response leaked internal error details")
	}
}
