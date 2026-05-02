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
	Register("statistical", func(cfg *config.Config, router *gin.RouterGroup) Middleware {
		// Example of attaching a specific route for statistical configuration
		// router.GET("/statistical/config", func(c *gin.Context) { c.JSON(200, gin.H{"smoothing": true}) })

		return NewStatisticalMiddleware()
	})
}

type statisticalMiddleware struct {
	next service.QueryService
}

// NewStatisticalMiddleware creates a middleware that applies statistical techniques
// to restructure or predict query results.
func NewStatisticalMiddleware() Middleware {
	return func(next service.QueryService) service.QueryService {
		return &statisticalMiddleware{
			next: next,
		}
	}
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
	// Dummy logic for statistical modifications.
	// In a real application, you would deserialize res.Data.Result,
	// apply statistical smoothing, anomaly detection, or predictions,
	// and then reserialize it.

	// E.g., log.Println("Applying statistical techniques to response...")
	_ = log.Printf // just to prevent unused import error if we comment it out
}
