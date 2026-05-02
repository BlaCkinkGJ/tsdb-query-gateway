package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/example/prom-router/internal/client"
	"github.com/example/prom-router/internal/models"
)

type RouterService interface {
	Query(ctx context.Context, req models.PromQueryRequest) (*models.PromResponse, error)
	QueryRange(ctx context.Context, req models.PromQueryRangeRequest) (*models.PromResponse, error)
}

type routerServiceImpl struct {
	promEndpoints []string
	aiEndpoint    string
	promClient    client.PrometheusClient
	aiClient      client.AIClient
}

func NewRouterService(promEndpoints []string, aiEndpoint string, promClient client.PrometheusClient, aiClient client.AIClient) RouterService {
	return &routerServiceImpl{
		promEndpoints: promEndpoints,
		aiEndpoint:    aiEndpoint,
		promClient:    promClient,
		aiClient:      aiClient,
	}
}

func (s *routerServiceImpl) Query(ctx context.Context, req models.PromQueryRequest) (*models.PromResponse, error) {
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
		// Just a simple error handling: return the first error encountered.
		// In a production-ready system, we might want to return partial results or log the error and continue.
		return nil, fmt.Errorf("error querying downstream prometheus: %w", firstErr)
	}

	mergedResult := s.mergeResults(results)

	if s.aiEndpoint != "" && mergedResult != nil && mergedResult.Status == "success" {
		aiResult, err := s.aiClient.Process(ctx, s.aiEndpoint, mergedResult)
		if err != nil {
			// Fallback to original result if AI fails, or return error depending on requirements.
			// Let's return error for strictness.
			return nil, fmt.Errorf("error processing with AI middleware: %w", err)
		}
		return aiResult, nil
	}

	return mergedResult, nil
}

func (s *routerServiceImpl) QueryRange(ctx context.Context, req models.PromQueryRangeRequest) (*models.PromResponse, error) {
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

	mergedResult := s.mergeResults(results)

	if s.aiEndpoint != "" && mergedResult != nil && mergedResult.Status == "success" {
		aiResult, err := s.aiClient.Process(ctx, s.aiEndpoint, mergedResult)
		if err != nil {
			return nil, fmt.Errorf("error processing with AI middleware: %w", err)
		}
		return aiResult, nil
	}

	return mergedResult, nil
}

// mergeResults merges the data from multiple Prometheus responses.
// This is a simplified merge logic. Real Prometheus merging can be complex depending on ResultType (vector, matrix, etc).
func (s *routerServiceImpl) mergeResults(results []*models.PromResponse) *models.PromResponse {
	if len(results) == 0 {
		return nil
	}

	// We assume all successful queries return the same ResultType
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
		return results[0] // Fallback if no success found or data could not be merged
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
