package middleware

import (
	"context"
	"fmt"

	"github.com/example/prom-router/internal/client"
	"github.com/example/prom-router/internal/models"
	"github.com/example/prom-router/internal/service"
)

type aiMiddleware struct {
	next       service.QueryService
	aiEndpoint string
	aiClient   client.AIClient
}

// NewAIMiddleware creates a middleware that intercepts query results and processes them via an AI endpoint.
func NewAIMiddleware(aiEndpoint string, aiClient client.AIClient) Middleware {
	return func(next service.QueryService) service.QueryService {
		return &aiMiddleware{
			next:       next,
			aiEndpoint: aiEndpoint,
			aiClient:   aiClient,
		}
	}
}

func (m *aiMiddleware) Query(ctx context.Context, req models.PromQueryRequest) (*models.PromResponse, error) {
	// 1. Call the next node in the chain
	res, err := m.next.Query(ctx, req)
	if err != nil {
		return nil, err
	}

	// 2. Process with AI middleware if successful
	if m.aiEndpoint != "" && res != nil && res.Status == "success" {
		aiResult, err := m.aiClient.Process(ctx, m.aiEndpoint, res)
		if err != nil {
			return nil, fmt.Errorf("error processing with AI middleware: %w", err)
		}
		return aiResult, nil
	}

	return res, nil
}

func (m *aiMiddleware) QueryRange(ctx context.Context, req models.PromQueryRangeRequest) (*models.PromResponse, error) {
	// 1. Call the next node in the chain
	res, err := m.next.QueryRange(ctx, req)
	if err != nil {
		return nil, err
	}

	// 2. Process with AI middleware if successful
	if m.aiEndpoint != "" && res != nil && res.Status == "success" {
		aiResult, err := m.aiClient.Process(ctx, m.aiEndpoint, res)
		if err != nil {
			return nil, fmt.Errorf("error processing with AI middleware: %w", err)
		}
		return aiResult, nil
	}

	return res, nil
}
