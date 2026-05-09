package service

import (
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/client"
	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/discovery"
	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/models"
)

type RouterService struct {
	promEndpoints []string
	registry      discovery.Registry
}

func NewRouterService(promEndpoints []string, registry discovery.Registry) *RouterService {
	return &RouterService{
		promEndpoints: promEndpoints,
		registry:      registry,
	}
}

// Query implements the QueryService interface.
func (s *RouterService) Query(ctx context.Context, req models.PromQueryRequest) (*models.PromResponse, error) {
	if len(s.promEndpoints) == 0 {
		return nil, fmt.Errorf("no prometheus endpoints configured")
	}

	clientObj, err := s.registry.Get("PrometheusClient")
	if err != nil {
		return nil, err
	}
	promClient, ok := clientObj.(client.PrometheusClient)
	if !ok {
		return nil, fmt.Errorf("PrometheusClient is not of expected interface type")
	}

	results := make([]*models.PromResponse, len(s.promEndpoints))
	g, gCtx := errgroup.WithContext(ctx)

	for i, endpoint := range s.promEndpoints {
		idx, targetURL := i, endpoint // capture variables for closure
		g.Go(func() error {
			res, err := promClient.Query(gCtx, targetURL, req)
			if err != nil {
				return err
			}
			results[idx] = res
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("error querying downstream prometheus: %w", err)
	}

	return s.mergeResults(results), nil
}

// QueryRange implements the QueryService interface.
func (s *RouterService) QueryRange(ctx context.Context, req models.PromQueryRangeRequest) (*models.PromResponse, error) {
	if len(s.promEndpoints) == 0 {
		return nil, fmt.Errorf("no prometheus endpoints configured")
	}

	clientObj, err := s.registry.Get("PrometheusClient")
	if err != nil {
		return nil, err
	}
	promClient, ok := clientObj.(client.PrometheusClient)
	if !ok {
		return nil, fmt.Errorf("PrometheusClient is not of expected interface type")
	}

	results := make([]*models.PromResponse, len(s.promEndpoints))
	g, gCtx := errgroup.WithContext(ctx)

	for i, endpoint := range s.promEndpoints {
		idx, targetURL := i, endpoint // capture variables for closure
		g.Go(func() error {
			res, err := promClient.QueryRange(gCtx, targetURL, req)
			if err != nil {
				return err
			}
			results[idx] = res
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("error querying downstream prometheus: %w", err)
	}

	return s.mergeResults(results), nil
}

// mergeResults merges the data from multiple Prometheus responses.
func (s *RouterService) mergeResults(results []*models.PromResponse) *models.PromResponse {
	if len(results) == 0 {
		return nil
	}

	var mergedResultType string
	var allMetrics []json.RawMessage

	for _, res := range results {
		if res == nil || res.Status != "success" || len(res.Data.Result) == 0 {
			continue
		}

		if mergedResultType == "" {
			mergedResultType = res.Data.ResultType
		}

		var metrics []json.RawMessage
		if err := json.Unmarshal(res.Data.Result, &metrics); err == nil {
			allMetrics = append(allMetrics, metrics...)
		} else {
			// If it's a single object rather than an array, just append it
			allMetrics = append(allMetrics, res.Data.Result)
		}
	}

	if mergedResultType == "" {
		// Fallback to the first non-nil result if any
		for _, r := range results {
			if r != nil {
				return r
			}
		}
		return nil
	}

	mergedRaw, _ := json.Marshal(allMetrics)

	return &models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: mergedResultType,
			Result:     mergedRaw,
		},
	}
}
