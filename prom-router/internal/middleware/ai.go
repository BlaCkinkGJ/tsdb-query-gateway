package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/discovery"
	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/models"
	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/service"
	"github.com/gin-gonic/gin"
)

func init() {
	Register("ai", func(mwConfig map[string]interface{}, router *gin.RouterGroup, reg discovery.Registry) Middleware {
		var endpoint string
		if ep, ok := mwConfig["endpoint"].(string); ok {
			endpoint = ep
		} else {
			log.Println("AI Middleware enabled but 'endpoint' is missing in config")
		}

		httpClient := &http.Client{
			Timeout: 60 * time.Second,
		}

		return &aiMiddlewareFactory{
			aiEndpoint: endpoint,
			httpClient: httpClient,
		}
	})
}

type aiMiddlewareFactory struct {
	aiEndpoint string
	httpClient *http.Client
}

func (f *aiMiddlewareFactory) Wrap(next service.QueryService) service.QueryService {
	return &aiMiddleware{
		next:       next,
		aiEndpoint: f.aiEndpoint,
		httpClient: f.httpClient,
	}
}

type aiMiddleware struct {
	next       service.QueryService
	aiEndpoint string
	httpClient *http.Client
}

func (m *aiMiddleware) Query(ctx context.Context, req models.PromQueryRequest) (*models.PromResponse, error) {
	res, err := m.next.Query(ctx, req)
	if err != nil {
		return nil, err
	}

	if m.aiEndpoint != "" && res != nil && res.Status == "success" {
		return m.processAI(ctx, res)
	}

	return res, nil
}

func (m *aiMiddleware) QueryRange(ctx context.Context, req models.PromQueryRangeRequest) (*models.PromResponse, error) {
	res, err := m.next.QueryRange(ctx, req)
	if err != nil {
		return nil, err
	}

	if m.aiEndpoint != "" && res != nil && res.Status == "success" {
		return m.processAI(ctx, res)
	}

	return res, nil
}

func (m *aiMiddleware) processAI(ctx context.Context, data *models.PromResponse) (*models.PromResponse, error) {
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", m.aiEndpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Read up to 1024 bytes to avoid huge error strings.
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("AI endpoint error: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var modifiedResp models.PromResponse
	if err := json.NewDecoder(resp.Body).Decode(&modifiedResp); err != nil {
		return nil, fmt.Errorf("failed to decode AI response: %w", err)
	}

	return &modifiedResp, nil
}
