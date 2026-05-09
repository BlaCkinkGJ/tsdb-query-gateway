package service

import (
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/client"
	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/models"
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
	g, gCtx := errgroup.WithContext(ctx)

	for i, endpoint := range s.promEndpoints {
		idx, targetURL := i, endpoint // capture variables for closure
		g.Go(func() error {
			res, err := s.promClient.Query(gCtx, targetURL, req)
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

	return s.mergeResults(results)
}

// QueryRange implements the QueryService interface.
func (s *RouterService) QueryRange(ctx context.Context, req models.PromQueryRangeRequest) (*models.PromResponse, error) {
	if len(s.promEndpoints) == 0 {
		return nil, fmt.Errorf("no prometheus endpoints configured")
	}

	results := make([]*models.PromResponse, len(s.promEndpoints))
	g, gCtx := errgroup.WithContext(ctx)

	for i, endpoint := range s.promEndpoints {
		idx, targetURL := i, endpoint // capture variables for closure
		g.Go(func() error {
			res, err := s.promClient.QueryRange(gCtx, targetURL, req)
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

	return s.mergeResults(results)
}

// mergeResults merges the data from multiple Prometheus responses.
func (s *RouterService) mergeResults(results []*models.PromResponse) (*models.PromResponse, error) {
	if len(results) == 0 {
		return nil, nil
	}

	var mergedResultType string
	var allResults []json.RawMessage
	seenSeries := make(map[string]bool)

	for _, res := range results {
		if res == nil || res.Status != "success" || len(res.Data.Result) == 0 {
			continue
		}

		if mergedResultType == "" {
			mergedResultType = res.Data.ResultType
		} else if mergedResultType != res.Data.ResultType {
			return nil, fmt.Errorf("conflicting ResultType: %s and %s", mergedResultType, res.Data.ResultType)
		}

		// Scalar and String types represent fixed-size results (not mergeable arrays).
		// We return the first successful result for these types.
		if mergedResultType == "scalar" || mergedResultType == "string" {
			return res, nil
		}

		var partial []json.RawMessage
		if err := json.Unmarshal(res.Data.Result, &partial); err != nil {
			return nil, fmt.Errorf("failed to unmarshal result part: %w", err)
		}

		// Simple de-duplication for vector/matrix results based on the raw JSON content of the result item.
		// In a production environment, this should ideally decode the "metric" labels for better accuracy.
		for _, item := range partial {
			itemStr := string(item)
			if !seenSeries[itemStr] {
				allResults = append(allResults, item)
				seenSeries[itemStr] = true
			}
		}
	}

	if mergedResultType == "" {
		// Fallback to the first non-nil result if any
		for _, r := range results {
			if r != nil {
				return r, nil
			}
		}
		return nil, nil
	}

	mergedBytes, err := json.Marshal(allResults)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged results: %w", err)
	}

	return &models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: mergedResultType,
			Result:     json.RawMessage(mergedBytes),
		},
	}, nil
}
