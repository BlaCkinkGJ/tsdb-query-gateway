package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

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
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i, endpoint := range s.promEndpoints {
		wg.Add(1)
		go func(idx int, targetURL string) {
			defer wg.Done()
			res, err := promClient.Query(ctx, targetURL, req)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return
			}
			results[idx] = res
		}(i, endpoint)
	}

	wg.Wait()

	if firstErr != nil {
		return nil, fmt.Errorf("error querying downstream prometheus: %w", firstErr)
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
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i, endpoint := range s.promEndpoints {
		wg.Add(1)
		go func(idx int, targetURL string) {
			defer wg.Done()
			res, err := promClient.QueryRange(ctx, targetURL, req)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return
			}
			results[idx] = res
		}(i, endpoint)
	}

	wg.Wait()

	if firstErr != nil {
		return nil, fmt.Errorf("error querying downstream prometheus: %w", firstErr)
	}

	return s.mergeResults(results), nil
}

// mergeResults merges the data from multiple Prometheus responses.
func (s *RouterService) mergeResults(results []*models.PromResponse) *models.PromResponse {
	if len(results) == 0 {
		return nil
	}

	var mergedResultType string
	var allMetrics []interface{}

	for _, res := range results {
		if res == nil || res.Status != "success" {
			continue
		}

		if mergedResultType == "" {
			mergedResultType = res.Data.ResultType
		}

		var metrics []interface{}
		if err := json.Unmarshal(res.Data.Result, &metrics); err == nil {
			allMetrics = append(allMetrics, metrics...)
		}
	}

	if mergedResultType == "" {
		return results[0] // Fallback
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
