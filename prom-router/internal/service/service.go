package service

import (
	"context"

	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/models"
)

// QueryService defines the core interface for handling Prometheus queries.
// Both the base router and middlewares implement this interface to form a daisy chain.
type QueryService interface {
	Query(ctx context.Context, req models.PromQueryRequest) (*models.PromResponse, error)
	QueryRange(ctx context.Context, req models.PromQueryRangeRequest) (*models.PromResponse, error)
}
