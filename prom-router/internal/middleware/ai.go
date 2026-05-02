package middleware

import (
	"context"
	"fmt"
	"log"

	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/client"
	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/config"
	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/models"
	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/service"
	"github.com/gin-gonic/gin"
)

func init() {
	Register("ai", func(cfg *config.Config, router *gin.RouterGroup) Middleware {
		if cfg.AIEndpoint == "" {
			log.Println("AI Middleware enabled but AIEndpoint is empty in config")
		}

		// Example of attaching a specific route for AI configuration if needed
		// router.GET("/ai/status", func(c *gin.Context) { c.JSON(200, gin.H{"status": "active"}) })

		aiClient := client.NewAIClient()
		return NewAIMiddleware(cfg.AIEndpoint, aiClient)
	})
}

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
