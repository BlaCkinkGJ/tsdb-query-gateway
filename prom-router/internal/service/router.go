package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/example/prom-router/internal/client"
	"github.com/example/prom-router/internal/models"
)

type RouterService struct {
	promEndpoints []string
	promClient    client.PrometheusClient
}

func NewRouterService(promEndpoints []string, promClient client.PrometheusClient) *RouterService {
	return &RouterService{
		promEndpoints: promEndpoints,
		promClient:    promClient,
	}
}

// Query implements the QueryService interface.
func (s *RouterService) Query(ctx context.Context, req models.PromQueryRequest) (*models.PromResponse, error) {
	if len(s.promEndpoints) == 0 {
		return nil, fmt.Errorf("no prometheus endpoints configured")
	}

	results := make([]*models.PromResponse, len(s.promEndpoints))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i, endpoint := range s.promEndpoints {
		wg.Add(1)
		go func(idx int, targetURL string) {
			defer wg.Done()
			res, err := s.promClient.Query(ctx, targetURL, req)
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

	results := make([]*models.PromResponse, len(s.promEndpoints))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i, endpoint := range s.promEndpoints {
		wg.Add(1)
		go func(idx int, targetURL string) {
			defer wg.Done()
			res, err := s.promClient.QueryRange(ctx, targetURL, req)
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
