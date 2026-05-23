package middleware

import (
	"context"
	"testing"

	"github.com/BlaCkinkGJ/tsdb-query-gateway/pkg/models"
	"github.com/BlaCkinkGJ/tsdb-query-gateway/internal/service"
)

type dummyService struct {
	calls int
}

func (s *dummyService) Query(ctx context.Context, req models.PromQueryRequest) (*models.PromResponse, error) {
	s.calls++
	return &models.PromResponse{Status: "success"}, nil
}
func (s *dummyService) QueryRange(ctx context.Context, req models.PromQueryRangeRequest) (*models.PromResponse, error) {
	return nil, nil
}

type dummyMiddleware struct {
	next  service.QueryService
	calls *int
}

func (m *dummyMiddleware) Query(ctx context.Context, req models.PromQueryRequest) (*models.PromResponse, error) {
	*m.calls++
	return m.next.Query(ctx, req)
}
func (m *dummyMiddleware) QueryRange(ctx context.Context, req models.PromQueryRangeRequest) (*models.PromResponse, error) {
	return nil, nil
}

type dummyMiddlewareFactory struct {
	calls *int
}

func (f *dummyMiddlewareFactory) Wrap(next service.QueryService) service.QueryService {
	return &dummyMiddleware{
		next:  next,
		calls: f.calls,
	}
}

func TestMiddlewareChain(t *testing.T) {
	baseSvc := &dummyService{}
	mwCalls := 0

	mwFactory := &dummyMiddlewareFactory{calls: &mwCalls}

	chainedSvc := Chain(baseSvc, mwFactory)

	req := models.PromQueryRequest{}
	_, err := chainedSvc.Query(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mwCalls != 1 {
		t.Errorf("expected middleware to be called 1 time, got %d", mwCalls)
	}
	if baseSvc.calls != 1 {
		t.Errorf("expected base service to be called 1 time, got %d", baseSvc.calls)
	}
}
