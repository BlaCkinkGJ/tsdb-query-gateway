package middleware

import (
	"context"
	"log"

	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/config"
	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/models"
	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/service"
	"github.com/gin-gonic/gin"
)

func init() {
	if err := Register("statistical", func(mwConfig config.MiddlewareConfig, router *gin.RouterGroup) Middleware {
		// Example of attaching a specific route for statistical configuration
		// router.GET("/statistical/config", func(c *gin.Context) { c.JSON(200, gin.H{"smoothing": true}) })

		return &statisticalMiddlewareFactory{}
	}); err != nil {
		log.Printf("Failed to register statistical middleware: %v", err)
	}
}

type statisticalMiddlewareFactory struct{}

func (f *statisticalMiddlewareFactory) Wrap(next service.QueryService) service.QueryService {
	return &statisticalMiddleware{
		next: next,
	}
}

type statisticalMiddleware struct {
	next service.QueryService
}

func (m *statisticalMiddleware) Query(ctx context.Context, req models.PromQueryRequest) (*models.PromResponse, error) {
	res, err := m.next.Query(ctx, req)
	if err != nil {
		return nil, err
	}

	if res != nil && res.Status == "success" {
		m.applyStatistics(res)
	}

	return res, nil
}

func (m *statisticalMiddleware) QueryRange(ctx context.Context, req models.PromQueryRangeRequest) (*models.PromResponse, error) {
	res, err := m.next.QueryRange(ctx, req)
	if err != nil {
		return nil, err
	}

	if res != nil && res.Status == "success" {
		m.applyStatistics(res)
	}

	return res, nil
}

func (m *statisticalMiddleware) applyStatistics(res *models.PromResponse) {
	// TODO: Dummy logic for statistical modifications.
	// In a real application, you would deserialize res.Data.Result,
	// apply statistical smoothing, anomaly detection, or predictions,
	// and then reserialize it.
}
