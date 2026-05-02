package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/example/prom-router/internal/models"
)

type PrometheusClient interface {
	Query(ctx context.Context, targetURL string, req models.PromQueryRequest) (*models.PromResponse, error)
	QueryRange(ctx context.Context, targetURL string, req models.PromQueryRangeRequest) (*models.PromResponse, error)
}

type prometheusClientImpl struct {
	httpClient *http.Client
}

func NewPrometheusClient() PrometheusClient {
	return &prometheusClientImpl{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *prometheusClientImpl) Query(ctx context.Context, targetURL string, req models.PromQueryRequest) (*models.PromResponse, error) {
	u, err := url.Parse(targetURL + "/api/v1/query")
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

	httpReq, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	return c.doRequest(httpReq)
}

func (c *prometheusClientImpl) QueryRange(ctx context.Context, targetURL string, req models.PromQueryRangeRequest) (*models.PromResponse, error) {
	u, err := url.Parse(targetURL + "/api/v1/query_range")
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

	httpReq, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var promResp models.PromResponse
	if err := json.Unmarshal(body, &promResp); err != nil {
		return nil, err
	}

	return &promResp, nil
}
