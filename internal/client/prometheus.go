package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/BlaCkinkGJ/tsdb-query-gateway/pkg/models"
)

type PrometheusClient interface {
	Query(ctx context.Context, targetURL string, req models.PromQueryRequest) (*models.PromResponse, error)
	QueryRange(ctx context.Context, targetURL string, req models.PromQueryRangeRequest) (*models.PromResponse, error)
}

type prometheusClientImpl struct {
	httpClient *http.Client
}

func NewPrometheusClient(timeout time.Duration) PrometheusClient {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &prometheusClientImpl{
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *prometheusClientImpl) Query(ctx context.Context, targetURL string, req models.PromQueryRequest) (*models.PromResponse, error) {
	rawURL, err := url.JoinPath(targetURL, "/api/v1/query")
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("query", req.Query)
	if req.Time != "" {
		q.Set("time", req.Time)
	}
	if req.Timeout != "" {
		q.Set("timeout", req.Timeout)
	}
	u.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	return c.doRequest(httpReq)
}

func (c *prometheusClientImpl) QueryRange(ctx context.Context, targetURL string, req models.PromQueryRangeRequest) (*models.PromResponse, error) {
	rawURL, err := url.JoinPath(targetURL, "/api/v1/query_range")
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("query", req.Query)
	q.Set("start", req.Start)
	q.Set("end", req.End)
	q.Set("step", req.Step)
	if req.Timeout != "" {
		q.Set("timeout", req.Timeout)
	}
	u.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	return c.doRequest(httpReq)
}

func (c *prometheusClientImpl) doRequest(req *http.Request) (*models.PromResponse, error) {
	resp, err := c.httpClient.Do(req)
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
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var promResp models.PromResponse
	if err := json.NewDecoder(resp.Body).Decode(&promResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &promResp, nil
}
