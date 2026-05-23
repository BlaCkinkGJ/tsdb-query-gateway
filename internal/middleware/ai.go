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

	"github.com/BlaCkinkGJ/tsdb-query-gateway/internal/config"
	"github.com/BlaCkinkGJ/tsdb-query-gateway/pkg/models"
	"github.com/BlaCkinkGJ/tsdb-query-gateway/internal/service"
	"github.com/gin-gonic/gin"
)

func init() {
	if err := Register("ai", func(mwConfig config.MiddlewareConfig, router *gin.RouterGroup) Middleware {
		var config struct {
			Endpoint string `json:"endpoint"`
			Timeout  int    `json:"timeout,omitempty"` // Example of adding configurable timeout
		}

		if len(mwConfig.Config) > 0 {
			if err := json.Unmarshal(mwConfig.Config, &config); err != nil {
				log.Fatalf("Fatal: AI Middleware config unmarshal error: %v", err)
			}
		}

		if config.Endpoint == "" {
			log.Fatalf("Fatal: AI Middleware enabled but 'endpoint' is missing in config")
		}

		timeout := 60 * time.Second
		if config.Timeout > 0 {
			timeout = time.Duration(config.Timeout) * time.Second
		}

		httpClient := &http.Client{
			Timeout: timeout,
		}

		return &aiMiddlewareFactory{
			aiEndpoint: config.Endpoint,
			httpClient: httpClient,
		}
	}); err != nil {
		log.Printf("Failed to register AI middleware: %v", err)
	}
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.aiEndpoint, bytes.NewReader(payloadBytes))
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
		bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 1024))
		if readErr != nil {
			return nil, fmt.Errorf("unexpected status code: %d, failed to read body: %w", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("AI endpoint error: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var modifiedResp models.PromResponse
	if err := json.NewDecoder(resp.Body).Decode(&modifiedResp); err != nil {
		return nil, fmt.Errorf("failed to decode AI response: %w", err)
	}

	return &modifiedResp, nil
}
